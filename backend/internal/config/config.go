package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config agrega todas as configurações da aplicação.
type Config struct {
	AppPort        string
	JWTSecret      string
	JWTExpMinutes  int
	AESKey         []byte
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	LDAPURL        string
	LDAPBaseDN     string
	LDAPBindDN     string
	LDAPBindPass   string
	PollInterval   time.Duration
	MaxClusters    int
}

// LoadEnv tenta carregar variáveis de ambiente de um arquivo .env (modo dev).
func LoadEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		return godotenv.Load(".env")
	}
	return nil
}

// New cria uma nova instância de Config baseada em variáveis de ambiente.
func New() *Config {
	return &Config{
		AppPort:       getEnv("APP_PORT", "8080"),
		JWTSecret:     getEnv("APP_JWT_SECRET", "change-me-secret"),
		JWTExpMinutes: getEnvInt("APP_JWT_EXP_MINUTES", 60),
		AESKey:        []byte(getEnv("APP_AES_KEY", "change-me-32-bytes-key-change-me")),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "vkube"),
		DBPassword:    getEnv("DB_PASSWORD", "vkube"),
		DBName:        getEnv("DB_NAME", "vkube"),
		LDAPURL:       getEnv("LDAP_URL", "ldap://ldap.example.com:389"),
		LDAPBaseDN:    getEnv("LDAP_BASE_DN", "dc=example,dc=com"),
		LDAPBindDN:    getEnv("LDAP_BIND_DN", "cn=admin,dc=example,dc=com"),
		LDAPBindPass:  getEnv("LDAP_BIND_PASSWORD", "admin"),
		PollInterval:  time.Duration(getEnvInt("POLL_INTERVAL_SECONDS", 15)) * time.Second,
		MaxClusters:   getEnvInt("MAX_CLUSTERS_PER_USER", 20),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var val int
		_, err := fmt.Sscanf(v, "%d", &val)
		if err == nil {
			return val
		}
	}
	return def
}

