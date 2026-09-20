# syntax=docker/dockerfile:1

# ---- build: go + templ + tailwind, produce a static binary ----
FROM golang:1.26-alpine AS build
RUN apk add --no-cache curl
WORKDIR /src

# tooling: templ (matches go.mod) + tailwind standalone (no Node)
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020
RUN ARCH=$(uname -m); case "$ARCH" in x86_64) TW=x64;; aarch64) TW=arm64;; esac; \
    curl -sSL -o /usr/local/bin/tailwindcss \
      "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-${TW}" \
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
