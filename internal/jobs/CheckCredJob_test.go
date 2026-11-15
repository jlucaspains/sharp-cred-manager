package jobs

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
	"github.com/stretchr/testify/assert"
)

type notifyCall struct {
	groups []CheckNotificationGroup
}

type mockNotifier struct {
	executed bool
	calls    []notifyCall
	failOn   map[int]error
}

func (m *mockNotifier) Notify(groups []CheckNotificationGroup) error {
	m.executed = true
	m.calls = append(m.calls, notifyCall{groups: groups})
	return m.failOn[len(m.calls)-1]
}

func (m *mockNotifier) IsReady() bool {
	return true
}

type rawNotifyCall struct {
	certs   []*models.CertCheckResult
	secrets []*models.SecretCheckResult
	appRegs []*models.AppRegCheckResult
}

type mockRawNotifier struct {
	notifyRawCalled bool
	certResults     []*models.CertCheckResult
	secretResults   []*models.SecretCheckResult
	appRegResults   []*models.AppRegCheckResult
	calls           []rawNotifyCall
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
	m.calls = append(m.calls, rawNotifyCall{certs: certs, secrets: secrets, appRegs: appRegs})
	return nil
}

var certList = []models.CheckCertItem{
	{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
}

var secretList = []models.CheckSecretItem{
	{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	},
}

func setupMockSecret(enabled bool, expiresAt time.Time) {
	contentType := "text/plain"
	services.SetMockSecretResult(&azsecrets.SecretProperties{
		Attributes: &azsecrets.SecretAttributes{
			Enabled: &enabled,
			Expires: &expiresAt,
		},
		ContentType: &contentType,
	})
}

var appRegList = []models.CheckAppRegItem{
	{
		Name:        "app-id",
		AppId:       "app-id",
		AppObjectId: "object-id-1",
		AppName:     "TestApp",
	},
}

func setupMockAppReg(valid bool, warning bool) {
	cred := models.AppRegCredentialResult{
		KeyId:             "key-1",
		DisplayName:       "CI Key",
		CredentialType:    models.AppRegCredentialSecret,
		IsValid:           valid,
		ExpirationWarning: warning,
		ValidationIssues:  []string{},
	}
	if !valid {
		cred.ValidationIssues = append(cred.ValidationIssues, "Credential is expired")
	}

	services.SetMockAppRegResult(&models.AppRegCheckResult{
		Name:              "app-id",
		AppName:           "TestApp",
		AppId:             "app-id",
		IsValid:           valid,
		ExpirationWarning: warning,
		Credentials:       []models.AppRegCredentialResult{cred},
	})
}

func TestJobInit(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   1,
		SecretWarningDays: 1,
		AppRegWarningDays: 1,
		CertList:          certList,
		SecretList:        secretList,
		Notifier:          &mockNotifier{},
	})

	assert.Nil(t, err)
	assert.Equal(t, "* * * * *", job.cron)
	assert.Equal(t, "https://blog.lpains.net", job.certList[0].Url)
	assert.Equal(t, "blog.lpains.net", job.certList[0].Name)
	assert.Equal(t, models.CertCheckURL, job.certList[0].Type)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", job.secretList[0].Name)

	job.ticker.Stop()
}

func TestJobInitDefaultWarningDays(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: &mockNotifier{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 30, job.certWarningDays)
	assert.Equal(t, 30, job.secretWarningDays)
	assert.Equal(t, 30, job.appRegWarningDays)

	job.ticker.Stop()
}

func TestJobInitDistinctWarningDays(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   20,
		SecretWarningDays: 45,
		AppRegWarningDays: 60,
		CertList:          certList,
		Notifier:          &mockNotifier{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 20, job.certWarningDays)
	assert.Equal(t, 45, job.secretWarningDays)
	assert.Equal(t, 60, job.appRegWarningDays)

	job.ticker.Stop()
}

