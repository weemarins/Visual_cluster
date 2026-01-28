package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/vkube-topology/backend/internal/auth"
	"github.com/example/vkube-topology/backend/internal/config"
)

// RegisterRoutes registra todas as rotas /api/v1.
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	api := r.Group("/api/v1")

	// Auth
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", loginHandler(cfg))
		authGroup.GET("/me", auth.AuthMiddleware(cfg), meHandler())
	}

	// Clusters CRUD
	clusterGroup := api.Group("/clusters")
	clusterGroup.Use(auth.AuthMiddleware(cfg))
	{
		clusterGroup.GET("", listClustersHandler(cfg))
		clusterGroup.POST("", auth.RequireRole("admin"), createClusterHandler(cfg))
		clusterGroup.GET("/:id", getClusterHandler(cfg))
		clusterGroup.PUT("/:id", auth.RequireRole("admin"), updateClusterHandler(cfg))
		clusterGroup.DELETE("/:id", auth.RequireRole("admin"), deleteClusterHandler(cfg))
	}

	// Topologia
	topologyGroup := api.Group("/topology")
	topologyGroup.Use(auth.AuthMiddleware(cfg))
	{
		topologyGroup.GET("/:clusterID", topologyHandler(cfg))
	}

	// Healthcheck simples
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

