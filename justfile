set dotenv-load := true
set dotenv-path := "app/.env.local"

run:
    sleep 2; open "http://localhost:${PORT:-8080}"
    cd app && go run ./cmd

install:
    brew install go
    cd app && go mod download
    cd app && go version
