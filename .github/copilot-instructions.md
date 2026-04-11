# Copilot Instructions

## What This Project Does

Sharp Cert Manager monitors TLS certificates (from URLs) and Azure Key Vault certificates/secrets for expiration. It has two entry points: a web dashboard (`cmd/api/`) and a CLI tool (`cmd/sharp-cred-manager/`). Background jobs run on cron schedules and send expiry alerts to Teams or Slack webhooks.

## Build & Run

```bash
# Build Tailwind CSS (required before building/running)
tailwindcss -i ./frontend/styles.css -o ./public/styles.css --minify

# Run web server locally (reads from .env)
go run ./cmd/api/main.go

# Run CLI tool
go run ./cmd/sharp-cred-manager/ check --url https://example.com

# Build Docker image
docker build -f Dockerfile -t sharp-cred-manager .
```

## Tests & Lint

```bash
# Run all tests
go test -cover ./...

# Run a single test (example)
go test -v ./internal/handlers/ -run TestGetCertList

# Run tests in one package
go test -v ./internal/services/
```

Linting is done via SonarCloud in CI (`.github/workflows/pr.yml`). There is no local lint script.

## Architecture

```
cmd/api/main.go              ← Web server entry point (starts HTTP server + background jobs)
cmd/sharp-cred-manager/      ← CLI entry point (Cobra-based)
internal/
  handlers/                  ← HTTP handlers (API JSON + server-side HTML rendering)
  services/                  ← Business logic (TLS checks, Azure Key Vault integration)
  jobs/                      ← Cron-scheduled jobs + webhook notifiers
  models/                    ← Request/response structs
  midlewares/                ← HTTP middleware (request/response logger)
frontend/                    ← Go html/template files
public/                      ← Compiled CSS + static assets (htmx.min.js, favicon)
```

The frontend is server-side rendered with Go's `html/template`. HTMX is used for dynamic updates — cert/secret cards load as skeleton placeholders and fetch their own data via `hx-get` on load. No build step for JS; Tailwind CSS must be compiled separately.

## Key Conventions

### HTTP Handlers

Use receiver methods on the `Handlers` struct. Register routes in `cmd/api/main.go` using Go's built-in `http.NewServeMux()` with method-prefixed patterns:

```go
// internal/handlers/
func (h Handlers) MyEndpoint(w http.ResponseWriter, r *http.Request) { ... }

// cmd/api/main.go
router.HandleFunc("GET /api/my-endpoint", handlers.MyEndpoint)
```

### Configuration

Config is read from numbered environment variables at startup — there is no config file format. Iterate with `os.LookupEnv` in a loop:

```go
for i := 1; true; i++ {
    val, ok := os.LookupEnv(fmt.Sprintf("SITE_%d", i))
    if !ok { break }
}
```

Key env vars: `SITE_1..N`, `AZUREKEYVAULTCERT_1..N`, `AZUREKEYVAULTSECRET_1..N`, `CHECK_CRED_JOB_SCHEDULE`, `WEBHOOK_URL`, `WEBHOOK_TYPE` (teams/slack), `WEB_HOST_PORT` (default `:8000`), `HEADLESS` (skip web server). For local dev, put these in a `.env` file — `godotenv` loads it automatically.

### Background Jobs

Jobs in `internal/jobs/` implement an `Init()` method and use `github.com/adhocore/gronx` for cron scheduling. `CheckCredJob` (in `CheckCredJob.go`) monitors both certs and secrets in a single execution, building `[]CheckNotificationGroup` and calling `Notify` once. The `Notifier` interface (`Notify([]CheckNotificationGroup)`, `IsReady()`) decouples webhook dispatch from job logic. `emptyNotifier.go` is the no-op implementation used when no webhook is configured.

`CheckNotificationGroup{Label, Items}` is the extensibility point for future credential types (e.g. AKV Keys): add a new group in `execute()` and the templates require no changes.

### Frontend Templates

Templates are loaded via glob pattern at handler init time. When adding a new page:
1. Create `frontend/mypage.html`
2. Add a handler in `internal/handlers/frontend.go` that calls `initTemplates()` then `ExecuteTemplate`
3. Use HTMX attributes (`hx-get`, `hx-trigger="load"`, `hx-swap="outerHTML"`) for deferred data fetching — see `certItem.html` as the reference pattern

### Error Handling in Handlers

```go
// API endpoints — return JSON error
if err != nil {
    h.JSON(w, http.StatusBadRequest, &models.ErrorResult{Errors: []string{err.Error()}})
    return
}

// HTML endpoints — use the handleError helper
err = tmpl.ExecuteTemplate(w, "page.html", data)
handleError(w, err)
```

### Logging

Use the standard `log` package for the web server. The CLI uses `log/slog` (enabled with `--verbose`). No structured logging library is used in the web server.

## Docker

Multi-stage build: `golang:1.24-alpine3.22` builder → `scratch` final image. The binary runs as non-root `appuser` (UID 10001). Frontend templates and `public/` assets are copied into the image and served from the working directory — they are **not** embedded via `go:embed`.

## Azure Integration

Azure Key Vault access uses `azidentity.NewDefaultAzureCredential()`. For local dev with Azure resources, set `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, and `AZURE_CLIENT_SECRET` in `.env`. The `azcertificates` and `azsecrets` SDK packages are used directly in `internal/services/`.
