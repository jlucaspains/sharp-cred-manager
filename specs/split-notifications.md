# Split Notifications and Per-Type Levels — Design Spec

**Date:** 2026-11-11
**Issue:** [#210](https://github.com/jlucaspains/sharp-cert-manager/issues/210) — Option to send multiple messages

---

## Overview

Today the check job sends one combined notification that holds certificates, secrets and app registrations, all filtered by a single level (`CHECK_CRED_JOB_NOTIFICATION_LEVEL`). This spec adds two features that can be used separately:

1. **Per-type notification levels.** Each credential type can override the global level.
2. **Split mode.** The job sends one notification per credential type instead of one combined message.

The default behavior does not change. Existing deployments with no new env vars set get exactly the message they get today.

---

## Key Decisions

- **The two features are independent.** Per-type levels also apply in combined mode. Split mode works with or without overrides.
- **Mode is an enum, not a boolean:** `CHECK_CRED_JOB_NOTIFICATION_MODE=combined|split`. This leaves room for future modes, such as routing each type to its own webhook.
- **Title in split mode** is unchanged: each message uses the configured `MESSAGE_TITLE` (or its default). The group heading inside each message already shows the credential type.
- **Skipping in split mode:** no message is sent for a type that has no items configured, or for a type with no items left after level filtering. If nothing qualifies, split mode sends nothing. Combined mode keeps its current behavior and always sends.
- **Generic webhook** supports both features. In split mode it makes one `NotifyRaw` call per type. The payload shape stays the same: the other lists are sent as empty arrays.
- **Partial failure:** in split mode a failed send does not stop the remaining sends. Errors are combined with `errors.Join` and logged once.
- **Out of scope:** per-type webhook URLs and per-type message titles or bodies.

---

## Configuration

| Variable | Description | Default |
|---|---|---|
| `CHECK_CRED_JOB_NOTIFICATION_MODE` | `combined` sends one message with all types. `split` sends one message per credential type. | `combined` |
| `CERT_NOTIFICATION_LEVEL` | Minimum notification level for certificates (`Info`, `Warning`, `Error`). Overrides `CHECK_CRED_JOB_NOTIFICATION_LEVEL`. | global level |
| `SECRET_NOTIFICATION_LEVEL` | Minimum notification level for Key Vault secrets. Overrides the global level. | global level |
| `APP_REG_NOTIFICATION_LEVEL` | Minimum notification level for app registration credentials. Overrides the global level. | global level |

Level resolution for each type:

1. The type-specific variable, if set and valid.
2. Otherwise `CHECK_CRED_JOB_NOTIFICATION_LEVEL`, if set and valid.
3. Otherwise `Warning` (as today).

An invalid override value (for example `CERT_NOTIFICATION_LEVEL=Loud`) logs a warning at `Init` and falls back to the global level. An invalid mode value logs a warning and falls back to `combined`.

---

## Code Changes

### `internal/jobs/CheckCredJob.go`

**Types**

```go
type NotificationMode int

const (
    Combined NotificationMode = iota
    Split
)

var modes = map[string]NotificationMode{
    "combined": Combined,
    "split":    Split,
}
```

Mode parsing is case-insensitive. Level parsing keeps the existing `levels` map (`Info`, `Warning`, `Error`).

**Config**: `CheckCredJobConfig` gains:

```go
Mode        string
CertLevel   string
SecretLevel string
AppRegLevel string
```

`Level` stays and acts as the global fallback.

**Job state**: replace `level Level` with `certLevel`, `secretLevel` and `appRegLevel` (type `Level`), and add `mode NotificationMode`. Add a helper `resolveLevel(override, global string) Level` that applies the resolution rules above.

**Filtering**: `shouldNotify` takes the level explicitly:

```go
func shouldNotify(level Level, model CheckNotification) bool {
    return level == Info || !model.IsValid || (level == Warning && model.ExpirationWarning)
}
```

`buildCertGroup` and `filterCertResults` use `c.certLevel`. The secret and app registration functions use `c.secretLevel` and `c.appRegLevel` the same way.

**Notifier interface**: unchanged. `Notify(groups []CheckNotificationGroup) error` and `RawNotifier.NotifyRaw` keep their current signatures.

**`execute()`**

```go
func (c *CheckCredJob) execute() {
    certs := c.checkCerts()
    secrets := c.checkSecrets()
    appRegs := c.checkAppRegs()

    var err error
    if raw, ok := c.notifier.(RawNotifier); ok {
        err = c.notifyRaw(raw, certs, secrets, appRegs)
    } else {
        err = c.notifyGroups(certs, secrets, appRegs)
    }

    if err != nil {
        log.Printf("Error sending notification: %s", err)
    }
}
```

- `notifyGroups` builds the three groups as today.
  - **Combined:** one `Notify(nonEmptyGroups)` call, the same as current behavior.
  - **Split:** for each group with `len(Items) > 0`, call `Notify([]CheckNotificationGroup{group})`, then return `errors.Join(errs...)`.
- `notifyRaw` filters the three result lists.
  - **Combined:** one `NotifyRaw(certs, secrets, appRegs)` call, the same as today.
  - **Split:** one call per non-empty filtered list, with empty slices for the other two. Then return `errors.Join(errs...)`.

Sends happen in a fixed order: Certificates, then Secrets, then App Registrations.

### `cmd/api/main.go`

`startJobs` and `runOnce` build the same config twice. Move that into one helper:

```go
func getCheckCredJobConfig(schedule string, certList, secretList, appRegList) jobs.CheckCredJobConfig
```

The helper reads `CHECK_CRED_JOB_NOTIFICATION_LEVEL`, `CHECK_CRED_JOB_NOTIFICATION_MODE`, `CERT_NOTIFICATION_LEVEL`, `SECRET_NOTIFICATION_LEVEL` and `APP_REG_NOTIFICATION_LEVEL`.

### `readme.md`

- Add the four new variables to the env var table.
- Fix the `CHECK_CRED_JOB_NOTIFICATION_LEVEL` description. It currently says "(cert and secret)"; it should say it is the global default for all credential types and can be overridden per type.
- Add a short "Notification modes" section with a split-mode example.

---

## Testing

All new tests go in `internal/jobs/CheckCredJob_test.go` and the notifier test files, using the existing mock notifier and `httptest` patterns.

**Level resolution**
- No overrides: all types use the global level.
- An override set for one type: only that type changes.
- An invalid override: falls back to the global level.
- Nothing set: `Warning`.

**Filtering**
- Certificates at `Info` and secrets at `Error`. A valid cert and an expiring-but-valid secret produce one cert item and no secret items.

**Combined mode**
- Makes one `Notify` call. This guards against regressions.
- Per-type levels apply inside the combined message.

**Split mode (Teams/Slack)**
- Three types with qualifying items produce three `Notify` calls in a fixed order. Each call has one group with the matching label.
- A type with no qualifying items produces no call for that type.
- No qualifying items at all produces zero calls.
- The first send fails: the remaining sends still happen and the returned error includes the failure.

**Split mode (generic)**
- Each `NotifyRaw` call has exactly one non-empty list.

**Mode parsing**
- `split`, `Split` and `SPLIT` all resolve to split mode.
- An invalid value falls back to combined.

---

## Rollout

The change is backward compatible: the default mode is `combined`, and level overrides fall back to the global level. No Go interface signatures change.
