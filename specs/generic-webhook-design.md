# Generic Webhook Notifier — Design Spec

**Date:** 2026-05-03
**Issue:** [#187](https://github.com/jlucaspains/sharp-cert-manager/issues/187) — Additional notification integrations (item 1)

---

## Overview

Add a `generic` webhook notifier type that POSTs a structured JSON payload to any HTTP endpoint with optional Bearer or Basic authentication. Unlike the Teams/Slack notifiers which render platform-specific templates, the generic notifier sends raw check result data so any consumer can process it without parsing human-readable strings.

---

## Architecture

Three changes to the codebase:

1. **Rename** `WebHookNotifier` → `TeamsSlackNotifier` (file: `webHookNotifier.go` → `teamsSlackNotifier.go`). Update all references in `notificationTemplates.go`, `main.go`, and tests.
2. **New file** `internal/jobs/genericWebhookNotifier.go` — implements the `Notifier` interface (and the new `RawNotifier` interface) with struct-marshalled JSON and auth header support.
3. **Update `main.go`** — `getJobNotifier()` returns a `GenericWebhookNotifier` when `WEBHOOK_TYPE=generic`, otherwise a `TeamsSlackNotifier` as today.

---

## New Interface: `RawNotifier`

```go
type RawNotifier interface {
    Notifier
    NotifyRaw(
        certs   []*models.CertCheckResult,
        secrets []*models.SecretCheckResult,
        appRegs []*models.AppRegCheckResult,
    ) error
}
```

`GenericWebhookNotifier` implements both `Notifier` (where `Notify` is a no-op) and `RawNotifier`.

---

## Data Flow

`CheckCredJob.execute()` is restructured to collect raw results first, then branch on notifier type:

```go
func (c *CheckCredJob) execute() {
    certResults   := c.checkCerts()
    secretResults := c.checkSecrets()
    appRegResults := c.checkAppRegs()

    if raw, ok := c.notifier.(RawNotifier); ok {
        if err := raw.NotifyRaw(certResults, secretResults, appRegResults); err != nil {
            log.Printf("Error sending notification: %s", err)
        }
        return
    }

    groups := c.buildGroups(certResults, secretResults, appRegResults)
    if err := c.notifier.Notify(groups); err != nil {
        log.Printf("Error sending notification: %s", err)
    }
}
```

The existing `buildCertGroup`, `buildSecretGroup`, and `buildAppRegGroup` methods are refactored to accept pre-fetched results rather than fetching internally, eliminating duplicate check calls.

Level filtering is applied in both paths before the notifier is called. For the `RawNotifier` path, filtering is applied directly on raw model fields (`IsValid`, `ExpirationWarning`) using the same logic as `shouldNotify`:
- Certs and secrets: include the result if it passes the level filter.
- App registrations: filter at credential level within each `AppRegCheckResult`; exclude app regs with no passing credentials.

---

## JSON Payload Schema

`Content-Type: application/json`

```json
{
  "certificates": [
    {
      "hostname": "example.com",
      "displayName": "example.com",
      "source": "https://example.com",
      "issuer": "Let's Encrypt",
      "signature": "...",
      "certStartDate": "2025-01-01T00:00:00Z",
      "certEndDate": "2025-04-01T00:00:00Z",
      "certDnsNames": ["example.com", "www.example.com"],
      "isValid": true,
      "tlsVersion": 772,
      "isCA": false,
      "commonName": "example.com",
      "validationIssues": [],
      "expirationWarning": true,
      "validityInDays": 28
    }
  ],
  "secrets": [
    {
      "name": "my-secret",
      "displayName": "my-secret",
      "source": "https://myvault.vault.azure.net",
      "url": "https://myvault.vault.azure.net/secrets/my-secret",
      "contentType": "text/plain",
      "enabled": true,
      "expiresOn": "2025-04-01T00:00:00Z",
      "notBefore": null,
      "isValid": true,
      "validationIssues": [],
      "expirationWarning": false,
      "hasExpiration": true,
      "validityInDays": 60
    }
  ],
  "appRegistrations": [
    {
      "name": "my-app",
      "appName": "My App",
      "appId": "...",
      "appObjectId": "...",
      "isValid": true,
      "expirationWarning": false,
      "credentials": [
        {
          "keyId": "...",
          "displayName": "client-secret-1",
          "credentialType": 0,
          "startDateTime": "2024-01-01T00:00:00Z",
          "endDateTime": "2026-01-01T00:00:00Z",
          "isValid": true,
          "validationIssues": [],
          "expirationWarning": false,
          "hasExpiration": true,
          "validityInDays": 245
        }
      ]
    }
  ]
}
```

Arrays are empty (`[]`) when no items of that type are monitored or pass the level filter — never `null`.

---

## Configuration

All existing env vars continue to work unchanged. New vars (only read when `WEBHOOK_TYPE=generic`):

| Env var | Required | Values | Description |
|---|---|---|---|
| `WEBHOOK_TYPE` | yes | `generic` | Selects this notifier |
| `WEBHOOK_URL` | yes | any URL | Endpoint to POST to |
| `WEBHOOK_AUTH_TYPE` | no | `bearer`, `basic` | Auth method; omit for no auth |
| `WEBHOOK_AUTH_TOKEN` | no | any string | Bearer token (used when `WEBHOOK_AUTH_TYPE=bearer`) |
| `WEBHOOK_AUTH_USERNAME` | no | any string | Username (used when `WEBHOOK_AUTH_TYPE=basic`) |
| `WEBHOOK_AUTH_PASSWORD` | no | any string | Password (used when `WEBHOOK_AUTH_TYPE=basic`) |

`IsReady()` returns `true` when `WEBHOOK_URL` is non-empty.

Auth header values:
- `bearer` → `Authorization: Bearer <WEBHOOK_AUTH_TOKEN>`
- `basic` → `Authorization: Basic <base64(username:password)>`

---

## Error Handling

Mirrors `TeamsSlackNotifier` behaviour:

- Non-2xx HTTP response → return error including status and response body
- Network / client error → return error as-is
- JSON marshal failure → return error
- All errors surface to `execute()` which logs via `log.Printf`

No retries, no fallback.

---

## Files Changed

| File | Change |
|---|---|
| `internal/jobs/webHookNotifier.go` → `teamsSlackNotifier.go` | Rename, update type/struct names |
| `internal/jobs/webHookNotifier_test.go` → `teamsSlackNotifier_test.go` | Rename, update references |
| `internal/jobs/notificationTemplates.go` | Update `NotifierType` constants and `Notifiers` map |
| `internal/jobs/CheckCredJob.go` | Add `RawNotifier` interface; refactor `execute()` and build methods |
| `internal/jobs/CheckCredJob_test.go` | Add test for `RawNotifier` dispatch |
| `internal/jobs/genericWebhookNotifier.go` | New file |
| `internal/jobs/genericWebhookNotifier_test.go` | New file |
| `cmd/api/main.go` | Update `getJobNotifier()` to handle `generic` type |

---

## Tests

**`genericWebhookNotifier_test.go`:**
- `TestGenericWebhookNotifier_Notify_Bearer` — verifies `Authorization: Bearer <token>` header
- `TestGenericWebhookNotifier_Notify_Basic` — verifies `Authorization: Basic <encoded>` header
- `TestGenericWebhookNotifier_Notify_NoAuth` — verifies no `Authorization` header when type is absent
- `TestGenericWebhookNotifier_Notify_ErrorResponse` — verifies non-2xx returns an error
- `TestGenericWebhookNotifier_IsReady` — `true` with URL, `false` without
- `TestGenericWebhookNotifier_Payload` — verifies JSON body contains all three typed arrays with full model fields

**`CheckCredJob_test.go` addition:**
- `TestCheckCredJob_UsesRawNotifier` — verifies `NotifyRaw` is called (not `Notify`) when notifier implements `RawNotifier`
