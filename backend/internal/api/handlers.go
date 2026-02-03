package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes" // Importante para o tipo de retorno do helper

	"github.com/example/vkube-topology/backend/internal/auth"
	"github.com/example/vkube-topology/backend/internal/config"
	"github.com/example/vkube-topology/backend/internal/crypto"
	"github.com/example/vkube-topology/backend/internal/db"
	"github.com/example/vkube-topology/backend/internal/k8s"
	"github.com/example/vkube-topology/backend/internal/models"
)

// =================================================================================
// AUTHENTICATION HANDLERS
// =================================================================================

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
}

func loginHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
			return
		}

		// 🔐 LOGIN LOCAL DE MANUTENÇÃO (BREAK-GLASS)
		if os.Getenv("ENABLE_LOCAL_LOGIN") == "true" {
			if !auth.AuthenticateLocal(req.Username, req.Password) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciais inválidas"})
				return
			}

			localUser := os.Getenv("LOCAL_ADMIN_USER")
			if localUser == "" || req.Username != localUser {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "acesso não autorizado"})
				return
			}

			var user models.User
			result := db.DB.Where("username = ?", localUser).First(&user)
			if result.Error != nil {
				user = models.User{
					Username:    localUser,
					DisplayName: "Maintenance Admin",
					Role:        "admin",
				}
				db.DB.Create(&user)
			}

			token, exp, err := auth.GenerateToken(user.Username, user.Role, cfg)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao gerar token"})
				return
			}

			c.JSON(http.StatusOK, loginResponse{
				Token:     token,
				ExpiresAt: exp,
				Username:  user.Username,
				Role:      user.Role,
			})
			return
		}

		// 🔐 LOGIN PADRÃO (LDAP)
		_, displayName, err := auth.LDAPAuthenticate(req.Username, req.Password, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciais inválidas"})
			return
		}

		var user models.User
		result := db.DB.Where("username = ?", req.Username).First(&user)
		if result.Error != nil {
			user = models.User{
				Username:    req.Username,
				DisplayName: displayName,
				Role:        "viewer",
			}

			var count int64
			db.DB.Model(&models.User{}).Count(&count)
			if count == 0 {
				user.Role = "admin"
			}
			db.DB.Create(&user)
		}

		token, exp, err := auth.GenerateToken(user.Username, user.Role, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao gerar token"})
			return
		}

		c.JSON(http.StatusOK, loginResponse{
			Token:     token,
			ExpiresAt: exp,
			Username:  user.Username,
			Role:      user.Role,
		})
	}
}

func meHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)
		c.JSON(http.StatusOK, gin.H{
			"username": claims.Username,
			"role":     claims.Role,
		})
	}
}

// =================================================================================
// CLUSTER CRUD HANDLERS
// =================================================================================

type clusterDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type createClusterRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	KubeconfigBase64 string `json:"kubeconfigBase64" binding:"required"`
}

func listClustersHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		var clusters []models.Cluster
		db.DB.Where("owner_username = ?", claims.Username).Find(&clusters)

		resp := make([]clusterDTO, 0, len(clusters))
		for _, cl := range clusters {
			resp = append(resp, clusterDTO{
				ID:          cl.ID,
				Name:        cl.Name,
				Description: cl.Description,
			})
		}
		c.JSON(http.StatusOK, resp)
	}
}

func createClusterHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		var req createClusterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
			return
		}

		kubeconfig, err := base64.StdEncoding.DecodeString(req.KubeconfigBase64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kubeconfig base64 inválido"})
			return
		}

		ciphertext, err := crypto.EncryptAES(cfg.AESKey, kubeconfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao criptografar kubeconfig"})
			return
		}

		cluster := models.Cluster{
			Name:                req.Name,
			Description:         req.Description,
			OwnerUsername:       claims.Username,
			EncryptedKubeconfig: ciphertext,
		}

		if err := db.DB.Create(&cluster).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao salvar cluster"})
			return
		}

		c.JSON(http.StatusCreated, clusterDTO{
			ID:          cluster.ID,
			Name:        cluster.Name,
			Description: cluster.Description,
		})
	}
}

func getClusterHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}

		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		var cluster models.Cluster
		if err := db.DB.Where("id = ? AND owner_username = ?", id, claims.Username).First(&cluster).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "cluster não encontrado"})
			return
		}

		c.JSON(http.StatusOK, clusterDTO{
			ID:          cluster.ID,
			Name:        cluster.Name,
			Description: cluster.Description,
		})
	}
}

func updateClusterHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		var cluster models.Cluster
		if err := db.DB.Where("id = ? AND owner_username = ?", id, claims.Username).First(&cluster).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "cluster não encontrado"})
			return
		}

		var req createClusterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
			return
		}

		cluster.Name = req.Name
		cluster.Description = req.Description

		if req.KubeconfigBase64 != "" {
			kubeconfig, err := base64.StdEncoding.DecodeString(req.KubeconfigBase64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kubeconfig base64 inválido"})
				return
			}
			ciphertext, err := crypto.EncryptAES(cfg.AESKey, kubeconfig)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao criptografar kubeconfig"})
				return
			}
			cluster.EncryptedKubeconfig = ciphertext
		}

		if err := db.DB.Save(&cluster).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao atualizar cluster"})
			return
		}

		c.JSON(http.StatusOK, clusterDTO{
			ID:          cluster.ID,
			Name:        cluster.Name,
			Description: cluster.Description,
		})
	}
}

func deleteClusterHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		if err := db.DB.Where("id = ? AND owner_username = ?", id, claims.Username).Delete(&models.Cluster{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao remover cluster"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// =================================================================================
// RESOURCE HANDLERS (YAML & LOGS)
// =================================================================================

func getResourceYAMLHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Obtém cliente K8s reutilizando lógica do Helper
		client, err := getK8sClientFromRequest(c, cfg)
		if err != nil {
			// O helper já escreveu o erro no JSON response
			return
		}

		// 2. Parâmetros da Query
		ns := c.Query("namespace")
		name := c.Query("name")
		kind := c.Query("kind")

		if ns == "" || name == "" || kind == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "namespace, name e kind são obrigatórios"})
			return
		}

		// 3. Chama função do pacote k8s (AINDA VAMOS IMPLEMENTAR)
		yamlContent, err := k8s.GetResourceYAML(context.Background(), client, ns, kind, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao buscar YAML: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, yamlContent)
	}
}

func getResourceLogsHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		client, err := getK8sClientFromRequest(c, cfg)
		if err != nil {
			return
		}

		ns := c.Query("namespace")
		name := c.Query("name")
		container := c.Query("container")
		tailStr := c.Query("tail")

		tail := 100 // Default
		if tailStr != "" {
			if t, err := strconv.Atoi(tailStr); err == nil {
				tail = t
			}
		}

		if ns == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "namespace e name são obrigatórios"})
			return
		}

		// 3. Chama função do pacote k8s (AINDA VAMOS IMPLEMENTAR)
		logs, err := k8s.GetPodLogs(context.Background(), client, ns, name, container, int64(tail))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao buscar logs: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"lines": logs})
	}
}

// =================================================================================
// TOPOLOGY HANDLERS
// =================================================================================

func topologyHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("clusterID")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}

		ns := c.Query("namespace")
		if ns == "" {
			ns = "all"
		}

		claimsVal, _ := c.Get("user")
		claims := claimsVal.(*auth.Claims)

		var cluster models.Cluster
		if err := db.DB.Where("id = ? AND owner_username = ?", id, claims.Username).First(&cluster).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "cluster não encontrado"})
			return
		}

		kubeconfig, err := crypto.DecryptAES(cfg.AESKey, cluster.EncryptedKubeconfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao decifrar kubeconfig"})
			return
		}

		client, err := k8s.NewClient(kubeconfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao criar client Kubernetes"})
			return
		}

		graph, err := k8s.BuildTopologyGraph(context.Background(), client, ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao construir grafo"})
			return
		}

		c.JSON(http.StatusOK, graph)
	}
}

// =================================================================================
// HELPERS
// =================================================================================

// getK8sClientFromRequest busca o cluster pelo ID na URL, verifica permissão e retorna o client K8s
func getK8sClientFromRequest(c *gin.Context, cfg *config.Config) (*kubernetes.Clientset, error) {
	// Pega o parametro :id da rota
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id do cluster inválido"})
		return nil, err
	}

	claimsVal, _ := c.Get("user")
	claims := claimsVal.(*auth.Claims)

	var cluster models.Cluster
	if err := db.DB.Where("id = ? AND owner_username = ?", id, claims.Username).First(&cluster).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster não encontrado"})
		return nil, err
	}

	kubeconfig, err := crypto.DecryptAES(cfg.AESKey, cluster.EncryptedKubeconfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao decifrar kubeconfig"})
		return nil, err
	}

	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao criar client Kubernetes"})
		return nil, err
	}

	return client, nil
}