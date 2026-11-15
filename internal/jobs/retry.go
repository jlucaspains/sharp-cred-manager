package jobs

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const webhookClientTimeout = 30 * time.Second

// maxErrorBodyBytes limits how much of an error response body is read and
// included in the returned error (and therefore in the logs).
const maxErrorBodyBytes = 4 * 1024

type retryPolicy struct {
	maxAttempts   int                 // total attempts, counting the first send
	baseDelay     time.Duration       // first backoff wait, doubled after each failure
	maxRetryAfter time.Duration       // cap applied to Retry-After values
	jitter        float64             // max random extra time as a fraction of the delay
	sleep         func(time.Duration) // time.Sleep in production; replaced in tests
}

var defaultRetryPolicy = retryPolicy{
	maxAttempts:   3,
	baseDelay:     1 * time.Second,
	maxRetryAfter: 30 * time.Second,
	jitter:        0.2,
	sleep:         time.Sleep,
}

// resolveRetryPolicy returns p, or defaultRetryPolicy when p is the zero value.
func resolveRetryPolicy(p retryPolicy) retryPolicy {
	if p.maxAttempts <= 0 {
		return defaultRetryPolicy
	}
	if p.sleep == nil {
		p.sleep = time.Sleep
	}
	return p
}

// postWithRetry sends a request built by newRequest, retrying on retryable
// failures. newRequest is called once per attempt so that the request body is
// rebuilt each time. It returns nil on 200/202, otherwise the last error.
func postWithRetry(client *http.Client, newRequest func() (*http.Request, error), target string, p retryPolicy) error {
	p = resolveRetryPolicy(p)

	for attempt := 1; ; attempt++ {
		req, err := newRequest()
		if err != nil {
			return err
		}

		var retryable bool
		var retryAfter time.Duration
		var hasRetryAfter bool

		response, err := client.Do(req)
		if err != nil {
			err = redactURLError(err, target)
			retryable = true
		} else {
			if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusAccepted {
				response.Body.Close()
				return nil
			}

			body := readErrorBody(response.Body)
			response.Body.Close()

			if body != "" {
				err = fmt.Errorf("error sending notification to %s. Error %s: %s", target, response.Status, body)
			} else {
				err = fmt.Errorf("error sending notification to %s. Error %s", target, response.Status)
			}

			retryable = isRetryableStatus(response.StatusCode)
			if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
				retryAfter, hasRetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
			}
		}

		if !retryable || attempt >= p.maxAttempts {
			if attempt > 1 {
				return fmt.Errorf("%w (after %d attempts)", err, attempt)
			}
			return err
		}

		var delay time.Duration
		if hasRetryAfter {
			delay = min(retryAfter, p.maxRetryAfter)
		} else {
			delay = backoffDelay(attempt, p)
		}

		log.Printf("Notification to %s failed (attempt %d/%d), retrying in %s: %s", target, attempt, p.maxAttempts, delay, err)
		p.sleep(delay)
	}
}

func redactURLError(err error, target string) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("error sending notification to %s: %w", target, urlErr.Err)
	}
	return err
}

func readErrorBody(r io.Reader) string {
	body, _ := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes+1))
	if len(body) > maxErrorBodyBytes {
		return string(body[:maxErrorBodyBytes]) + "... (truncated)"
	}
	return string(body)
}

func isRetryableStatus(code int) bool {
	return code == http.StatusRequestTimeout || code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

// parseRetryAfter parses a Retry-After header in either delay-seconds or
// HTTP-date form. It returns false for missing, invalid or negative values.
func parseRetryAfter(header string, now time.Time) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds < 0 {
			return 0, false
		}
		// Avoid overflowing time.Duration; callers cap the value anyway.
		if int64(seconds) > math.MaxInt64/int64(time.Second) {
			return time.Duration(math.MaxInt64), true
		}
		return time.Duration(seconds) * time.Second, true
	}

	if date, err := http.ParseTime(header); err == nil {
		delay := date.Sub(now)
		if delay < 0 {
			return 0, false
		}
		return delay, true
	}

	return 0, false
}

// backoffDelay returns baseDelay * 2^(attempt-1) plus up to jitter extra time.
func backoffDelay(attempt int, p retryPolicy) time.Duration {
	delay := p.baseDelay << (attempt - 1)
	if p.jitter > 0 {
		delay += time.Duration(rand.Float64() * p.jitter * float64(delay))
	}
	return delay
}
