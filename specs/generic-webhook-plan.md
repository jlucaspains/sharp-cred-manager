# Generic Webhook Notifier Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `generic` webhook notifier type that POSTs structured JSON (with all raw check result fields) to any HTTP endpoint with optional Bearer or Basic auth.

**Architecture:** Rename `WebHookNotifier` → `TeamsSlackNotifier` to free the "generic" name, introduce a `RawNotifier` interface for notifiers that need raw model data, refactor `CheckCredJob.execute()` to collect raw results first and branch on interface, then implement `GenericWebhookNotifier` as a new struct.

**Tech Stack:** Go standard library (`net/http`, `encoding/json`), `testify/assert`, `httptest` for tests.

---

## File Map

| File | Action |
|---|---|
| `internal/jobs/webHookNotifier.go` | Rename → `teamsSlackNotifier.go`; rename `WebHookNotifier` → `TeamsSlackNotifier`, `WebHookNotificationCard` → `TeamsSlackNotificationCard` |
| `internal/jobs/webHookNotifier_test.go` | Rename → `teamsSlackNotifier_test.go`; update all type references |
| `internal/jobs/notificationTemplates.go` | No type changes needed; `Teams`/`Slack` constants and `Notifiers` map stay as-is |
| `internal/jobs/CheckCredJob.go` | Add `RawNotifier` interface; add `checkCerts/checkSecrets/checkAppRegs`; add `filterCertResults/filterSecretResults/filterAppRegResults`; refactor `execute()` and `buildXGroup()` methods |
| `internal/jobs/CheckCredJob_test.go` | Add `mockRawNotifier` and `TestCheckCredJob_UsesRawNotifier` |
| `internal/jobs/genericWebhookNotifier.go` | New file — `GenericWebhookNotifier` and `GenericWebhookPayload` |
| `internal/jobs/genericWebhookNotifier_test.go` | New file — all notifier tests |
| `cmd/api/main.go` | Update `getJobNotifier()` to handle `WEBHOOK_TYPE=generic` |

---

## Task 1: Rename WebHookNotifier → TeamsSlackNotifier

**Files:**
- Rename: `internal/jobs/webHookNotifier.go` → `internal/jobs/teamsSlackNotifier.go`
- Rename: `internal/jobs/webHookNotifier_test.go` → `internal/jobs/teamsSlackNotifier_test.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Rename the source files**

```bash
git mv internal/jobs/webHookNotifier.go internal/jobs/teamsSlackNotifier.go
git mv internal/jobs/webHookNotifier_test.go internal/jobs/teamsSlackNotifier_test.go
```

- [ ] **Step 2: Update struct and type names in `teamsSlackNotifier.go`**

Replace every occurrence of `WebHookNotifier` with `TeamsSlackNotifier` and `WebHookNotificationCard` with `TeamsSlackNotificationCard`. The full file after edits:

```go
package jobs

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"
)

type TeamsSlackNotifier struct {
	NotifierType      NotifierType
	WebhookUrl        string
	NotificationTitle string
	NotificationBody  string
	NotificationUrl   string
	Mentions          []string
	parsedTemplate    *template.Template
	httpClient        *http.Client
}

type TeamsSlackNotificationCard struct {
	Title           string
	Description     string
	NotificationUrl string
	Groups          []CheckNotificationGroup
	Mentions        []string
}

func (m *TeamsSlackNotifier) Init(notifierType NotifierType, webhookUrl string, notificationTitle string, notificationBody string, notificationUrl string, messageMentions string) {
	if notificationTitle == "" {
		notificationTitle = "Sharp Cred Manager Summary"
	}

	if notificationBody == "" {
		notificationBody = fmt.Sprintf("The following credentials were checked on %s", time.Now().Format("01/02/2006"))
	}

	m.NotifierType = notifierType
	m.NotificationTitle = notificationTitle
	m.NotificationBody = notificationBody
	m.NotificationUrl = notificationUrl
	m.WebhookUrl = webhookUrl
	m.Mentions = parseMentions(messageMentions)
}

