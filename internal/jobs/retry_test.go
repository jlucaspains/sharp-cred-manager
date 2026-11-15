package jobs

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

type recordedSleeps struct {
	mu     sync.Mutex
	delays []time.Duration
}

func (r *recordedSleeps) sleep(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delays = append(r.delays, d)
}

func testRetryPolicy(r *recordedSleeps) retryPolicy {
	p := defaultRetryPolicy
	p.sleep = r.sleep
	return p
}

type scriptedResponse struct {
	status     int
	body       string
	retryAfter string
}

type scriptedServer struct {
	*httptest.Server
	mu       sync.Mutex
	attempts int
	bodies   []string
	auths    []string
}

// newScriptedServer replies with the given responses in order, repeating the
// last one once the script runs out.
func newScriptedServer(t *testing.T, responses ...scriptedResponse) *scriptedServer {
	s := &scriptedServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		idx := min(s.attempts, len(responses)-1)
		s.attempts++
		s.bodies = append(s.bodies, string(body))
		s.auths = append(s.auths, r.Header.Get("Authorization"))
		s.mu.Unlock()

		resp := responses[idx]
		if resp.retryAfter != "" {
			w.Header().Set("Retry-After", resp.retryAfter)
		}
		w.WriteHeader(resp.status)
		w.Write([]byte(resp.body))
	}))
	t.Cleanup(s.Close)
	return s
}

func postBody(url, body string) func() (*http.Request, error) {
	return func() (*http.Request, error) {
		return http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	}
}

func TestPostWithRetry_SuccessFirstAttempt(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, 1, ts.attempts)
	assert.Empty(t, sleeps.delays)
}

func TestPostWithRetry_429ThenSuccess(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusTooManyRequests}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, 2, ts.attempts)
	assert.Len(t, sleeps.delays, 1)
}

func TestPostWithRetry_503Exhausted(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusServiceUnavailable, body: "busy"})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.NotNil(t, err)
	assert.Equal(t, 3, ts.attempts)
	assert.Len(t, sleeps.delays, 2)
	assert.Equal(t, "error sending notification to Teams. Error 503 Service Unavailable: busy (after 3 attempts)", err.Error())
}

func TestPostWithRetry_400NotRetried(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusBadRequest})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Equal(t, "error sending notification to Teams. Error 400 Bad Request", err.Error())
	assert.Equal(t, 1, ts.attempts)
	assert.Empty(t, sleeps.delays)
}

func TestPostWithRetry_408ThenAccepted(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusRequestTimeout}, scriptedResponse{status: http.StatusAccepted})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, 2, ts.attempts)
}

func TestPostWithRetry_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close()

	attempts := 0
	newRequest := func() (*http.Request, error) {
		attempts++
		return http.NewRequest(http.MethodPost, url, strings.NewReader("x"))
	}
	sleeps := &recordedSleeps{}

	err := postWithRetry(&http.Client{}, newRequest, "Teams", testRetryPolicy(sleeps))

	assert.NotNil(t, err)
	assert.Equal(t, 3, attempts)
	assert.Len(t, sleeps.delays, 2)
	assert.Contains(t, err.Error(), "after 3 attempts")
}

func TestPostWithRetry_NewRequestErrorNotRetried(t *testing.T) {
	attempts := 0
	newRequest := func() (*http.Request, error) {
		attempts++
		return http.NewRequest(http.MethodPost, "://bad", nil)
	}
	sleeps := &recordedSleeps{}

	err := postWithRetry(&http.Client{}, newRequest, "Teams", testRetryPolicy(sleeps))

	assert.NotNil(t, err)
	assert.Equal(t, 1, attempts)
	assert.Empty(t, sleeps.delays)
}

func TestPostWithRetry_SameBodyEveryAttempt(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusInternalServerError})
	sleeps := &recordedSleeps{}
	payload := []byte(`{"hello":"world"}`)
	newRequest := func() (*http.Request, error) {
		return http.NewRequest(http.MethodPost, ts.URL, bytes.NewReader(payload))
	}

	postWithRetry(ts.Client(), newRequest, "Teams", testRetryPolicy(sleeps))

	assert.Equal(t, []string{string(payload), string(payload), string(payload)}, ts.bodies)
}

func TestPostWithRetry_RetryAfterSeconds(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusTooManyRequests, retryAfter: "5"}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, []time.Duration{5 * time.Second}, sleeps.delays)
}

func TestPostWithRetry_RetryAfterCapped(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusServiceUnavailable, retryAfter: "120"}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, []time.Duration{30 * time.Second}, sleeps.delays)
}

func TestPostWithRetry_RetryAfterGarbageFallsBack(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusTooManyRequests, retryAfter: "garbage"}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Len(t, sleeps.delays, 1)
	assert.GreaterOrEqual(t, sleeps.delays[0], 1*time.Second)
	assert.LessOrEqual(t, sleeps.delays[0], 1200*time.Millisecond)
}