func TestJobInitBadCron(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule: "* * * *",
		CertList: certList,
		Notifier: &mockNotifier{},
	})

	assert.Equal(t, "a valid cron schedule is required", err.Error())
}

func TestJobInitBadNotifier(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: nil,
	})

	assert.Equal(t, "a valid notifier is required", err.Error())
}

func TestJobStartStop(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: &mockNotifier{},
	})
	assert.Nil(t, err)
	job.Start()
	assert.True(t, job.running)
	job.Stop()
	assert.False(t, job.running)
}

func TestTryExecuteNotDue(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule: "0 0 1 1 1",
		CertList: certList,
		Notifier: &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.False(t, notifier.executed)
}

func TestTryExecuteDue(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestExecuteNow(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule: "* * * * *",
		CertList: certList,
		Notifier: &mockNotifier{},
	})
	job.notifier = notifier
	job.RunNow()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithCertWarning(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule:        "* * * * *",
		CertWarningDays: 10000,
		CertList:        certList,
		Notifier:        &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithSecret(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	setupMockSecret(true, expiresAt)
	defer services.SetMockSecretResult(nil)

	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule:   "* * * * *",
		SecretList: secretList,
		Notifier:   &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithSecretWarning(t *testing.T) {
	expiresAt := time.Now().UTC().Add(15 * 24 * time.Hour)
	setupMockSecret(true, expiresAt)
	defer services.SetMockSecretResult(nil)

	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 10000,
		SecretList:        secretList,
		Notifier:          &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithAppReg(t *testing.T) {
	setupMockAppReg(true, false)
	defer services.SetMockAppRegResult(nil)

	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule:   "* * * * *",
		AppRegList: appRegList,
		Notifier:   &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithAppRegWarning(t *testing.T) {
	setupMockAppReg(true, true)
	defer services.SetMockAppRegResult(nil)

	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 10000,
		AppRegList:        appRegList,
		Notifier:          &mockNotifier{},
	})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestGetCertNotificationValidWithWarning(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		CertList:          certList,
		Notifier:          &mockNotifier{},
	})

	expirationDate := time.Now().AddDate(0, 0, 15)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		DisplayName:       "test.example.com",
		IsValid:           true,
		ExpirationWarning: true,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{},
	}

	result := job.getCertNotification(cert)

	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Equal(t, "test.example.com", result.Name)
	assert.True(t, len(result.Messages) > 0, "Messages should contain expiration date")
	assert.True(t, strings.Contains(result.Messages[0], "Certificate expires in"), "Should contain expiration message")
	assert.True(t, strings.Contains(result.Messages[0], "days"), "Should contain 'days' in message")
}

func TestGetCertNotificationValidWithoutWarning(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		CertList:          certList,
		Notifier:          &mockNotifier{},
	})

	expirationDate := time.Now().AddDate(0, 1, 0)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		DisplayName:       "test.example.com",
		IsValid:           true,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{},
	}

	result := job.getCertNotification(cert)

	assert.True(t, result.IsValid)
	assert.False(t, result.ExpirationWarning)
	assert.Equal(t, "test.example.com", result.Name)
	assert.True(t, len(result.Messages) > 0, "Messages should contain expiration date even without warning")
	assert.True(t, strings.Contains(result.Messages[0], "Certificate expires in"), "Should contain expiration message")
}

func TestGetCertNotificationValidWithValidationIssues(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		CertList:          certList,
		Notifier:          &mockNotifier{},
	})

	expirationDate := time.Now().AddDate(0, 1, 0)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           true,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{"Issue 1", "Issue 2"},
	}

	result := job.getCertNotification(cert)

	assert.True(t, result.IsValid)
	assert.Equal(t, 3, len(result.Messages), "Should have 2 validation issues + 1 expiration message")
	assert.Equal(t, "Issue 1", result.Messages[0])
	assert.Equal(t, "Issue 2", result.Messages[1])
	assert.True(t, strings.Contains(result.Messages[2], "Certificate expires in"), "Third message should be expiration")
}