func parseMentions(mentions string) []string {
	if mentions == "" {
		return []string{}
	}

	return strings.Split(mentions, ",")
}

func (m *TeamsSlackNotifier) Notify(groups []CheckNotificationGroup) error {
	client := m.getClient()
	parsedTemplate := m.getTemplate()
	card := TeamsSlackNotificationCard{
		Title:           m.NotificationTitle,
		Description:     m.NotificationBody,
		NotificationUrl: m.NotificationUrl,
		Groups:          groups,
		Mentions:        m.Mentions,
	}

	var templateBody bytes.Buffer
	err := parsedTemplate.Execute(&templateBody, card)

	if err != nil {
		return err
	}

	stringBody := templateBody.String()
	fmt.Println(stringBody)

	response, err := client.Post(m.WebhookUrl, "application/json", bytes.NewReader(templateBody.Bytes()))

	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 202 {
		body := new(bytes.Buffer)
		body.ReadFrom(response.Body)
		if body.Len() > 0 {
			return fmt.Errorf("error sending notification to %s. Error %s: %s", m.NotifierType, response.Status, body.String())
		}
		return fmt.Errorf("error sending notification to %s. Error %s", m.NotifierType, response.Status)
	}

	return nil
}

func (m *TeamsSlackNotifier) getTemplate() *template.Template {
	if m.parsedTemplate == nil {
		m.parsedTemplate, _ = template.New("template").Funcs(template.FuncMap{
			"split": func(s, sep string) []string {
				return strings.Split(s, sep)
			},
			"replace": func(input, from, to string) string {
				return strings.ReplaceAll(input, from, to)
			},
		}).Parse(NotificationTemplates[m.NotifierType])
	}

	return m.parsedTemplate
}

func (m *TeamsSlackNotifier) getClient() *http.Client {
	if m.httpClient == nil {
		m.httpClient = &http.Client{}
	}

	return m.httpClient
}

func (m *TeamsSlackNotifier) IsReady() bool {
	return m.WebhookUrl != ""
}
```

- [ ] **Step 3: Update `teamsSlackNotifier_test.go` — replace all `WebHookNotifier` references**

Change every `WebHookNotifier` variable name and type reference to `TeamsSlackNotifier`. Example (apply to all test functions):

```go
// Before
WebHookNotifier := &WebHookNotifier{}
WebHookNotifier.Init(Teams, ts.URL, "title", "body", "url", "")
err := WebHookNotifier.Notify([]CheckNotificationGroup{})

// After
notifier := &TeamsSlackNotifier{}
notifier.Init(Teams, ts.URL, "title", "body", "url", "")
err := notifier.Notify([]CheckNotificationGroup{})
```

Apply consistently to all 7 test functions in the file.

- [ ] **Step 4: Update `main.go` — replace `&jobs.WebHookNotifier{}` with `&jobs.TeamsSlackNotifier{}`**

In `getJobNotifier()`:
```go
// Before
result := &jobs.WebHookNotifier{}
result.Init(jobs.Notifiers[webhookType], WebhookUrl, messageTitle, messageBody, messageUrl, messageMentions)
return result

// After (keep the rest of the function the same for now)
result := &jobs.TeamsSlackNotifier{}
result.Init(jobs.Notifiers[webhookType], WebhookUrl, messageTitle, messageBody, messageUrl, messageMentions)
return result
```

- [ ] **Step 5: Run all tests and verify they pass**

```bash
go test -cover ./...
```

Expected: all tests pass, no compile errors.

- [ ] **Step 6: Commit**

```bash
git add internal/jobs/teamsSlackNotifier.go internal/jobs/teamsSlackNotifier_test.go cmd/api/main.go
git commit -m "Rename WebHookNotifier to TeamsSlackNotifier"
```

---

## Task 2: Write failing test for RawNotifier dispatch

**Files:**
- Modify: `internal/jobs/CheckCredJob_test.go`

- [ ] **Step 1: Add `mockRawNotifier` and the failing test to `CheckCredJob_test.go`**

Add after the existing `mockNotifier` struct (around line 25):

```go
type mockRawNotifier struct {
	notifyRawCalled bool
	certResults     []*models.CertCheckResult
	secretResults   []*models.SecretCheckResult
	appRegResults   []*models.AppRegCheckResult
}

