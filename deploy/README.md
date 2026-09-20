# Deploy — jevai on the Mac Studio

Live: **https://jevai.glicc.id** (Mac Studio `dalangs-Mac-Studio`, behind the Cloudflare tunnel).

The engine is a single self-contained Go binary (static assets are embedded), so deploy is
copy-one-file. No Go/templ/Tailwind toolchain is needed on the host.

## Build (on the dev Mac — same arch as prod: darwin/arm64)

```bash
templ generate
tailwindcss -c tailwind.config.js -i internal/web/input.css -o internal/web/static/css/app.css --minify
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/engine-darwin-arm64 ./cmd/engine
```

## Ship + run (per-user LaunchAgent, KeepAlive, no sudo)

```bash
ssh kurnia-mac 'mkdir -p ~/jevai'
scp bin/engine-darwin-arm64 kurnia-mac:jevai/engine
scp deploy/id.jevai.engine.plist kurnia-mac:Library/LaunchAgents/
ssh kurnia-mac 'chmod +x ~/jevai/engine; \
  launchctl bootout gui/$(id -u)/id.jevai.engine 2>/dev/null; \
  launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/id.jevai.engine.plist'
```

Engine listens on `127.0.0.1:8092`. Logs: `~/jevai/engine.log`. DB: `~/jevai/jevai.db`.

## Public route (Cloudflare tunnel — reuses mac-multi-server tooling)

```bash
ssh kurnia-mac 'bash -lc "cd ~/mac-multi-server; set -a; . ./.env; set +a; \
  . lib/common.sh; . lib/cloudflare.sh; cf_route_add jevai.\$DOMAIN 127.0.0.1 8092"'
```

## Go live for real (leave SAMPLE mode)

Add `TYPESAFE_API_KEY` to the plist's `EnvironmentVariables`, then reload:

```bash
ssh kurnia-mac 'launchctl kickstart -k gui/$(id -u)/id.jevai.engine'
```

## Notes

- Ports on the host: 8088 panel · 8089 · 8090 beszel · 8091 chat · **8092 jevai**.
- Redeploy = rebuild binary, `scp` it over, `launchctl kickstart -k gui/$(id -u)/id.jevai.engine`.
- Docker image (`docker build -t jevai-engine .`) also works (22.7 MB distroless) for cloud later.
