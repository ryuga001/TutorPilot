package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     string `json:"uid"`
	Email      string `json:"email"`
	CustomerID int    `json:"cid"`
	Role       string `json:"role"`
	jwt.RegisteredClaims
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

// Generate signs an access token for a user with their tenant's secret.
func (m *Manager) Generate(secret []byte, userID, email string, customerID int, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	claims := Claims{
		UserID:     userID,
		Email:      email,
		CustomerID: customerID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
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
	return claims, nil
}
