## Visual Kubernetes Topology

Aplicação web cloud-native para visualização interativa de topologia Kubernetes.

### Estrutura do monorepo

- `backend/` - API em Go (Gin, GORM, client-go, LDAP, JWT)
- `frontend/` - SPA em React + TypeScript + Vite + React Flow + Tailwind
- `deploy/` - Manifests Kubernetes (backend, frontend, PostgreSQL, ConfigMaps, Secrets, Ingress)

### Requisitos

- Go 1.23+
- Node.js 18+
- Docker / Kubernetes (kind / minikube / cluster real)
- PostgreSQL 14+ (local ou via Kubernetes)

### Backend - Desenvolvimento local

```bash
cd backend
go mod tidy
go run ./cmd/server
```

Variáveis de ambiente principais (podem ser definidas em `.env` para desenvolvimento):

```bash
export APP_PORT=8080
export APP_JWT_SECRET=change-me-secret
export APP_JWT_EXP_MINUTES=60
export APP_AES_KEY=change-me-32-bytes-key-change-me
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=vkube
export DB_PASSWORD=vkube
export DB_NAME=vkube
export LDAP_URL=ldap://ldap.example.com:389
export LDAP_BASE_DN=dc=example,dc=com
export LDAP_BIND_DN=cn=admin,dc=example,dc=com
export LDAP_BIND_PASSWORD=admin
export POLL_INTERVAL_SECONDS=15
export MAX_CLUSTERS_PER_USER=20
```

### Frontend - Desenvolvimento local

```bash
cd frontend
npm install
npm run dev
```

### Build de produção

```bash
cd frontend
npm run build
```

Os artefatos de build serão gerados em `frontend/dist/`.

### Deploy em Kubernetes

1. Crie o namespace:

```bash
kubectl create namespace vkube
```

2. Aplique os manifests:

```bash
kubectl apply -n vkube -f deploy/postgres.yaml
kubectl apply -n vkube -f deploy/backend-rbac.yaml
kubectl apply -n vkube -f deploy/backend-config.yaml
kubectl apply -n vkube -f deploy/backend-secret.yaml
kubectl apply -n vkube -f deploy/backend.yaml
kubectl apply -n vkube -f deploy/frontend.yaml
kubectl apply -n vkube -f deploy/ingress.yaml
```

3. Verifique os pods:

```bash
kubectl get pods -n vkube
```

4. Acesse a aplicação via Ingress configurado.

#### OpenShift (Route)

Se estiver usando OpenShift, aplique a rota opcional:

```bash
kubectl apply -n vkube -f deploy/backend-route.yaml
```

### Funcionalidades principais

- Autenticação via LDAP com JWT
- Upload e gerenciamento seguro de múltiplos kubeconfigs (AES-256)
- Conexão simultânea com múltiplos clusters Kubernetes
- Descoberta automática de recursos (Nodes, Namespaces, Deployments, StatefulSets, DaemonSets, ReplicaSets, Pods, Services, HPAs)
- Visualização de topologia em grafo (React Flow)
- Filtros por namespace, collapse de pods, painel lateral de detalhes
- Atualização periódica de topologia (polling)