func TestGetCertNotificationInvalid(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		CertList:          certList,
		Notifier:          &mockNotifier{},
	})

	expirationDate := time.Now().AddDate(0, 0, -5)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           false,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{"Certificate expired"},
	}

	result := job.getCertNotification(cert)

	assert.False(t, result.IsValid)
	assert.Equal(t, 1, len(result.Messages), "Should only have validation issue, no expiration message for invalid certs")
	assert.Equal(t, "Certificate expired", result.Messages[0])
	assert.False(t, strings.Contains(strings.Join(result.Messages, " "), "Certificate expires in"))
}

func TestGetSecretNotificationValidWithWarning(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		SecretList:        secretList,
		Notifier:          &mockNotifier{},
	})

	expiresAt := time.Now().AddDate(0, 0, 15)
	secret := &models.SecretCheckResult{
		Name:              "testfake.vault.azure.net/my-secret",
		DisplayName:       "my-secret",
		Source:            "testfake.vault.azure.net/",
		IsValid:           true,
		ExpirationWarning: true,
		ExpiresOn:         &expiresAt,
		ValidationIssues:  []string{},
	}

	result := job.getSecretNotification(secret)

	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Equal(t, "my-secret", result.Name)
	assert.Equal(t, "testfake", result.Source)
	assert.True(t, len(result.Messages) > 0)
	assert.True(t, strings.Contains(result.Messages[0], "Secret expires in"))
	assert.True(t, strings.Contains(result.Messages[0], "days"))

	job.ticker.Stop()
}

func TestGetSecretNotificationValidNoExpiry(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		SecretList:        secretList,
		Notifier:          &mockNotifier{},
	})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
		DisplayName:      "my-secret",
		Source:           "testfake.vault.azure.net/",
		IsValid:          true,
		ExpiresOn:        nil,
		ValidationIssues: []string{},
	}

	result := job.getSecretNotification(secret)

	assert.True(t, result.IsValid)
	assert.Empty(t, result.Messages)

	job.ticker.Stop()
}

func TestGetSecretNotificationInvalid(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:          "* * * * *",
		CertWarningDays:   30,
		SecretWarningDays: 30,
		AppRegWarningDays: 30,
		SecretList:        secretList,
		Notifier:          &mockNotifier{},
	})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
		DisplayName:      "my-secret",
		Source:           "testfake.vault.azure.net/",
		IsValid:          false,
		ExpiresOn:        nil,
		ValidationIssues: []string{"Secret is disabled"},
	}

	result := job.getSecretNotification(secret)

	assert.False(t, result.IsValid)
	assert.Equal(t, 1, len(result.Messages))
	assert.Equal(t, "Secret is disabled", result.Messages[0])
	assert.False(t, strings.Contains(strings.Join(result.Messages, " "), "Secret expires in"))

	job.ticker.Stop()
}

func TestGetCredentialNotificationValidSecret(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:  "* * * * *",
		AppRegList: appRegList,
		Notifier:  &mockNotifier{},
	})

	end := time.Now().AddDate(0, 0, 90)
	cred := models.AppRegCredentialResult{
		DisplayName: "CI Key", CredentialType: models.AppRegCredentialSecret,
		IsValid: true, HasExpiration: true, ValidityInDays: 90, EndDateTime: &end,
	}

	result := job.getCredentialNotification("TestApp", cred)

	assert.True(t, result.IsValid)
	assert.Equal(t, "TestApp", result.Source)
	assert.Equal(t, "🔑 CI Key", result.Name)
	assert.Empty(t, result.Messages)
}

