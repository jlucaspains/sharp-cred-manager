# App Registration Credential Expiration Warnings

## Problem Statement
The app currently monitors TLS certificates and Azure Key Vault secrets. We want to add support for monitoring **Azure App Registration credentials** — both client secrets (`passwordCredentials`) and certificates (`keyCredentials`) — checking their expiration dates, and surfacing this in a new dedicated tab on the dashboard.

## Key Decisions
- **Validity** = `endDateTime` is not in the past and not within warning threshold; `startDateTime` is not in the future. No "enabled" concept on either credential type.
- **Config**: `APPREGISTRATION_N=<appId>` — one entry per app registration; all client secrets **and** certificates on that registration are monitored automatically. The tenant is derived from `AZURE_TENANT_ID` (single-tenant only). Multi-tenant setups are not supported.
- **Warning threshold**: New `APP_REG_WARNING_VALIDITY_DAYS` env var (default: 30). Kept separate from `SECRET_WARNING_VALIDITY_DAYS`.
- **API**: Microsoft Graph API via direct HTTP calls using a bearer token obtained from `azidentity.NewDefaultAzureCredential()` + `azcore/policy`. The `msgraph-sdk-go` SDK was intentionally skipped to avoid a large transitive dependency tree; `azcore` is already a direct dependency.
- **Required Graph permission**: `Application.Read.All` (application permission in Entra ID).
- **Discovery**: `GetConfigAppRegs()` prepares the app registrations for load, returning one `CheckAppRegItem` per app registration. Individual credentials, the app name, and object id are resolved on demand in `CheckAppRegStatus`.
- **Data model**: One `AppRegCheckResult` per app registration, containing a `[]AppRegCredentialResult` — one entry per secret or certificate. The top-level `IsValid` and `ExpirationWarning` fields reflect the worst state across all credentials.
- **UI**: One card per app registration (not per credential). The card body lists all credentials with their type and expiry. Clicking opens a modal with full per-credential details.
- **Job**: Extend `CheckCredJob` with `appRegList` and `appRegWarningDays`; add an "App Registrations" `CheckNotificationGroup`. Each app registration produces one `CheckNotification`; issues from individual credentials appear as separate message lines.

## Architecture Overview

### New Data Flow
```
APPREGISTRATION_N env var (appId) + AZURE_TENANT_ID env var
    → GetConfigAppRegs() (appRegService.go)
        → TenantId populated from AZURE_TENANT_ID (display only)
        → CheckAppRegItem (one per app)
    → CheckAppRegStatus(item, warningDays) → AppRegCheckResult
        → Graph API: GET /applications?$filter=appId eq '{appId}'
            → iterate passwordCredentials → AppRegCredentialResult (type: Secret)
            → iterate keyCredentials      → AppRegCredentialResult (type: Certificate)
        → top-level IsValid = all credentials valid
        → top-level ExpirationWarning = any credential warning
    → /api/appreg-list, /api/check-appreg handlers
    → CheckCredJob (extended)
    → frontend App Registrations tab (one card per app)
```

---

## UI Mockups

### Tab Bar (3 tabs, App Registrations active)

```
┌─────────────────────────────────────────────────────────────┐
│  sharp-cred-manager                                         │
├─────────────────────────────────────────────────────────────┤
│  Certificates │ Secrets │ App Registrations                 │
│  ────────────   ───────   ══════════════════                │
│                           (active: blue underline)          │
└─────────────────────────────────────────────────────────────┘
```

The `showTab()` JS function is extended to handle `'appregs'`. The panel is lazy-loaded via HTMX on first click, identical to the Secrets tab pattern.

---

### App Registrations Panel — Loading State

One skeleton card per app registration. HTMX replaces each after a 200ms delay.

```
┌──────────────────────────┐  ┌──────────────────────────┐  ┌──────────────────────────┐
│  🔵  tenant/appId        │  │  🔵  tenant/appId        │  │  🔵  tenant/appId        │
│  ████████████████        │  │  ████████████████        │  │  ████████████████        │
│  ██████████████████      │  │  ██████████████████      │  │  ██████████████████      │
│  ████████████            │  │  ████████████            │  │  ████████████            │
│  ████████████████        │  │  ████████████████        │  │  ████████████████        │
│  ████████████████████    │  │  ████████████████████    │  │  ████████████████████    │
└──────────────────────────┘  └──────────────────────────┘  └──────────────────────────┘
  (pulsing skeleton)
```

---

### App Registrations Panel — Loaded State

One card per app registration. The header icon reflects the **worst** status across all credentials. The body lists each credential individually.

