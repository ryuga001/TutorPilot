package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Subject types. A principal is always exactly one of these; tutors and students
// additionally carry the id of the directory record they speak for.
const (
	SubjectAdmin   = "admin"
	SubjectTutor   = "tutor"
	SubjectStudent = "student"
)

type Claims struct {
	UserID     string `json:"uid"`
	Email      string `json:"email"`
	CustomerID int    `json:"cid"`
	Role       string `json:"role"`

	// SubjectType decides where the principal may sign in (only admins on the
	// root host) and, with SubjectID, which rows they can reach.
	SubjectType string `json:"styp"`
	SubjectID   int    `json:"sid,omitempty"`

	// MustChangePassword gates every route but the password-change endpoints, so
	// an invited member cannot use their temporary credentials for anything else.
	MustChangePassword bool `json:"pwr,omitempty"`

	jwt.RegisteredClaims
}

// Identity is everything needed to mint an access token for a principal.
type Identity struct {
	UserID             string
	Email              string
	CustomerID         int
	Role               string
	SubjectType        string
	SubjectID          int
	MustChangePassword bool
}

// SecretFunc resolves the HS256 signing secret for the tenant that a token
// claims to belong to. It receives the customer id parsed from the token.
type SecretFunc func(customerID int) ([]byte, error)

// Manager issues and parses access tokens. Each token is signed with the
// per-tenant secret from customers.jwt_secret, so the manager itself holds no
// secret — only the access-token TTL.
type Manager struct {
	accessTTL time.Duration
}

func New(accessTTL time.Duration) *Manager {
	return &Manager{accessTTL: accessTTL}
}

func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

// Generate signs an access token for a principal with their tenant's secret.
func (m *Manager) Generate(secret []byte, id Identity) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	claims := Claims{
		UserID:             id.UserID,
		Email:              id.Email,
		CustomerID:         id.CustomerID,
		Role:               id.Role,
		SubjectType:        id.SubjectType,
		SubjectID:          id.SubjectID,
		MustChangePassword: id.MustChangePassword,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   id.UserID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	return signed, exp, err
}

// Parse validates a token. The customer id is read from the (as-yet unverified)
// claims to look up that tenant's secret, then the signature is verified.
func (m *Manager) Parse(tokenStr string, secretFor SecretFunc) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		if claims.CustomerID == 0 {
			return nil, fmt.Errorf("token missing customer id")
		}
		return secretFor(claims.CustomerID)
	})
	if err != nil {
		return nil, err
	}
	// Tokens minted before subject types existed are admin tokens.
	if claims.SubjectType == "" {
		claims.SubjectType = SubjectAdmin
	}
	return claims, nil
}
