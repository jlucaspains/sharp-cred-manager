# Azure Key Vault Secret Validity Checking

## Problem Statement
The app currently monitors TLS certificates (from URLs and Azure Key Vault). We want to add support for monitoring **Azure Key Vault secrets** — checking their expiration date and enabled/active status — and surfacing this in a tabbed dashboard (Tab 1: Certificates, Tab 2: Secrets).

## Key Decisions
- **Validity** = expiration date within warning threshold + `enabled` flag is `true` in Key Vault. Create a new config for the warning threshold called `SECRET_WARNING_VALIDITY_DAYS`
- **Config**: Hard rename `AZUREKEYVAULT_N` → `AZUREKEYVAULTCERT_N`; new `AZUREKEYVAULTSECRET_N` env var
- **Bulk vault**: If `AZUREKEYVAULTSECRET_N` has no secret name in path (just a vault URL), list and monitor **all** secrets in that vault
- **Rotation**: Out of scope for this version
- **UI**: Tabbed dashboard — existing certs grid on Tab 1, new secrets grid on Tab 2
- **Job**: Merge the cert and secret monitoring into a single `CheckCredJob`. Use a single env variable called `CHECK_CRED_JOB_SCHEDULE` to determine the schedule. Keep `CERT_WARNING_VALIDITY_DAYS` and `SECRET_WARNING_VALIDITY_DAYS` as separate thresholds.

## Architecture Overview

### New Data Flow
```
AZUREKEYVAULTSECRET_N env var
    → GetConfigSecrets() (secretService.go)
        → if no secret name: ListSecretPropertiesPages() → expand to N items
        → CheckSecretStatus(item) → SecretCheckResult
            → validate: enabled + not expired
    → /api/secret-list, /api/check-secret handlers
    → CheckSecretJob (new for secrets only)
    → frontend Secrets tab
```

### Breaking Change
`AZUREKEYVAULT_N` → `AZUREKEYVAULTCERT_N` in `certService.go` and docs/README.

---

## Tasks

### 1. Backend – Models
> Status: Complete

- Add `internal/models/secretCheckResult.go`
  - `SecretCheckResult` struct: Name, Url, Type (enum with AzureKeyVault only initially), ContentType, Enabled, ExpiresOn, NotBefore, IsValid, ValidationIssues, ExpirationWarning, ValidityInDays
  - `CheckSecretItem` struct: Name, Url, Type, SecretName (empty = all secrets)

### 2. Backend – Breaking Rename
> Status: Complete

- In `internal/services/certService.go`: rename `AZUREKEYVAULT_%d` → `AZUREKEYVAULTCERT_%d`
- Update tests in `internal/services/certService_test.go`

### 3. Frontend – Tabbed Dashboard
> Status: Complete

- Refactor `frontend/body.html`:
  - Add tab bar (Certificates / Secrets)
  - Move existing certs grid into Certificates tab panel
  - Add Secrets tab panel (initially empty, loaded via HTMX on tab activation)
- Rename `frontend/item.html` to `frontend/certItem.html`
- Create `frontend/secretItem.html` – loading skeleton for a secret card (mirrors `certItem.html`)
- Create `frontend/secretItemLoaded.html` – loaded secret card (Name, Enabled status, days until expiry)
- Create `frontend/secretItemModal.html` – modal detail view (Name, URL, Enabled, Expires, NotBefore, ValidationIssues)

### 4. Backend – Secret Service
> Status: Complete

- Add `github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets` to go.mod
- Create `internal/services/secretService.go`:
  - `GetConfigSecrets() []models.CheckSecretItem`: reads `AZUREKEYVAULTSECRET_N`, expands vault-only URLs to all secrets via `ListSecretPropertiesPages`
  - `CheckSecretStatus(item, warningDays) (*models.SecretCheckResult, error)`: fetches secret properties, checks `Enabled` + `Expires` attributes
  - Validity rules: secret must have `enabled=true` AND `expires` must not be within `warningDays` of today (warning) or already past (invalid)
