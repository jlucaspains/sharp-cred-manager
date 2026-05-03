package jobs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestGenericWebhookNotifier_IsReady_WithUrl(t *testing.T) {
	n := &GenericWebhookNotifier{}
	n.Init("https://example.com/webhook", "", "", "", "")
	assert.True(t, n.IsReady())
}

func TestGenericWebhookNotifier_IsReady_WithoutUrl(t *testing.T) {
	n := &GenericWebhookNotifier{}
	n.Init("", "", "", "", "")
	assert.False(t, n.IsReady())
}

func TestGenericWebhookNotifier_NoAuth(t *testing.T) {
	var capturedAuth string
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})

	n := &GenericWebhookNotifier{}
	n.Init(ts.URL, "", "", "", "")
	err := n.NotifyRaw([]*models.CertCheckResult{}, []*models.SecretCheckResult{}, []*models.AppRegCheckResult{})

	assert.Nil(t, err)
	assert.Empty(t, capturedAuth)
}

func TestGenericWebhookNotifier_BearerAuth(t *testing.T) {
	var capturedAuth string
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})

	n := &GenericWebhookNotifier{}
	n.Init(ts.URL, "bearer", "my-token", "", "")
	err := n.NotifyRaw([]*models.CertCheckResult{}, []*models.SecretCheckResult{}, []*models.AppRegCheckResult{})

	assert.Nil(t, err)
	assert.Equal(t, "Bearer my-token", capturedAuth)
}

func TestGenericWebhookNotifier_BasicAuth(t *testing.T) {
	var capturedAuth string
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	})

	n := &GenericWebhookNotifier{}
	n.Init(ts.URL, "basic", "", "user", "pass")
	err := n.NotifyRaw([]*models.CertCheckResult{}, []*models.SecretCheckResult{}, []*models.AppRegCheckResult{})

	assert.Nil(t, err)
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
	assert.Equal(t, expected, capturedAuth)
}

func TestGenericWebhookNotifier_ErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	n := &GenericWebhookNotifier{}
	n.Init(ts.URL, "", "", "", "")
	err := n.NotifyRaw([]*models.CertCheckResult{}, []*models.SecretCheckResult{}, []*models.AppRegCheckResult{})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "400")
}

func TestGenericWebhookNotifier_Payload(t *testing.T) {
	var capturedBody []byte
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		capturedBody = buf.Bytes()
	})

	certEndDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	certs := []*models.CertCheckResult{
		{
			Hostname:          "example.com",
			DisplayName:       "example.com",
			Source:            "https://example.com",
			Issuer:            "Let's Encrypt",
			CertEndDate:       certEndDate,
			IsValid:           true,
			ExpirationWarning: false,
			ValidityInDays:    29,
		},
	}

	expiresOn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	secrets := []*models.SecretCheckResult{
		{
			DisplayName:    "my-secret",
			Source:         "https://myvault.vault.azure.net",
			IsValid:        true,
			HasExpiration:  true,
			ExpiresOn:      &expiresOn,
			ValidityInDays: 60,
		},
	}

	appRegs := []*models.AppRegCheckResult{
		{
			AppName: "MyApp",
			AppId:   "app-123",
			IsValid: true,
			Credentials: []models.AppRegCredentialResult{
				{
					DisplayName:    "client-secret",
					CredentialType: models.AppRegCredentialSecret,
					IsValid:        true,
					ValidityInDays: 90,
				},
			},
		},
	}

	n := &GenericWebhookNotifier{}
	n.Init(ts.URL, "", "", "", "")
	err := n.NotifyRaw(certs, secrets, appRegs)

	assert.Nil(t, err)

	var payload GenericWebhookPayload
	assert.Nil(t, json.Unmarshal(capturedBody, &payload))
	assert.Equal(t, 1, len(payload.Certificates))
	assert.Equal(t, "example.com", payload.Certificates[0].DisplayName)
	assert.Equal(t, 29, payload.Certificates[0].ValidityInDays)
	assert.Equal(t, 1, len(payload.Secrets))
	assert.Equal(t, "my-secret", payload.Secrets[0].DisplayName)
	assert.Equal(t, 60, payload.Secrets[0].ValidityInDays)
	assert.Equal(t, 1, len(payload.AppRegistrations))
	assert.Equal(t, "MyApp", payload.AppRegistrations[0].AppName)
	assert.Equal(t, 1, len(payload.AppRegistrations[0].Credentials))
	assert.Equal(t, 90, payload.AppRegistrations[0].Credentials[0].ValidityInDays)
}
