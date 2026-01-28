package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/example/vkube-topology/backend/internal/config"
	"github.com/example/vkube-topology/backend/internal/models"
)

var (
	DB *gorm.DB
)

// InitPostgres inicializa a conexão com PostgreSQL.
func InitPostgres(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	log.Println("conectado ao PostgreSQL")
	return nil
}

// AutoMigrate executa as migrações automáticas dos modelos principais.
func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Cluster{},
	)
}

// Close fecha a conexão com o banco (usado em testes / shutdown).
func Close() {
	if DB == nil {
		return
	}
	sqlDB, err := DB.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

