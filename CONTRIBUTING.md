# Contributing

Obrigado por contribuir com o Visual Kubernetes Topology! Este documento descreve como rodar, testar e construir localmente.

## Requisitos locais

- Go (>=1.20)
- Node.js (18+)
- Docker
- kubectl / oc (opcional)

## Backend - executar localmente

1. Defina variáveis de ambiente (veja `backend/.env.example`) ou crie um arquivo `.env` e carregue com `direnv`/`source`.

2. Para rodar localmente apontando para um PostgreSQL local:

```bash
# no diretório raiz do repo
cd backend
# instala dependências
go mod tidy
# executa
go run ./cmd/server
```

3. Para testes unitários:

```bash
cd backend
go test ./... -v
```

4. Lint / vet:

```bash
go vet ./...
# se usar golangci-lint
golangci-lint run
```

## Frontend - executar localmente

```bash
cd frontend
npm install
npm run dev
```

Lint & build:

```bash
npm run lint
npm run build
```

## Build de imagens e deploy (exemplo)

```bash
# Backend
docker build -t ghcr.io/weemarins/vkube-backend:local -f backend/Dockerfile backend
docker push ghcr.io/weemarins/vkube-backend:local

# Atualiza no OpenShift/K8s
oc set image deployment/backend backend=ghcr.io/weemarins/vkube-backend:local -n vkube
oc rollout status deployment/backend -n vkube
```

## Dicas

- Nunca commite segredos no repositório. Use Kubernetes Secrets ou um secret manager.
- Em produção, prefira migrações versionadas (ex.: `golang-migrate`) em vez de `AutoMigrate()` automático.
- Se quiser testar contra clusters reais, adicione o kubeconfig no app via UI (criptografado) ou monte como Secret apenas em ambiente de debugging.
