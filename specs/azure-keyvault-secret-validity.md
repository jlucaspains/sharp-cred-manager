# Azure Key Vault Secret Validity Checking

## Problem Statement
The app currently monitors TLS certificates (from URLs and Azure Key Vault). We want to add support for monitoring **Azure Key Vault secrets** — checking their expiration date and enabled/active status — and surfacing this in a tabbed dashboard (Tab 1: Certificates, Tab 2: Secrets).

## Key Decisions
- **Validity** = expiration date within warning threshold + `enabled` flag is `true` in Key Vault. Create a new config for the warning threshold called `SECRET_WARNING_VALIDITY_DAYS`
- **Config**: Hard rename `AZUREKEYVAULT_N` → `AZUREKEYVAULTCERT_N`; new `AZUREKEYVAULTSECRET_N` env var
- **Bulk vault**: If `AZUREKEYVAULTSECRET_N` has no secret name in path (just a vault URL), list and monitor **all** secrets in that vault
- **Rotation**: Out of scope for this version
- **UI**: Tabbed dashboard — existing certs grid on Tab 1, new secrets grid on Tab 2
- **Job**: Create a new job to monitor secrets called `CheckSecretJob`. Follow example implementation from `CheckCertJob`. Create a new env variable called `CHECK_SECRET_JOB_SCHEDULE` to determine the schedule to run the secret job.

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
> Status: Not Started

- Refactor `frontend/body.html`:
  - Add tab bar (Certificates / Secrets)
  - Move existing certs grid into Certificates tab panel
  - Add Secrets tab panel (initially empty, loaded via HTMX on tab activation)
- Rename `frontend/item.html` to `frontend/certItem.html`
- Create `frontend/secretItem.html` – loading skeleton for a secret card (mirrors `certItem.html`)
- Create `frontend/secretItemLoaded.html` – loaded secret card (Name, Enabled status, days until expiry)
- Create `frontend/secretItemModal.html` – modal detail view (Name, URL, Enabled, Expires, NotBefore, ValidationIssues)

### 4. Backend – Secret Service
> Status: Not Started

- Add `github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets` to go.mod
- Create `internal/services/secretService.go`:
  - `GetConfigSecrets() []models.CheckSecretItem`: reads `AZUREKEYVAULTSECRET_N`, expands vault-only URLs to all secrets via `ListSecretPropertiesPages`
  - `CheckSecretStatus(item, warningDays) (*models.SecretCheckResult, error)`: fetches secret properties, checks `Enabled` + `Expires` attributes
  - Validity rules: secret must have `enabled=true` AND `expires` must not be within `warningDays` of today (warning) or already past (invalid)
- Add unit tests in `internal/services/secretService_test.go`

### 5. Backend – API Handlers
> Status: Not Started

- Add `SecretList []models.CheckSecretItem` to `Handlers` struct in `internal/handlers/handlers.go`
- Create `internal/handlers/secretCheckHandlers.go`:
  - `GetSecretList(w, r)` → returns `SecretList` as JSON
  - `CheckSecretStatus(w, r)` → accepts `?name=`, looks up in SecretList, calls `services.CheckSecretStatus`
- Add handler tests in `internal/handlers/secretCheckHandlers_test.go`

### 6. Backend – Wire Up in main.go
> Status: Not Started

- In `cmd/api/main.go`:
  - Call `services.GetConfigSecrets()` on startup alongside `GetConfigCerts()`
  - Set `handlers.SecretList` on the Handlers struct
  - Register routes: `GET /api/secret-list` and `GET /api/check-secret`

### 7. Frontend – Backend Routes for Secret Templates
> Status: Not Started

- Add to `internal/handlers/frontend.go`:
  - `GET /secret-item?name=...` → renders `secretItemLoaded.html` (calls `CheckSecretStatus`)
  - `GET /secret-item-detail?name=...` → renders `secretItemModal.html`
- Register routes in `cmd/api/main.go`

### 8. Backend – Job Integration
> Status: Not Started

- Create `CheckSecretJob` in `internal/jobs/CheckSecretJob.go`
  - Implement it similarly to `internal/jobs/CheckCertJob.go`
  - Create a new model `SecretCheckNotification`
  - Send notification using typical notifier.
- Update `cmd/api/main.go` `startJobs()` to pass the secrets list
- Update job tests in `internal/jobs/CheckCertJob_test.go`

### 9. Documentation
> Status: Not Started

- Update `readme.md`: document new env var names, `AZUREKEYVAULTCERT_N`, `AZUREKEYVAULTSECRET_N`, breaking change note
- Update examples with scenarios for certs and secrets