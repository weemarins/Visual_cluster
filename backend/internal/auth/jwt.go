package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/example/vkube-topology/backend/internal/config"
)

// Claims personalizados de JWT.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken gera um JWT para o usuário autenticado.
func GenerateToken(username, role string, cfg *config.Config) (string, time.Time, error) {
	exp := time.Now().Add(time.Duration(cfg.JWTExpMinutes) * time.Minute)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseToken valida e retorna os claims de um token JWT.
func ParseToken(tokenStr string, cfg *config.Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