func TestGetCredentialNotificationExpired(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:  "* * * * *",
		AppRegList: appRegList,
		Notifier:  &mockNotifier{},
	})

	cred := models.AppRegCredentialResult{
		DisplayName: "Old Key", CredentialType: models.AppRegCredentialSecret,
		IsValid: false, ValidationIssues: []string{"Credential is expired"},
	}

	result := job.getCredentialNotification("TestApp", cred)

	assert.False(t, result.IsValid)
	assert.Equal(t, "TestApp", result.Source)
	assert.Equal(t, "🔑 Old Key", result.Name)
	assert.Equal(t, 1, len(result.Messages))
	assert.Equal(t, "Credential is expired", result.Messages[0])
}

func TestGetCredentialNotificationWarningCertificate(t *testing.T) {
	job := &CheckCredJob{}
	job.Init(CheckCredJobConfig{
		Schedule:  "* * * * *",
		AppRegList: appRegList,
		Notifier:  &mockNotifier{},
	})

	cred := models.AppRegCredentialResult{
		DisplayName: "Auth Cert", CredentialType: models.AppRegCredentialCertificate,
		IsValid: true, ExpirationWarning: true, ValidityInDays: 10,
	}

	result := job.getCredentialNotification("TestApp", cred)

	assert.True(t, result.IsValid)
	assert.Equal(t, "TestApp", result.Source)
	assert.Equal(t, "📜 Auth Cert", result.Name)
	assert.Equal(t, 1, len(result.Messages))
	assert.Equal(t, "expires in 10 days", result.Messages[0])
}

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