```
┌──────────────────────────────────┐  ┌──────────────────────────────────┐  ┌──────────────────────────────────┐
│  🟢  MyWebApp                    │  │  🔴  BillingService              │  │  🟢  DataPipeline                │
│  ─────────────────────────────── │  │  ─────────────────────────────── │  │  ─────────────────────────────── │
│  🔑 CI Deploy Key    120 days    │  │  🔑 API Auth Key     EXPIRED     │  │  🔑 Worker Secret  45 days       │
│  📜 Auth Cert        200 days    │  │  📜 Auth Cert        300 days    │  │  📜 Auth Cert      22 days ⚠     │
│                                  │  │                                  │  │                                  │
│  (click for details)             │  │  (click for details)             │  │  (click for details)             │
└──────────────────────────────────┘  └──────────────────────────────────┘  └──────────────────────────────────┘
  all valid                             any invalid → red header icon          all valid but one warning (⚠)
```

Icon and row colour logic:
- Card header icon green (`bg-green-600`) — all credentials valid
- Card header icon red (`bg-red-600`) — one or more credentials invalid
- Per-credential row: expiry shown in days; `EXPIRED` in red text when past; `⚠` suffix when within warning threshold but not yet expired; `N/A` when no `endDateTime`
- `🔑` = client secret (`passwordCredential`); `📜` = certificate (`keyCredential`)

---

### App Reg Item Card — Field Layout (`appRegItemLoaded.html`)

```
┌──────────────────────────────────────────────────┐
│  [●]  MyWebApp                                   │  ← AppName; icon green/red (worst status)
│  ───────────────────────────────────────────     │
│  🔑  CI Deploy Key              120 days         │  ← Secret, DisplayName, ValidityInDays
│  📜  Auth Cert                  200 days         │  ← Certificate, DisplayName, ValidityInDays
│  🔑  Legacy Secret              EXPIRED          │  ← Invalid credential, red text
└──────────────────────────────────────────────────┘
```

- Each credential row: type icon + `DisplayName` + expiry status
- Expiry status: `{N} days` (valid), `{N} days ⚠` (warning), `EXPIRED` (invalid), `N/A` (no end date)
- Rows are not individually clickable — click anywhere on the card opens the detail modal for the whole app

---

### App Reg Detail Modal (`appRegItemModal.html`)

```
┌────────────────────────────────────────────────────────────────────┐
│  [✓/✗]  MyWebApp                                            [✕]   │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  App Name   │  MyWebApp                                           │
│  App ID     │  xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx               │
│  Tenant ID  │  yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy               │
│                                                                    │
│  ── Credentials ─────────────────────────────────────────────     │
│                                                                    │
│  Type         │ Name           │ Valid From  │ Expires On │ Status │
│  ─────────────┼────────────────┼────────────┼────────────┼─────── │
│  🔑 Secret    │ CI Deploy Key  │ Jan 1 2025 │ Apr 1 2026 │ ✓ 120d │
│  📜 Cert      │ Auth Cert      │ Mar 1 2024 │ Mar 1 2027 │ ✓ 200d │
│  🔑 Secret    │ Legacy Secret  │ Jan 1 2023 │ Jan 1 2024 │ ✗ Exp  │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│  [ OK ]                                                            │
└────────────────────────────────────────────────────────────────────┘
```

- App-level section: App Name, App ID, Tenant ID
- Credentials table: one row per credential; columns are Type, Name (DisplayName), Valid From (StartDateTime), Expires On (EndDateTime), Status
- Status column: `✓ {N}d` (valid), `⚠ {N}d` (warning), `✗ Exp` (expired), `✗ Future` (startDateTime in future), `✓ N/A` (no expiry set)
- Modal header icon uses overall app validity (same green/red logic as card)

---

## Tasks

### 1. Backend – Models
> Status: Complete

- Add `internal/models/appRegCheckResult.go`:
  - `AppRegCredentialType int` enum: `AppRegCredentialSecret`, `AppRegCredentialCertificate`
  - `AppRegCredentialResult` struct: `KeyId`, `DisplayName`, `CredentialType AppRegCredentialType`, `StartDateTime *time.Time`, `EndDateTime *time.Time`, `IsValid`, `ValidationIssues []string`, `ExpirationWarning`, `HasExpiration`, `ValidityInDays`
  - `AppRegCheckResult` struct: `Name` (`tenantId/appId`), `AppName`, `AppId`, `TenantId`, `AppObjectId`, `IsValid` (false if any credential invalid), `ExpirationWarning` (true if any credential warning), `Credentials []AppRegCredentialResult`
  - `CheckAppRegItem` struct: `Name` (`tenantId/appId`), `TenantId`, `AppId`, `AppObjectId`, `AppName`

### 2. Backend – App Reg Service
> Status: Complete

