package jobs

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
)

type GenericWebhookNotifier struct {
	WebhookUrl   string
	AuthType     string
	AuthToken    string
	AuthUsername string
	AuthPassword string
	httpClient   *http.Client
	retryPolicy  retryPolicy
}

type GenericWebhookPayload struct {
	Certificates     []models.CertCheckResult   `json:"certificates"`
	Secrets          []models.SecretCheckResult  `json:"secrets"`
	AppRegistrations []models.AppRegCheckResult  `json:"appRegistrations"`
}

func (g *GenericWebhookNotifier) Init(webhookUrl, authType, authToken, authUsername, authPassword string) {
	g.WebhookUrl = webhookUrl
	g.AuthType = authType
	g.AuthToken = authToken
	g.AuthUsername = authUsername
	g.AuthPassword = authPassword
}

func (g *GenericWebhookNotifier) Notify(_ []CheckNotificationGroup) error {
	return nil
}

func (g *GenericWebhookNotifier) NotifyRaw(certs []*models.CertCheckResult, secrets []*models.SecretCheckResult, appRegs []*models.AppRegCheckResult) error {
	payload := GenericWebhookPayload{
		Certificates:     make([]models.CertCheckResult, 0, len(certs)),
		Secrets:          make([]models.SecretCheckResult, 0, len(secrets)),
		AppRegistrations: make([]models.AppRegCheckResult, 0, len(appRegs)),
	}
	for _, c := range certs {
		payload.Certificates = append(payload.Certificates, *c)
	}
	for _, s := range secrets {
		payload.Secrets = append(payload.Secrets, *s)
	}
	for _, a := range appRegs {
		payload.AppRegistrations = append(payload.AppRegistrations, *a)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if g.retryPolicy.maxAttempts == 0 {
		g.retryPolicy = defaultRetryPolicy
	}

	newRequest := func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, g.WebhookUrl, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		switch g.AuthType {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+g.AuthToken)
		case "basic":
			req.SetBasicAuth(g.AuthUsername, g.AuthPassword)
		}
		return req, nil
	}

	return postWithRetry(g.getClient(), newRequest, "generic webhook", g.retryPolicy)
}

func (g *GenericWebhookNotifier) IsReady() bool {
	return g.WebhookUrl != ""
}

func (g *GenericWebhookNotifier) getClient() *http.Client {
	if g.httpClient == nil {
		g.httpClient = &http.Client{Timeout: webhookClientTimeout}
	}
	return g.httpClient
}
