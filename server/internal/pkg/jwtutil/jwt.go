package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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

	SubjectType string `json:"styp"`
	SubjectID   int    `json:"sid,omitempty"`

	MustChangePassword bool `json:"pwr,omitempty"`

	jwt.RegisteredClaims
}

type Identity struct {
	UserID             string
	Email              string
	CustomerID         int
	Role               string
	SubjectType        string
	SubjectID          int
	MustChangePassword bool
}

type SecretFunc func(customerID int) ([]byte, error)

type Manager struct {
	accessTTL time.Duration
}

func New(accessTTL time.Duration) *Manager {
	return &Manager{accessTTL: accessTTL}
}

func (m *Manager) AccessTTL() time.Duration { return m.accessTTL }

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

	if claims.SubjectType == "" {
		claims.SubjectType = SubjectAdmin
	}
	return claims, nil
}
