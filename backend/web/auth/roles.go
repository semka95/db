package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// RoleAdmin represents admin role
// RoleUser represents user role
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// Claims represents the authorization claims transmitted via a JWT
type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewClaims constructs a Claims value for the identified user
func NewClaims(subject string, roles []string, now time.Time, expires time.Duration) *Claims {
	c := &Claims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expires)),
		},
	}

	return c
}

// HasRole returns true if the claims has at least one of the provided roles.
func (c *Claims) HasRole(roles ...string) bool {
	for _, has := range c.Roles {
		for _, want := range roles {
			if has == want {
				return true
			}
		}
	}
	return false
}
