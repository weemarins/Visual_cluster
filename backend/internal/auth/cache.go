package auth

import (
	"sync"
	"time"
)

// CachedLogin armazena credenciais em cache para validação rápida
type CachedLogin struct {
	Username string
	Role     string
	ExpireAt time.Time
}

// LoginCache é um cache em memória com TTL para logins
type LoginCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedLogin
}

var loginCache = &LoginCache{
	entries: make(map[string]*CachedLogin),
}

// CacheTTL define por quanto tempo o login fica em cache (em minutos)
const CacheTTL = 30 * time.Minute

// SetCachedLogin armazena um login em cache
func SetCachedLogin(username string, role string) {
	loginCache.mu.Lock()
	defer loginCache.mu.Unlock()

	loginCache.entries[username] = &CachedLogin{
		Username: username,
		Role:     role,
		ExpireAt: time.Now().Add(CacheTTL),
	}
}

// GetCachedLogin recupera um login do cache se ainda for válido
func GetCachedLogin(username string) *CachedLogin {
	loginCache.mu.RLock()
	defer loginCache.mu.RUnlock()

	cached, exists := loginCache.entries[username]
	if !exists {
		return nil
	}

	// Verifica se expirou
	if time.Now().After(cached.ExpireAt) {
		// Remove entrada expirada
		go func() {
			loginCache.mu.Lock()
			defer loginCache.mu.Unlock()
			delete(loginCache.entries, username)
		}()
		return nil
	}

	return cached
}

// InvalidateCachedLogin remove um login do cache (útil ao trocar senha)
func InvalidateCachedLogin(username string) {
	loginCache.mu.Lock()
	defer loginCache.mu.Unlock()

	delete(loginCache.entries, username)
}

// CleanupExpiredLogins remove logins expirados (deve ser chamado periodicamente)
func CleanupExpiredLogins() {
	loginCache.mu.Lock()
	defer loginCache.mu.Unlock()

	now := time.Now()
	for username, cached := range loginCache.entries {
		if now.After(cached.ExpireAt) {
			delete(loginCache.entries, username)
		}
	}
}