- Add unit tests in `internal/services/secretService_test.go`

### 5. Backend – API Handlers
> Status: Complete

- Add `SecretList []models.CheckSecretItem` to `Handlers` struct in `internal/handlers/handlers.go`
- Create `internal/handlers/secretCheckHandlers.go`:
  - `GetSecretList(w, r)` → returns `SecretList` as JSON
  - `CheckSecretStatus(w, r)` → accepts `?name=`, looks up in SecretList, calls `services.CheckSecretStatus`
- Add handler tests in `internal/handlers/secretCheckHandlers_test.go`

### 6. Backend – Wire Up in main.go
> Status: Complete

- In `cmd/api/main.go`:
  - Call `services.GetConfigSecrets()` on startup alongside `GetConfigCerts()`
  - Set `handlers.SecretList` on the Handlers struct
  - Register routes: `GET /api/secret-list` and `GET /api/check-secret`

### 7. Frontend – Backend Routes for Secret Templates
> Status: Complete

- Add to `internal/handlers/frontend.go`:
  - `GET /secret-item?name=...` → renders `secretItemLoaded.html` (calls `CheckSecretStatus`)
  - `GET /secret-item-detail?name=...` → renders `secretItemModal.html`
- Register routes in `cmd/api/main.go`

### 8. Backend – Job Integration
> Status: Complete

- Replaced `CheckCertJob` and `CheckSecretJob` with a single `CheckCredJob` in `internal/jobs/CheckCredJob.go`:
  - `CheckCredJob` struct with `Init(schedule, level, certWarningDays, secretWarningDays, certList, secretList, notifier)`
  - Checks certs and secrets in a single `execute()`, building grouped notifications (`CheckNotificationGroup`) before calling `Notify` once
  - `CHECK_CRED_JOB_SCHEDULE` env var controls the single shared cron schedule
  - `CERT_WARNING_VALIDITY_DAYS` and `SECRET_WARNING_VALIDITY_DAYS` remain separate thresholds
- Added `CheckNotificationGroup{Label, Items}` to support extensible grouped notifications (future types like AKV Keys require no template changes)
- Updated `Notifier` interface: `Notify(groups []CheckNotificationGroup)`
- Updated `WebHookNotificationCard` to use `Groups []CheckNotificationGroup`; both Teams and Slack templates render a labeled section header per group
- Updated `cmd/api/main.go`: single `startJobs(certList, secretList)` call; removed `startSecretJobs`
- Added `CheckCredJob_test.go` consolidating all cert and secret job tests

### 9. Documentation
> Status: Complete

- Update `readme.md`:
  - **Breaking change note**: `AZUREKEYVAULT_N` renamed to `AZUREKEYVAULTCERT_N` in env vars table and all examples
  - Add new env vars to the table:
    - `AZUREKEYVAULTCERT_1..N` — replaces `AZUREKEYVAULT_N`; Azure Key Vault certificate URLs to monitor
    - `AZUREKEYVAULTSECRET_1..N` — Azure Key Vault secret URLs to monitor; vault-only URL monitors all secrets
    - `SECRET_WARNING_VALIDITY_DAYS` — days before expiry to trigger a warning for secrets (default: 30)
    - `CHECK_CRED_JOB_SCHEDULE` — single cron schedule for monitoring both certs and secrets
    - `SECRET_CHECK_INCLUDE_DISABLED` — when using a vault-only URL, include disabled secrets in monitoring (default: false)
    - `SECRET_CHECK_REQUIRE_EXPIRE_DATE` — when using a vault-only URL, only monitor secrets that have an expiration date set (default: true)
  - Add a "Secret Monitoring" section explaining the feature and dashboard Secrets tab
  - Update the Docker/ACA examples to show secrets monitoring alongside certs
  - Update the Features checklist to include Azure Key Vault secret monitoring