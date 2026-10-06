package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Manager struct{ Secret []byte }

func (m Manager) Issue(userID string) (string, error) {
	c := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(24 * time.Hour).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.Secret)
}
func (m Manager) Parse(token string) (string, error) {
	t, e := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("invalid signing method")
		}
		return m.Secret, nil
	})
	if e != nil || !t.Valid {
		return "", errors.New("invalid token")
	}
	s, e := t.Claims.GetSubject()
	return s, e
}
