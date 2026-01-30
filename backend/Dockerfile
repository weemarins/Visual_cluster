# ---------- build ----------
FROM golang:1.23-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# ---------- runtime ----------
FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /app/server /app/server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
