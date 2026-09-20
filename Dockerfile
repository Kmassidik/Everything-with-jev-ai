# syntax=docker/dockerfile:1

# ---- build: go + templ + tailwind, produce a static binary ----
# Debian (glibc) so the Tailwind standalone binary runs (it is not musl-compatible).
FROM golang:1.26-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src

# tooling: templ (matches go.mod) + tailwind standalone (no Node)
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020
# Pin Tailwind v3 (config-file based) to match the dev shell; v4 uses CSS config.
RUN ARCH=$(dpkg --print-architecture); case "$ARCH" in amd64) TW=x64;; arm64) TW=arm64;; esac; \
    curl -fsSL -o /usr/local/bin/tailwindcss \
      "https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.17/tailwindcss-linux-${TW}" \
    && chmod +x /usr/local/bin/tailwindcss

COPY . .
RUN templ generate \
 && tailwindcss -c tailwind.config.js -i internal/web/input.css -o internal/web/static/css/app.css --minify \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/engine ./cmd/engine

# ---- runtime: minimal ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/engine /app/engine
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/engine"]
