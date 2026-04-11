package jobs

import (
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	executed bool
}

func (m *mockNotifier) Notify(groups []CheckNotificationGroup) error {
	m.executed = true
	return nil
}

func (m *mockNotifier) IsReady() bool {
	return true
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

func TestJobInit(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init("* * * * *", "", 1, 1, certList, secretList, &mockNotifier{})

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

	err := job.Init("* * * * *", "", 0, 0, certList, secretList, &mockNotifier{})

	assert.Nil(t, err)
	assert.Equal(t, 30, job.certWarningDays)
	assert.Equal(t, 30, job.secretWarningDays)

	job.ticker.Stop()
}

func TestJobInitDistinctWarningDays(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init("* * * * *", "", 20, 45, certList, secretList, &mockNotifier{})

	assert.Nil(t, err)
	assert.Equal(t, 20, job.certWarningDays)
	assert.Equal(t, 45, job.secretWarningDays)

	job.ticker.Stop()
}

func TestJobInitBadCron(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init("* * * *", "", 0, 0, certList, secretList, &mockNotifier{})

	assert.Equal(t, "a valid cron schedule is required", err.Error())
}

func TestJobInitBadNotifier(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init("* * * * *", "", 0, 0, certList, secretList, nil)

	assert.Equal(t, "a valid notifier is required", err.Error())
}

func TestJobStartStop(t *testing.T) {
	job := &CheckCredJob{}

	err := job.Init("* * * * *", "", 0, 0, certList, secretList, &mockNotifier{})
	assert.Nil(t, err)
	job.Start()
	assert.True(t, job.running)
	job.Stop()
	assert.False(t, job.running)
}

func TestTryExecuteNotDue(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init("0 0 1 1 1", "", 0, 0, certList, secretList, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.False(t, notifier.executed)
}

func TestTryExecuteDue(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 0, 0, certList, nil, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestExecuteNow(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 0, 0, certList, nil, &mockNotifier{})
	job.notifier = notifier
	job.RunNow()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWithCertWarning(t *testing.T) {
	job := &CheckCredJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 10000, 30, certList, nil, &mockNotifier{})
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
	job.Init("* * * * *", "", 0, 0, nil, secretList, &mockNotifier{})
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
	job.Init("* * * * *", "", 30, 10000, nil, secretList, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestGetCertNotificationValidWithWarning(t *testing.T) {
	job := &CheckCredJob{}
	job.Init("* * * * *", "", 30, 30, certList, nil, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 0, 15)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
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
	job.Init("* * * * *", "", 30, 30, certList, nil, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 1, 0)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
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
	job.Init("* * * * *", "", 30, 30, certList, nil, &mockNotifier{})

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
	job.Init("* * * * *", "", 30, 30, certList, nil, &mockNotifier{})

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
	job.Init("* * * * *", "", 30, 30, nil, secretList, &mockNotifier{})

	expiresAt := time.Now().AddDate(0, 0, 15)
	secret := &models.SecretCheckResult{
		Name:              "testfake.vault.azure.net/my-secret",
		IsValid:           true,
		ExpirationWarning: true,
		ExpiresOn:         &expiresAt,
		ValidationIssues:  []string{},
	}

	result := job.getSecretNotification(secret)

	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", result.Name)
	assert.True(t, len(result.Messages) > 0)
	assert.True(t, strings.Contains(result.Messages[0], "Secret expires in"))
	assert.True(t, strings.Contains(result.Messages[0], "days"))

	job.ticker.Stop()
}

func TestGetSecretNotificationValidNoExpiry(t *testing.T) {
	job := &CheckCredJob{}
	job.Init("* * * * *", "", 30, 30, nil, secretList, &mockNotifier{})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
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
	job.Init("* * * * *", "", 30, 30, nil, secretList, &mockNotifier{})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
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