func TestFilterCertResults_ExcludesInfo(t *testing.T) {
	job := &CheckCredJob{certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	results := []*models.CertCheckResult{
		{IsValid: true, ExpirationWarning: false},  // should be excluded at Warning level
		{IsValid: true, ExpirationWarning: true},   // should be included
		{IsValid: false, ExpirationWarning: false},  // should be included
	}
	filtered := job.filterCertResults(results)
	assert.Equal(t, 2, len(filtered))
	assert.True(t, filtered[0].ExpirationWarning)
	assert.False(t, filtered[1].IsValid)
}

func TestFilterSecretResults_ExcludesInfo(t *testing.T) {
	job := &CheckCredJob{certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	results := []*models.SecretCheckResult{
		{IsValid: true, ExpirationWarning: false},
		{IsValid: true, ExpirationWarning: true},
	}
	filtered := job.filterSecretResults(results)
	assert.Equal(t, 1, len(filtered))
	assert.True(t, filtered[0].ExpirationWarning)
}

func TestFilterAppRegResults_FiltersCredentials(t *testing.T) {
	job := &CheckCredJob{certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	results := []*models.AppRegCheckResult{
		{
			AppName: "App1",
			IsValid: true,
			ExpirationWarning: true,
			Credentials: []models.AppRegCredentialResult{
				{IsValid: true, ExpirationWarning: true},   // included
				{IsValid: true, ExpirationWarning: false},  // excluded at Warning level
			},
		},
		{
			AppName: "App2",
			IsValid: true,
			ExpirationWarning: false,
			Credentials: []models.AppRegCredentialResult{
				{IsValid: true, ExpirationWarning: false},  // excluded — whole app reg should be dropped
			},
		},
	}
	filtered := job.filterAppRegResults(results)
	assert.Equal(t, 1, len(filtered), "App2 should be dropped since no credentials pass")
	assert.Equal(t, "App1", filtered[0].AppName)
	assert.Equal(t, 1, len(filtered[0].Credentials), "Only the warning credential should remain")
	assert.True(t, filtered[0].ExpirationWarning, "Recomputed ExpirationWarning should be true")
	assert.True(t, filtered[0].IsValid, "Recomputed IsValid should be true (remaining cred is valid)")
}

func TestParseLevel(t *testing.T) {
	cases := map[string]Level{"Info": Info, "info": Info, "WARNING": Warning, " error ": Error}
	for value, expected := range cases {
		level, ok := parseLevel(value)
		assert.True(t, ok, value)
		assert.Equal(t, expected, level, value)
	}

	_, ok := parseLevel("Loud")
	assert.False(t, ok)
}

func TestResolveGlobalLevel(t *testing.T) {
	assert.Equal(t, Warning, resolveGlobalLevel(""))
	assert.Equal(t, Info, resolveGlobalLevel("info"))
	assert.Equal(t, Warning, resolveGlobalLevel("Errors"))
}

func TestResolveLevel(t *testing.T) {
	assert.Equal(t, Info, resolveLevel("cert", "", Info))
	assert.Equal(t, Error, resolveLevel("cert", "Error", Info))
	assert.Equal(t, Error, resolveLevel("cert", "error", Info))
	assert.Equal(t, Info, resolveLevel("cert", "Loud", Info))
	assert.Equal(t, Warning, resolveLevel("cert", "   ", Warning))
}

func TestJobInitLevels(t *testing.T) {
	job := &CheckCredJob{}
	err := job.Init(CheckCredJobConfig{
		Schedule:  "* * * * *",
		Level:     "Error",
		CertLevel: "Info",
		Notifier:  &mockNotifier{},
	})
	defer job.Stop()
	assert.Nil(t, err)
	assert.Equal(t, Info, job.certLevel)
	assert.Equal(t, Error, job.secretLevel)
	assert.Equal(t, Error, job.appRegLevel)
}

func TestJobInitDefaultLevels(t *testing.T) {
	job := &CheckCredJob{}
	err := job.Init(CheckCredJobConfig{Schedule: "* * * * *", Notifier: &mockNotifier{}})
	defer job.Stop()
	assert.Nil(t, err)
	assert.Equal(t, Warning, job.certLevel)
	assert.Equal(t, Warning, job.secretLevel)
	assert.Equal(t, Warning, job.appRegLevel)
	assert.Equal(t, Combined, job.mode)
}

func TestJobInitMode(t *testing.T) {
	for _, value := range []string{"split", "Split", "SPLIT", " split "} {
		job := &CheckCredJob{}
		job.Init(CheckCredJobConfig{Schedule: "* * * * *", Mode: value, Notifier: &mockNotifier{}})
		job.Stop()
		assert.Equal(t, Split, job.mode, value)
	}

	for _, value := range []string{"", "combined", "bogus"} {
		job := &CheckCredJob{}
		job.Init(CheckCredJobConfig{Schedule: "* * * * *", Mode: value, Notifier: &mockNotifier{}})
		job.Stop()
		assert.Equal(t, Combined, job.mode, value)
	}
}

func futureDate(days int) time.Time {
	return time.Now().Add(time.Duration(days*24) * time.Hour)
}

// sampleResults returns one valid (non-warning) and one warning item per type.
func sampleResults() ([]*models.CertCheckResult, []*models.SecretCheckResult, []*models.AppRegCheckResult) {
	expires := futureDate(10)
	certs := []*models.CertCheckResult{
		{DisplayName: "valid-cert", IsValid: true, CertEndDate: futureDate(200)},
		{DisplayName: "warn-cert", IsValid: true, ExpirationWarning: true, CertEndDate: futureDate(10)},
	}
	secrets := []*models.SecretCheckResult{
		{DisplayName: "valid-secret", IsValid: true},
		{DisplayName: "warn-secret", IsValid: true, ExpirationWarning: true, ExpiresOn: &expires},
	}
	appRegs := []*models.AppRegCheckResult{
		{
			AppName: "App1",
			Credentials: []models.AppRegCredentialResult{
				{DisplayName: "valid-cred", IsValid: true},
				{DisplayName: "warn-cred", IsValid: true, ExpirationWarning: true, ValidityInDays: 10},
			},
		},
	}
	return certs, secrets, appRegs
}

func TestNotifyGroupsCombinedPerTypeLevels(t *testing.T) {
	notifier := &mockNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Combined, certLevel: Info, secretLevel: Error, appRegLevel: Warning}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyGroups(certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(notifier.calls))
	groups := notifier.calls[0].groups
	assert.Equal(t, 2, len(groups), "secrets have nothing at Error level")
	assert.Equal(t, "Certificates", groups[0].Label)
	assert.Equal(t, 2, len(groups[0].Items), "Info includes valid certs")
	assert.Equal(t, "App Registrations", groups[1].Label)
	assert.Equal(t, 1, len(groups[1].Items))
}

func TestNotifyGroupsCombinedSendsWhenEmpty(t *testing.T) {
	notifier := &mockNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Combined, certLevel: Error, secretLevel: Error, appRegLevel: Error}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyGroups(certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(notifier.calls))
	assert.Equal(t, 0, len(notifier.calls[0].groups))
}

func TestNotifyGroupsSplit(t *testing.T) {
	notifier := &mockNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyGroups(certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(notifier.calls))
	for i, label := range []string{"Certificates", "Secrets", "App Registrations"} {
		assert.Equal(t, 1, len(notifier.calls[i].groups))
		assert.Equal(t, label, notifier.calls[i].groups[0].Label)
	}
}

func TestNotifyGroupsSplitSkipsEmptyTypes(t *testing.T) {
	notifier := &mockNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Warning, secretLevel: Error, appRegLevel: Warning}
	certs, secrets, _ := sampleResults()

	err := job.notifyGroups(certs, secrets, []*models.AppRegCheckResult{})

	assert.Nil(t, err)
	assert.Equal(t, 1, len(notifier.calls))
	assert.Equal(t, "Certificates", notifier.calls[0].groups[0].Label)
}

