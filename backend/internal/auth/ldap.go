package auth

import (
	"fmt"

	ldap "github.com/go-ldap/ldap/v3"

	"github.com/example/vkube-topology/backend/internal/config"
)

// LDAPAuthenticate autentica o usuário no LDAP e retorna o DN e o displayName.
func LDAPAuthenticate(username, password string, cfg *config.Config) (string, string, error) {
	l, err := ldap.DialURL(cfg.LDAPURL)
	if err != nil {
		return "", "", err
	}
	defer l.Close()

	// Primeiro bind técnico
	if err := l.Bind(cfg.LDAPBindDN, cfg.LDAPBindPass); err != nil {
		return "", "", fmt.Errorf("erro bind técnico: %w", err)
	}

	// Busca usuário pelo uid
	searchRequest := ldap.NewSearchRequest(
		cfg.LDAPBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)),
		[]string{"dn", "cn", "displayName"},
		nil,
	)
	sr, err := l.Search(searchRequest)
	if err != nil {
		return "", "", fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if len(sr.Entries) != 1 {
		return "", "", fmt.Errorf("usuário não encontrado ou múltiplos resultados")
	}

	entry := sr.Entries[0]
	userDN := entry.DN
	displayName := entry.GetAttributeValue("displayName")
	if displayName == "" {
		displayName = entry.GetAttributeValue("cn")
	}

	// Bind como o próprio usuário para validar senha
	if err := l.Bind(userDN, password); err != nil {
		return "", "", fmt.Errorf("credenciais inválidas: %w", err)
	}

	return userDN, displayName, nil
}

