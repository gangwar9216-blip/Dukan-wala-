package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	Secret []byte
}

type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func (m Manager) Issue(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("empty user id")
	}

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	).SignedString(m.Secret)
}

func (m Manager) Parse(tokenString string) (string, error) {
	var claims Claims

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("invalid signing method")
			}
			return m.Secret, nil
		},
	)

	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	if claims.UserID != "" {
		return claims.UserID, nil
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	return "", errors.New("user id missing from token")
}
