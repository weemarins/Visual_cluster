## Visual Kubernetes Topology

Aplicação web cloud-native para visualização interativa de topologia Kubernetes.

### Estrutura do monorepo

- `backend/` - API em Go (Gin, GORM, client-go, LDAP, JWT)
- `frontend/` - SPA em React + TypeScript + Vite + React Flow + Tailwind
- `deploy/` - Manifests Kubernetes (backend, frontend, PostgreSQL, ConfigMaps, Secrets, Ingress)

### Requisitos

## Visual Kubernetes Topology

Aplicação web cloud-native para visualização interativa de topologia Kubernetes.

### Estrutura do monorepo

- `backend/` - API em Go (Gin, GORM, client-go, LDAP, JWT)
- `frontend/` - SPA em React + TypeScript + Vite + React Flow + Tailwind
- `deploy/` - Manifests Kubernetes (backend, frontend, PostgreSQL, ConfigMaps, Secrets, Ingress)

### Requisitos

- Go (versão testada: 1.20+)
- Node.js 18+
- Docker / Kubernetes (kind / minikube / cluster real)
- PostgreSQL 14+ (local ou via Kubernetes)

### Backend - Desenvolvimento local

```bash
cd backend
go mod tidy
go run ./cmd/server
```

Variáveis de ambiente principais (pode-se usar um arquivo `.env` em desenvolvimento). Um exemplo com valores placeholder está em `backend/.env.example`.

Principais variáveis (exemplos):

```bash
APP_PORT=8080
APP_JWT_SECRET=change-me-secret
APP_JWT_EXP_MINUTES=60
APP_AES_KEY=change-me-32-bytes-key-change-me
DB_HOST=localhost
DB_PORT=5432
DB_USER=vkube
DB_PASSWORD=vkube
DB_NAME=vkube
LDAP_URL=ldap://ldap.example.com:389
LDAP_BASE_DN=dc=example,dc=com
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_BIND_PASSWORD=admin
POLL_INTERVAL_SECONDS=15
MAX_CLUSTERS_PER_USER=20
# Flags de manutenção
ENABLE_LOCAL_LOGIN=false
LOCAL_ADMIN_USER=admin_breakglass
```

Observações de segurança e produção:
- Não use os valores defaults para `APP_JWT_SECRET` e `APP_AES_KEY` em produção; exija que estas variáveis estejam definidas.
- O backend atualmente executa `AutoMigrate()` no startup; em produção recomenda-se usar migrações versionadas (ex.: `golang-migrate`) em vez de AutoMigrate em runtime.

### Frontend - Desenvolvimento local

```bash
cd frontend
npm install
npm run dev
```

### Build de produção (frontend)

```bash
cd frontend
npm run build
```

Os artefatos de build serão gerados em `frontend/dist/`.

### Build e push de imagens Docker

Exemplo para backend e frontend (substitua `<TAG>`):

```bash
# Backend image
docker build -t ghcr.io/weemarins/vkube-backend:<TAG> -f backend/Dockerfile backend
docker push ghcr.io/weemarins/vkube-backend:<TAG>

# Frontend image
docker build -t ghcr.io/weemarins/vkube-frontend:<TAG> -f frontend/Dockerfile frontend
docker push ghcr.io/weemarins/vkube-frontend:<TAG>
```

### Deploy em Kubernetes / OpenShift

1. Crie o namespace:

```bash
kubectl create namespace vkube
# ou no OpenShift: oc new-project vkube
```

2. Aplique os manifests (ordem sugerida):

```bash
kubectl apply -n vkube -f deploy/postgres.yaml
kubectl apply -n vkube -f deploy/backend-rbac.yaml
kubectl apply -n vkube -f deploy/backend-config.yaml
kubectl apply -n vkube -f deploy/backend-secret.yaml
kubectl apply -n vkube -f deploy/backend.yaml
kubectl apply -n vkube -f deploy/frontend.yaml
kubectl apply -n vkube -f deploy/ingress.yaml
```

No OpenShift você pode usar `oc` e rotas. Para atualizar a imagem do backend e forçar rollout:

```bash
oc set image deployment/backend backend=ghcr.io/weemarins/vkube-backend:<TAG> -n vkube
oc rollout status deployment/backend -n vkube
```

Comandos úteis:

```bash
kubectl get pods -n vkube
kubectl get svc,deploy,ing -n vkube
oc rollout history deployment/backend -n vkube
oc rollout undo deployment/backend -n vkube
```

### Observabilidade / Operação
- Recomenda-se expor health/readiness endpoints e métricas Prometheus no backend.
- Use Secrets do Kubernetes / OpenShift para armazenar `APP_JWT_SECRET`, chaves AES e credenciais do banco.

### Testes / CI
- Adicione um pipeline (ex.: GitHub Actions) com passos `go vet`, `golangci-lint`, `go test ./...`, `npm ci && npm run lint && npm run build`.

### Funcionalidades principais

- Autenticação via LDAP com JWT
- Upload e gerenciamento seguro de múltiplos kubeconfigs (AES-256)
- Conexão simultânea com múltiplos clusters Kubernetes
- Descoberta automática de recursos (Nodes, Namespaces, Deployments, StatefulSets, DaemonSets, ReplicaSets, Pods, Services, HPAs)
- Visualização de topologia em grafo (React Flow)
- Filtros por namespace, collapse de pods, painel lateral de detalhes
- Atualização periódica de topologia (polling)

Para detalhes de contribuição e execução de testes, veja `CONTRIBUTING.md` e `backend/.env.example`.
