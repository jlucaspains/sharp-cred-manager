# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

Sharp Cred Manager monitors TLS certificates (from websites and Azure Key Vault) and Azure Key Vault secrets for expiration. It provides a web dashboard and sends scheduled webhook notifications (Teams/Slack) when credentials approach expiration.

## Commands

**Run the web server:**
```bash
go run ./cmd/api/main.go
```

**Run the CLI tool:**
```bash
go run ./cmd/sharp-cred-manager/ check --url https://example.com --warning-threshold 90
```

**Run all tests:**
```bash
go test -cover ./...
```

**Run a single test:**
```bash
go test -v ./internal/handlers/ -run TestCheckCertStatus
```

**Generate CSS (required after frontend/styles.css changes):**
```bash
tailwindcss -i ./frontend/styles.css -o ./public/styles.css --minify
```

**Hot reload during development:**
```bash
air  # configured via .air.toml
```

**Build Docker image:**
```bash
docker build -f Dockerfile -t sharp-cred-manager .
```

## Architecture

Two entry points:
- `cmd/api/main.go` — Web server + background job orchestrator
- `cmd/sharp-cred-manager/` — Cobra-based CLI tool

Core packages under `internal/`:
- `handlers/` — HTTP handlers serving both JSON API and server-side HTML (Go `html/template`)
- `services/` — Business logic: TLS certificate validation (`certService.go`) and Azure Key Vault secret checks (`secretService.go`)
- `jobs/` — Cron-scheduled background jobs (`CheckCredJob.go`) and webhook notification (`webHookNotifier.go`, `notificationTemplates.go`)
- `models/` — Shared request/response structs
- `midlewares/` — HTTP middleware (request logging)

Frontend: `frontend/*.html` Go templates + HTMX for dynamic updates + Tailwind CSS. Compiled static assets go in `public/`.

## Key Patterns

**Configuration is entirely environment-based** — no config files at runtime. `godotenv` auto-loads `.env` in development. Numbered env vars define monitored resources:
- `SITE_1..N` — websites to check
- `AZUREKEYVAULTCERT_1..N` — Key Vault certificate URLs
- `AZUREKEYVAULTSECRET_1..N` — Key Vault secret URLs
- `CHECK_CRED_JOB_SCHEDULE` — cron expression for background checks
- `WEBHOOK_URL` + `WEBHOOK_TYPE` (teams/slack) — notification targets
- `WEB_HOST_PORT` (default `:8000`), `CERT_WARNING_VALIDITY_DAYS` (default 30), `SECRET_WARNING_VALIDITY_DAYS` (default 30)
- `HEADLESS=true` — skip web server, run jobs only
- Azure auth: `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`

**HTTP routing** uses Go's standard `http.NewServeMux()` with method-prefixed patterns (`GET /api/check-cert`). Handlers are methods on a `Handlers` struct registered in `cmd/api/main.go`.

**Frontend pattern**: Pages load skeleton placeholders, then HTMX fetches real data via `hx-get` on page load. Templates are in `frontend/`, served from memory at startup.

**Notifier interface** (`Notify()`, `IsReady()`) decouples job scheduling from webhook dispatch. `emptyNotifier` is a no-op when no webhook is configured. Adding a new credential type means adding a new `CheckNotificationGroup` to `CheckCredJob`.

**Azure auth** uses `azidentity.NewDefaultAzureCredential()` — supports all standard Azure credential chains.

## CI/CD

- **PRs**: SonarCloud scan, Docker build test, Go tests with coverage posted to PR
- **Releases**: Docker image built and pushed to Docker Hub with semver tags + latest; Docker Hub description updated from `readme.md`