- Added `github.com/Azure/azure-sdk-for-go/sdk/azcore` as a direct dependency (promoted from indirect); no new SDK required
- Create `internal/services/appRegService.go`:
  - `GetConfigAppRegs() []models.CheckAppRegItem`: reads `APPREGISTRATION_N` env vars (each value is just an `appId`), populates `TenantId` from `AZURE_TENANT_ID` for display; returns one `CheckAppRegItem` per app
  - `CheckAppRegStatus(item models.CheckAppRegItem, warningDays int) (*models.AppRegCheckResult, error)`: calls `GET /applications/{objectId}` to fetch both `passwordCredentials` and `keyCredentials`; validates each; aggregates top-level `IsValid` and `ExpirationWarning`
  - Validity rules per credential: `endDateTime` past → invalid ("expired"); `startDateTime` in future → invalid ("not yet active"); `endDateTime` within `warningDays` → `ExpirationWarning=true`; no `endDateTime` → valid with `HasExpiration=false`
  - Mock vars `mockAppRegResult *models.AppRegCheckResult` and `mockAppRegListResult []models.CheckAppRegItem` for testability
- Add unit tests in `internal/services/appRegService_test.go`

### 3. Backend – API Handlers
> Status: Complete

- Add `AppRegList []models.CheckAppRegItem` and `AppRegWarningValidityDays int` to `Handlers` struct in `internal/handlers/handlers.go`
- Create `internal/handlers/appRegCheckHandlers.go`:
  - `GetAppRegList(w, r)` → returns `AppRegList` as JSON
  - `CheckAppRegStatus(w, r)` → accepts `?name=`, looks up in `AppRegList`, calls `services.CheckAppRegStatus`, returns `AppRegCheckResult` as JSON
- Add handler tests in `internal/handlers/appRegCheckHandlers_test.go`

### 4. Backend – Frontend Route Handlers
> Status: Complete

- Add to `internal/handlers/frontend.go`:
  - `GET /appregs-panel` → renders skeleton grid using `appRegItem.html` for each item in `AppRegList`
  - `GET /appreg-item?name=...` → calls `services.CheckAppRegStatus`, renders `appRegItemLoaded.html`
  - `GET /appreg-item-detail?name=...` → calls `services.CheckAppRegStatus`, renders `appRegItemModal.html`

### 5. Backend – Wire Up in main.go
> Status: Complete

- In `cmd/api/main.go`:
  - Add `getAppRegWarningValidityDays()` helper reading `APP_REG_WARNING_VALIDITY_DAYS` (default 30)
  - Call `services.GetConfigAppRegs()` on startup alongside the existing calls
  - Pass `appRegList` into `startWebServer`, `startJobs`, and `runOnce`
  - Set `handlers.AppRegList` and `handlers.AppRegWarningValidityDays` on the `Handlers` struct
  - Register routes: `GET /api/appreg-list`, `GET /api/check-appreg`, `GET /appregs-panel`, `GET /appreg-item`, `GET /appreg-item-detail`

### 6. Backend – Job Integration
> Status: Complete

- Extend `CheckCredJob` in `internal/jobs/CheckCredJob.go`:
  - Add `appRegList []models.CheckAppRegItem` and `appRegWarningDays int` fields
  - Update `Init(...)` signature to accept `appRegWarningDays int` and `appRegList []models.CheckAppRegItem`
  - Add `getAppRegNotification(*models.AppRegCheckResult) CheckNotification` method: uses `AppName` as `Name`; top-level `IsValid`/`ExpirationWarning`; per-credential issues are individual message lines (e.g. `"CI Deploy Key: expired"`)
  - In `execute()`, add an "App Registrations" `CheckNotificationGroup` loop alongside certs and secrets
- Update `CheckCredJob_test.go` with app reg job tests

### 7. Frontend – App Registrations Tab
> Status: Complete

- Update `frontend/body.html`:
  - Add "App Registrations" tab button (Tab 3) with same HTMX lazy-load pattern as Secrets tab (`hx-get="/appregs-panel"`, `hx-trigger="click once"`)
  - Add `panel-appregs` div (initially `display:none`)
  - Extend `showTab()` JS to include `'appregs'`
- Create `frontend/appRegItem.html` — loading skeleton card (mirrors `secretItem.html`; uses `/appreg-item?name=` HTMX trigger; name = `tenantId/appId`)
- Create `frontend/appRegItemLoaded.html` — loaded card with app name header (green/red icon) and per-credential rows (type icon + display name + expiry); clicking the card opens modal via `hx-get="/appreg-item-detail?name="`
- Create `frontend/appRegItemModal.html` — modal with app info section (Name, App ID, Tenant ID) followed by a credentials table (Type, Name, Valid From, Expires On, Status)

### 8. Documentation
> Status: Complete

- Update `readme.md`:
  - Add new env vars to the table:
    - `APPREGISTRATION_1..N` — `<appId>` values; all client secrets and certificates on each app registration are monitored. Tenant is derived from `AZURE_TENANT_ID`.
    - `APP_REG_WARNING_VALIDITY_DAYS` — days before expiry to trigger a warning (default: 30)
  - Add an "App Registration Monitoring" section explaining required Graph API permission (`Application.Read.All`), configuration format, and that both secrets and certificates are tracked
  - Update Features checklist to include App Registration credential monitoring
