package models

import "time"

// User representa um usuário autenticado via LDAP com informações de RBAC básico.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:128" json:"username"`
	DisplayName string  `gorm:"size:256" json:"displayName"`
	Role      string    `gorm:"size:32" json:"role"` // admin, viewer
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Cluster representa um cluster Kubernetes com kubeconfig criptografado.
type Cluster struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"size:128;not null" json:"name"`
	Description      string    `gorm:"size:512" json:"description"`
	OwnerUsername    string    `gorm:"size:128;index" json:"ownerUsername"`
	EncryptedKubeconfig []byte `gorm:"type:bytea" json:"-"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

