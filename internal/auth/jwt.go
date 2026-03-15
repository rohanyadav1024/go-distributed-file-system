// Package auth provides JWT utilities and gRPC auth middleware.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims defines DFS-specific claims carried in issued JWT tokens.
type Claims struct {
	Subject string `json:"sub"`
	jwt.RegisteredClaims
}

// GenerateToken signs a JWT for the given subject with the provided TTL.
func GenerateToken(secret string, subject string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		Subject: subject,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken parses and validates a JWT signed with the shared secret.
func ValidateToken(secret string, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