func TestPostWithRetry_BackoffDelays(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusBadGateway})
	sleeps := &recordedSleeps{}

	postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Len(t, sleeps.delays, 2)
	assert.GreaterOrEqual(t, sleeps.delays[0], 1*time.Second)
	assert.LessOrEqual(t, sleeps.delays[0], 1200*time.Millisecond)
	assert.GreaterOrEqual(t, sleeps.delays[1], 2*time.Second)
	assert.LessOrEqual(t, sleeps.delays[1], 2400*time.Millisecond)
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

	d, ok := parseRetryAfter("5", now)
	assert.True(t, ok)
	assert.Equal(t, 5*time.Second, d)

	d, ok = parseRetryAfter("Wed, 11 Nov 2026 12:00:10 GMT", now)
	assert.True(t, ok)
	assert.Equal(t, 10*time.Second, d)

	_, ok = parseRetryAfter("Wed, 11 Nov 2026 11:59:00 GMT", now)
	assert.False(t, ok)

	_, ok = parseRetryAfter("-1", now)
	assert.False(t, ok)

	_, ok = parseRetryAfter("", now)
	assert.False(t, ok)

	_, ok = parseRetryAfter("garbage", now)
	assert.False(t, ok)
}

func TestIsRetryableStatus(t *testing.T) {
	for _, code := range []int{408, 429, 500, 502, 503, 504} {
		assert.True(t, isRetryableStatus(code), code)
	}
	for _, code := range []int{400, 401, 403, 404, 409} {
		assert.False(t, isRetryableStatus(code), code)
	}
}

func TestTeamsSlackNotifier_RetriesWithSameBody(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusTooManyRequests}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	notifier := &TeamsSlackNotifier{retryPolicy: testRetryPolicy(sleeps)}
	notifier.Init(Slack, ts.URL, "title", "body", "", "")
	err := notifier.Notify([]CheckNotificationGroup{
		{Label: "Certificates", Items: []CheckNotification{{Name: "host1", IsValid: true}}},
	})

	assert.Nil(t, err)
	assert.Equal(t, 2, ts.attempts)
	assert.NotEmpty(t, ts.bodies[0])
	assert.Equal(t, ts.bodies[0], ts.bodies[1])
}

func TestGenericWebhookNotifier_RetriesWithAuth(t *testing.T) {
	tests := []struct {
		name     string
		authType string
		token    string
		user     string
		pass     string
		expected string
	}{
		{"bearer", "bearer", "my-token", "", "", "Bearer my-token"},
		{"basic", "basic", "", "user", "pass", "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newScriptedServer(t, scriptedResponse{status: http.StatusInternalServerError}, scriptedResponse{status: http.StatusOK})
			sleeps := &recordedSleeps{}

			n := &GenericWebhookNotifier{retryPolicy: testRetryPolicy(sleeps)}
			n.Init(ts.URL, tt.authType, tt.token, tt.user, tt.pass)
			err := n.NotifyRaw([]*models.CertCheckResult{}, []*models.SecretCheckResult{}, []*models.AppRegCheckResult{})

			assert.Nil(t, err)
			assert.Equal(t, 2, ts.attempts)
			assert.Equal(t, []string{tt.expected, tt.expected}, ts.auths)
			assert.Equal(t, ts.bodies[0], ts.bodies[1])
		})
	}
}

func TestParseRetryAfter_HugeSecondsDoesNotOverflow(t *testing.T) {
	for _, h := range []string{"9223372037", "10000000000", "99999999999999999"} {
		d, ok := parseRetryAfter(h, time.Now())
		assert.True(t, ok, h)
		assert.Greater(t, d, time.Duration(0), h)
	}
}

func TestPostWithRetry_HugeRetryAfterIsCapped(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusTooManyRequests, retryAfter: "10000000000"}, scriptedResponse{status: http.StatusOK})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.Nil(t, err)
	assert.Equal(t, []time.Duration{30 * time.Second}, sleeps.delays)
}

func TestPostWithRetry_NetworkErrorOmitsURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL + "/webhook/secret-token"
	ts.Close()
	sleeps := &recordedSleeps{}

	err := postWithRetry(&http.Client{}, postBody(url, "x"), "Teams", testRetryPolicy(sleeps))

	assert.NotNil(t, err)
	assert.NotContains(t, err.Error(), "secret-token")
	assert.Contains(t, err.Error(), "error sending notification to Teams:")
	assert.Contains(t, err.Error(), "after 3 attempts")
}

func TestPostWithRetry_ErrorBodyIsTruncated(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusBadRequest, body: strings.Repeat("a", maxErrorBodyBytes+500)})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.NotNil(t, err)
	assert.True(t, strings.HasSuffix(err.Error(), strings.Repeat("a", 10)+"... (truncated)"))
	assert.NotContains(t, err.Error(), strings.Repeat("a", maxErrorBodyBytes+1))
}

func TestPostWithRetry_ErrorBodyAtLimitNotTruncated(t *testing.T) {
	ts := newScriptedServer(t, scriptedResponse{status: http.StatusBadRequest, body: strings.Repeat("a", maxErrorBodyBytes)})
	sleeps := &recordedSleeps{}

	err := postWithRetry(ts.Client(), postBody(ts.URL, "x"), "Teams", testRetryPolicy(sleeps))

	assert.NotContains(t, err.Error(), "truncated")
}