func (m *mockRawNotifier) Notify(_ []CheckNotificationGroup) error {
	return nil
}

func (m *mockRawNotifier) IsReady() bool {
	return true
}

func (m *mockRawNotifier) NotifyRaw(certs []*models.CertCheckResult, secrets []*models.SecretCheckResult, appRegs []*models.AppRegCheckResult) error {
	m.notifyRawCalled = true
	m.certResults = certs
	m.secretResults = secrets
	m.appRegResults = appRegs
	return nil
}
```

Add at the end of the file:

```go
func TestCheckCredJob_UsesRawNotifier(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockRawNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: notifier,
	})
	job.notifier = notifier
	job.RunNow()

	assert.True(t, notifier.notifyRawCalled, "NotifyRaw should be called when notifier implements RawNotifier")
}
```

- [ ] **Step 2: Run the new test and verify it fails**

```bash
go test -v ./internal/jobs/ -run TestCheckCredJob_UsesRawNotifier
```

Expected: test compiles (`mockRawNotifier` satisfies `Notifier` via `Notify` + `IsReady`) but FAILS at runtime — `notifyRawCalled` is false because `execute()` does not yet do a `RawNotifier` type assertion.

- [ ] **Step 3: Commit the failing test**

```bash
git add internal/jobs/CheckCredJob_test.go
git commit -m "Add failing test for RawNotifier dispatch in CheckCredJob"
```

---

## Task 3: Refactor CheckCredJob to support RawNotifier

**Files:**
- Modify: `internal/jobs/CheckCredJob.go`

- [ ] **Step 1: Add `RawNotifier` interface after the `Notifier` interface (around line 17)**

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

- [ ] **Step 2: Add `checkCerts`, `checkSecrets`, `checkAppRegs` methods**

Replace the bodies of `buildCertGroup`, `buildSecretGroup`, `buildAppRegGroup` so they no longer call services directly. Add these new methods above the existing `buildCertGroup`:

```go
func (c *CheckCredJob) checkCerts() []*models.CertCheckResult {
	results := []*models.CertCheckResult{}
	for _, item := range c.certList {
		checkStatus, err := services.CheckCertStatus(item, c.certWarningDays)
		if err != nil {
			log.Printf("Error checking cert status: %s", err)
			continue
		}
		log.Printf("Cert status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}

func (c *CheckCredJob) checkSecrets() []*models.SecretCheckResult {
	results := []*models.SecretCheckResult{}
	for _, item := range c.secretList {
		checkStatus, err := services.CheckSecretStatus(item, c.secretWarningDays)
		if err != nil {
			log.Printf("Error checking secret status: %s", err)
			continue
		}
		log.Printf("Secret status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}

func (c *CheckCredJob) checkAppRegs() []*models.AppRegCheckResult {
	results := []*models.AppRegCheckResult{}
	for _, item := range c.appRegList {
		checkStatus, err := services.CheckAppRegStatus(item, c.appRegWarningDays)
		if err != nil {
			log.Printf("Error checking app registration status: %s", err)
			continue
		}
		log.Printf("App registration status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}
```

- [ ] **Step 3: Add filter methods for the raw path**

Add below the `checkAppRegs` method:

```go
func (c *CheckCredJob) filterCertResults(results []*models.CertCheckResult) []*models.CertCheckResult {
	filtered := []*models.CertCheckResult{}
	for _, r := range results {
		if c.shouldNotify(CheckNotification{IsValid: r.IsValid, ExpirationWarning: r.ExpirationWarning}) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (c *CheckCredJob) filterSecretResults(results []*models.SecretCheckResult) []*models.SecretCheckResult {
	filtered := []*models.SecretCheckResult{}
	for _, r := range results {
		if c.shouldNotify(CheckNotification{IsValid: r.IsValid, ExpirationWarning: r.ExpirationWarning}) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (c *CheckCredJob) filterAppRegResults(results []*models.AppRegCheckResult) []*models.AppRegCheckResult {
	filtered := []*models.AppRegCheckResult{}
	for _, r := range results {
		filteredCreds := []models.AppRegCredentialResult{}
		for _, cred := range r.Credentials {
			if c.shouldNotify(CheckNotification{IsValid: cred.IsValid, ExpirationWarning: cred.ExpirationWarning}) {
				filteredCreds = append(filteredCreds, cred)
			}
		}
		if len(filteredCreds) > 0 {
			filtered = append(filtered, &models.AppRegCheckResult{
				Name:              r.Name,
				AppName:           r.AppName,
				AppId:             r.AppId,
				AppObjectId:       r.AppObjectId,
				IsValid:           r.IsValid,
				ExpirationWarning: r.ExpirationWarning,
				Credentials:       filteredCreds,
			})
		}
	}
	return filtered
}
```

- [ ] **Step 4: Refactor `execute()` to collect raw results first, then branch**

Replace the existing `execute()` method:

```go
func (c *CheckCredJob) execute() {
	certResults := c.checkCerts()
	secretResults := c.checkSecrets()
	appRegResults := c.checkAppRegs()

	if raw, ok := c.notifier.(RawNotifier); ok {
		if err := raw.NotifyRaw(
			c.filterCertResults(certResults),
			c.filterSecretResults(secretResults),
			c.filterAppRegResults(appRegResults),
		); err != nil {
			log.Printf("Error sending notification: %s", err)
		}
		return
	}

	groups := []CheckNotificationGroup{}
	if certGroup := c.buildCertGroup(certResults); len(certGroup.Items) > 0 {
		groups = append(groups, certGroup)
	}
	if secretGroup := c.buildSecretGroup(secretResults); len(secretGroup.Items) > 0 {
		groups = append(groups, secretGroup)
	}
	if appRegGroup := c.buildAppRegGroup(appRegResults); len(appRegGroup.Items) > 0 {
		groups = append(groups, appRegGroup)
	}

	if err := c.notifier.Notify(groups); err != nil {
		log.Printf("Error sending notification: %s", err)
	}
}
```

- [ ] **Step 5: Refactor `buildCertGroup`, `buildSecretGroup`, `buildAppRegGroup` to accept pre-fetched results**

Replace all three methods:

```go
func (c *CheckCredJob) buildCertGroup(results []*models.CertCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "Certificates", ShowSource: true}
	for _, checkStatus := range results {
		n := c.getCertNotification(checkStatus)
		if c.shouldNotify(n) {
			group.Items = append(group.Items, n)
		}
	}
	return group
}

func (c *CheckCredJob) buildSecretGroup(results []*models.SecretCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "Secrets", ShowSource: true}
	for _, checkStatus := range results {
		n := c.getSecretNotification(checkStatus)
		if c.shouldNotify(n) {
			group.Items = append(group.Items, n)
		}
	}
	return group
}

func (c *CheckCredJob) buildAppRegGroup(results []*models.AppRegCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "App Registrations", ShowSource: true}
	for _, checkStatus := range results {
		for _, cred := range checkStatus.Credentials {
			n := c.getCredentialNotification(checkStatus.AppName, cred)
			if c.shouldNotify(n) {
				group.Items = append(group.Items, n)
			}
		}
	}
	return group
}
```

- [ ] **Step 6: Run all tests and verify they pass**

```bash
go test -cover ./...
```

Expected: all tests pass including the previously failing `TestCheckCredJob_UsesRawNotifier`.

- [ ] **Step 7: Commit**

```bash
git add internal/jobs/CheckCredJob.go internal/jobs/CheckCredJob_test.go
git commit -m "Refactor CheckCredJob to support RawNotifier interface"
```

---

## Task 4: Write failing tests for GenericWebhookNotifier

**Files:**
- Create: `internal/jobs/genericWebhookNotifier_test.go`

- [ ] **Step 1: Create the test file**

```go
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
```

- [ ] **Step 2: Run the tests and verify they all fail to compile**

```bash
go test -v ./internal/jobs/ -run TestGenericWebhookNotifier
```

Expected: compile error — `GenericWebhookNotifier` and `GenericWebhookPayload` undefined.

---

## Task 5: Implement GenericWebhookNotifier

**Files:**
- Create: `internal/jobs/genericWebhookNotifier.go`

- [ ] **Step 1: Create the implementation file**

```go
package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
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
}

type GenericWebhookPayload struct {
	Certificates     []models.CertCheckResult    `json:"certificates"`
	Secrets          []models.SecretCheckResult   `json:"secrets"`
	AppRegistrations []models.AppRegCheckResult   `json:"appRegistrations"`
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

	req, err := http.NewRequest(http.MethodPost, g.WebhookUrl, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	switch g.AuthType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+g.AuthToken)
	case "basic":
		req.SetBasicAuth(g.AuthUsername, g.AuthPassword)
	}

	resp, err := g.getClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		if buf.Len() > 0 {
			return fmt.Errorf("error sending notification to generic webhook. Error %s: %s", resp.Status, buf.String())
		}
		return fmt.Errorf("error sending notification to generic webhook. Error %s", resp.Status)
	}

	return nil
}

func (g *GenericWebhookNotifier) IsReady() bool {
	return g.WebhookUrl != ""
}

func (g *GenericWebhookNotifier) getClient() *http.Client {
	if g.httpClient == nil {
		g.httpClient = &http.Client{}
	}
	return g.httpClient
}
```

- [ ] **Step 2: Run the notifier tests and verify they pass**

```bash
go test -v ./internal/jobs/ -run TestGenericWebhookNotifier
```

Expected: all 7 tests PASS.

- [ ] **Step 3: Run all tests to check for regressions**

```bash
go test -cover ./...
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/jobs/genericWebhookNotifier.go internal/jobs/genericWebhookNotifier_test.go
git commit -m "Add GenericWebhookNotifier with bearer and basic auth support"
```

---

## Task 6: Wire up GenericWebhookNotifier in main.go

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update `getJobNotifier()` to handle `WEBHOOK_TYPE=generic`**

Replace the existing `getJobNotifier()` function:

```go
func getJobNotifier() jobs.Notifier {
	webhookType, _ := os.LookupEnv("WEBHOOK_TYPE")
	webhookUrl, _ := os.LookupEnv("WEBHOOK_URL")

	if webhookType == "generic" {
		authType, _ := os.LookupEnv("WEBHOOK_AUTH_TYPE")
		authToken, _ := os.LookupEnv("WEBHOOK_AUTH_TOKEN")
		authUsername, _ := os.LookupEnv("WEBHOOK_AUTH_USERNAME")
		authPassword, _ := os.LookupEnv("WEBHOOK_AUTH_PASSWORD")
		result := &jobs.GenericWebhookNotifier{}
		result.Init(webhookUrl, authType, authToken, authUsername, authPassword)
		return result
	}

	messageUrl, _ := os.LookupEnv("MESSAGE_URL")
	messageTitle, _ := os.LookupEnv("MESSAGE_TITLE")
	messageBody, _ := os.LookupEnv("MESSAGE_BODY")
	messageMentions, _ := os.LookupEnv("MESSAGE_MENTIONS")

	result := &jobs.TeamsSlackNotifier{}
	result.Init(jobs.Notifiers[webhookType], webhookUrl, messageTitle, messageBody, messageUrl, messageMentions)
	return result
}
```

- [ ] **Step 2: Run all tests**

```bash
go test -cover ./...
```

Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "Wire up GenericWebhookNotifier for WEBHOOK_TYPE=generic"
```
