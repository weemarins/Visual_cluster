package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/vkube-topology/backend/internal/api"
	"github.com/example/vkube-topology/backend/internal/auth"
	"github.com/example/vkube-topology/backend/internal/config"
	"github.com/example/vkube-topology/backend/internal/db"
)

func main() {
	// Carrega variáveis de ambiente (.env em dev, env vars em prod)
	if err := config.LoadEnv(); err != nil {
		log.Printf("warn: erro ao carregar .env: %v", err)
	}

	// Inicializa config
	cfg := config.New()

	// Conecta no banco
	if err := db.InitPostgres(cfg); err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer db.Close()

	// Migra modelos
	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("erro ao migrar modelos: %v", err)
	}

	r := gin.Default()

	// Registra rotas da API
	api.RegisterRoutes(r, cfg)

	// Inicia routine de limpeza de cache de login expirado
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			auth.CleanupExpiredLogins()
		}
	}()

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	if err := r.Run(addr); err != nil {
		log.Printf("erro ao subir servidor: %v", err)
		os.Exit(1)
	}
}