func TestNotifyGroupsSplitNothingToSend(t *testing.T) {
	notifier := &mockNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Error, secretLevel: Error, appRegLevel: Error}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyGroups(certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 0, len(notifier.calls))
}

func TestNotifyGroupsSplitContinuesOnFailure(t *testing.T) {
	notifier := &mockNotifier{failOn: map[int]error{0: errors.New("boom")}}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyGroups(certs, secrets, appRegs)

	assert.Equal(t, 3, len(notifier.calls), "remaining sends should still happen")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Certificates: boom")
}

func TestNotifyRawCombined(t *testing.T) {
	notifier := &mockRawNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Combined, certLevel: Info, secretLevel: Error, appRegLevel: Warning}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyRaw(notifier, certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(notifier.calls))
	assert.Equal(t, 2, len(notifier.calls[0].certs))
	assert.Equal(t, 0, len(notifier.calls[0].secrets))
	assert.Equal(t, 1, len(notifier.calls[0].appRegs))
}

func TestNotifyRawSplit(t *testing.T) {
	notifier := &mockRawNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Warning, secretLevel: Warning, appRegLevel: Warning}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyRaw(notifier, certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(notifier.calls))
	assert.Equal(t, 1, len(notifier.calls[0].certs))
	assert.Equal(t, 0, len(notifier.calls[0].secrets))
	assert.Equal(t, 0, len(notifier.calls[0].appRegs))
	assert.Equal(t, 0, len(notifier.calls[1].certs))
	assert.Equal(t, 1, len(notifier.calls[1].secrets))
	assert.Equal(t, 0, len(notifier.calls[1].appRegs))
	assert.Equal(t, 0, len(notifier.calls[2].certs))
	assert.Equal(t, 0, len(notifier.calls[2].secrets))
	assert.Equal(t, 1, len(notifier.calls[2].appRegs))
	assert.NotNil(t, notifier.calls[0].secrets, "empty lists should be non-nil so they marshal as []")
}

func TestNotifyRawSplitSkipsEmptyTypes(t *testing.T) {
	notifier := &mockRawNotifier{}
	job := &CheckCredJob{notifier: notifier, mode: Split, certLevel: Error, secretLevel: Warning, appRegLevel: Error}
	certs, secrets, appRegs := sampleResults()

	err := job.notifyRaw(notifier, certs, secrets, appRegs)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(notifier.calls))
	assert.Equal(t, 1, len(notifier.calls[0].secrets))
}
