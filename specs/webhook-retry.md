# Webhook Retry with Exponential Backoff — Design Spec

**Date:** 2026-11-11
**Related:** [#210](https://github.com/jlucaspains/sharp-cert-manager/issues/210) (split notifications). Split mode can send up to three posts back to back to the same webhook, which raises the risk of being rate limited.

---

## Overview

Today a failed webhook post is logged and dropped. Nothing is resent until the next cron run. This spec adds a bounded retry with exponential backoff to every outgoing webhook post (Teams, Slack and generic), in both combined and split modes.

---

## Key Decisions

- **Attempts:** at most 3 per message, counting the first send.
- **Retry on:**
  - network errors from the HTTP client (refused or reset connections, DNS failures, client timeouts)
  - HTTP `408 Request Timeout`
  - HTTP `429 Too Many Requests`
  - HTTP `5xx`
- **No retry on** any other non-success status, such as `400`, `401`, `403` or `404`. These point to a configuration or payload problem that a retry won't fix.
- **Success** stays defined as `200` or `202`, the same as today.
- **Backoff:** start at 1s and double after each failure, so the waits are 1s and then 2s. Up to 20% random extra time is added to each wait.
- **`Retry-After`:** on a 429 or 5xx response with a `Retry-After` header, the header value is used instead of the backoff delay. Both the seconds form and the HTTP-date form are supported, and the value is capped at **30s**. A missing, invalid or negative value falls back to the backoff delay.
- **HTTP client timeout:** both notifiers set a **30s** `http.Client.Timeout`. Without it, a webhook that hangs blocks the job forever and the retry never runs.
- **Duplicates are accepted:** if the service handled a post but the response was lost, the retry sends a second copy. Teams and Slack can't drop repeated posts. We accept the rare duplicate in exchange for recovering from network blips.
- **Not configurable:** attempts, delays and caps are fixed in code. There are no env vars for them for now.
- **Out of scope:** pausing between split-mode sends, and changing how mentions work in split mode.

---

## Code Changes

### New file: `internal/jobs/retry.go`

```go
type retryPolicy struct {
    maxAttempts   int                 // 3
    baseDelay     time.Duration       // 1s
    maxRetryAfter time.Duration       // 30s
    jitter        float64             // 0.2, i.e. up to 20% extra
    sleep         func(time.Duration) // time.Sleep in production; replaced in tests
}

var defaultRetryPolicy = retryPolicy{...}

// postWithRetry sends a request built by newRequest, retrying on retryable
// failures. newRequest is called once per attempt so that the request body is
// rebuilt each time. It returns nil on 200/202, otherwise the last error.
func postWithRetry(client *http.Client, newRequest func() (*http.Request, error), target string, p retryPolicy) error
```

Behavior:

1. For each attempt from 1 to `maxAttempts`:
   - Build a new request with `newRequest()`. If building fails, return that error immediately without retrying.
   - Send it with `client.Do(req)`.
   - **Network error:** record it. The error is retryable.
   - **200 or 202:** close the body and return `nil`.
   - **Any other status:** read the body, close it, and build the same error message the notifiers produce today: `error sending notification to <target>. Error <status>[: <body>]`. The error is retryable only for 408, 429 and 5xx.
2. If the error isn't retryable, or this was the last attempt, return it. When more than one attempt was made, add a note to the message, for example `(after 3 attempts)`.
3. Otherwise wait: use the parsed `Retry-After` value capped at `maxRetryAfter`, or `baseDelay * 2^(attempt-1)` plus random extra time. Log `Notification to <target> failed (attempt n/3), retrying in <delay>: <err>`, then call `p.sleep(delay)`.

Helpers:
- `isRetryableStatus(code int) bool`
- `parseRetryAfter(header string, now time.Time) (time.Duration, bool)`
- `backoffDelay(attempt int, p retryPolicy) time.Duration`

### `internal/jobs/teamsSlackNotifier.go`

- Add a `retryPolicy` field, set to `defaultRetryPolicy` when it is empty.
- Render the template into a `[]byte` once. The `newRequest` function builds `http.NewRequest(POST, url, bytes.NewReader(body))` with `Content-Type: application/json` on every attempt.
- Replace the inline status and body handling with `postWithRetry(..., string(m.NotifierType), m.retryPolicy)`.
- `getClient()` returns `&http.Client{Timeout: 30 * time.Second}`.

### `internal/jobs/genericWebhookNotifier.go`

- Add the same `retryPolicy` field.
- Marshal the payload once. The `newRequest` function builds the request and sets the `Content-Type` and auth headers (bearer or basic) on every attempt.
- Replace the inline status handling with `postWithRetry(..., "generic webhook", g.retryPolicy)`.
- `getClient()` sets the same 30s timeout.

### `internal/jobs/CheckCredJob.go`

No changes. Retries happen inside each `Notify` and `NotifyRaw` call, so `sendAll` still sees a single final error for each credential type.

---

## Worst-Case Timing

| Scenario | Added delay per message |
|---|---|
| Throttled with no `Retry-After` | about 3s (1s + 2s, plus up to 20% extra) |
| Throttled with `Retry-After` at or above the cap | 60s (30s + 30s) |
| Hanging endpoint | up to 90s of request time (3 × 30s timeout) plus about 3s of waiting |

In split mode with all three types affected, multiply by 3. That is acceptable for a cron job that runs at most once a minute and usually once a day. Runs don't overlap: `tryExecute` runs inside the ticker loop, so the next check waits until the current run finishes.

---

## Testing

New file `internal/jobs/retry_test.go`. Tests use `httptest` servers and replace `sleep` with a function that records the delays and returns immediately.

**`postWithRetry`**
- 200 on the first attempt: 1 attempt, no wait.
- 429 then 200: 2 attempts, succeeds.
- 503, 503, 503: 3 attempts, returns an error containing the status, the body and `after 3 attempts`.
- 400: 1 attempt, returns the error right away.
- 408 then 202: retried and succeeds.
- Network error: closing the server makes 3 attempts and returns an error.
- The request body is identical on every attempt.
- `Retry-After: 5` gives a 5s wait.
- `Retry-After: 120` is capped at 30s.
- `Retry-After: garbage` falls back to the backoff delay.
- An HTTP-date `Retry-After` value is parsed against a fixed `now`.
- Backoff waits are 1s and then 2s, each within +20%.

**Notifiers**
- Teams/Slack: a 429 then a 200 succeeds and the body is the same on both attempts.
- Generic: a 500 then a 200 succeeds, and the `Authorization` header is present on both attempts for both bearer and basic auth.
- Existing tests keep passing. Their single-attempt error messages don't change.
- Notifier tests that hit retryable errors use a policy with a no-op `sleep` so they finish quickly.

---

## Rollout

Backward compatible with no configuration changes. The only difference users will see is that failed posts are retried and logged, and in the worst case a run takes longer.
