package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/vkube-topology/backend/internal/config"
)

// AuthMiddleware valida o JWT presente no header Authorization: Bearer <token>.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token ausente"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "formato de token inválido"})
			return
		}
		claims, err := ParseToken(parts[1], cfg)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
			return
		}
		c.Set("user", claims)
		c.Next()
	}
}

// RequireRole garante que o usuário possui um dos papéis esperados.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := map[string]struct{}{}
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		val, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "sem contexto de usuário"})
			return
		}
		claims, ok := val.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "claims inválidos"})
			return
		}
		if _, ok := allowedSet[claims.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acesso negado"})
			return
		}
		c.Next()
	}
}

