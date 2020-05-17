package auth

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// RoleAdmin represents admin role
// RoleUser represents user role
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// Claims represents the authorization claims transmitted via a JWT
type Claims struct {
	Roles []string
	jwt.StandardClaims
}

// NewClaims constructs a Claims value for the identified user
func NewClaims(subject string, roles []string, now time.Time, expires time.Duration) *Claims {
	c := &Claims{
		Roles: roles,
		StandardClaims: jwt.StandardClaims{
			Subject:   subject,
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(expires).Unix(),
		},
	}

	return c
}
