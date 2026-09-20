# jevai-products — task runner.  `just` to list.
set shell := ["bash", "-cu"]

_default:
    @just --list

# generate templ Go + build tailwind css
generate:
    templ generate
    tailwindcss -c tailwind.config.js -i internal/web/input.css -o internal/web/static/css/app.css --minify

# live-reload dev server (regenerates on save)
dev:
    templ generate
    tailwindcss -c tailwind.config.js -i internal/web/input.css -o internal/web/static/css/app.css
    air

# build the engine binary
build: generate
    go build -o bin/engine ./cmd/engine

# run the engine
run: generate
    go run ./cmd/engine

test:
    go test ./...

lint:
    gofumpt -l -w .
    golangci-lint run

# docker
docker-build:
    docker build -t jevai-engine .
docker-up:
    docker compose up --build
