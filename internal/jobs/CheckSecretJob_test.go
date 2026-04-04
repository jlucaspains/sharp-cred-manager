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

func TestSecretJobInit(t *testing.T) {
	job := &CheckSecretJob{}

	err := job.Init("* * * * *", "", 1, secretList, &mockNotifier{})

	assert.Nil(t, err)
	assert.Equal(t, "* * * * *", job.cron)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", job.secretList[0].Name)
	assert.Equal(t, "https://testfake.vault.azure.net/secrets/my-secret", job.secretList[0].Url)
	assert.Equal(t, models.SecretCheckAzure, job.secretList[0].Type)

	job.ticker.Stop()
}

func TestSecretJobInitDefaultWarningDays(t *testing.T) {
	job := &CheckSecretJob{}

	err := job.Init("* * * * *", "", 0, secretList, &mockNotifier{})

	assert.Nil(t, err)
	assert.Equal(t, 30, job.warningDays)

	job.ticker.Stop()
}

func TestSecretJobInitBadCron(t *testing.T) {
	job := &CheckSecretJob{}

	err := job.Init("* * * *", "", 0, secretList, &mockNotifier{})

	assert.Equal(t, "a valid cron schedule is required", err.Error())
}

func TestSecretJobInitBadNotifier(t *testing.T) {
	job := &CheckSecretJob{}

	err := job.Init("* * * * *", "", 0, secretList, nil)

	assert.Equal(t, "a valid notifier is required", err.Error())
}

func TestSecretJobStartStop(t *testing.T) {
	job := &CheckSecretJob{}

	err := job.Init("* * * * *", "", 0, secretList, &mockNotifier{})
	assert.Nil(t, err)
	job.Start()
	assert.True(t, job.running)
	job.Stop()
	assert.False(t, job.running)
}

func TestSecretJobTryExecuteNotDue(t *testing.T) {
	job := &CheckSecretJob{}
	notifier := &mockNotifier{}
	job.Init("0 0 1 1 1", "", 0, secretList, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.False(t, notifier.executed)
}

func TestSecretJobTryExecuteDue(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	setupMockSecret(true, expiresAt)
	defer services.SetMockSecretResult(nil)

	job := &CheckSecretJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 0, secretList, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestSecretJobRunNow(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	setupMockSecret(true, expiresAt)
	defer services.SetMockSecretResult(nil)

	job := &CheckSecretJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 0, secretList, &mockNotifier{})
	job.notifier = notifier
	job.RunNow()

	assert.True(t, notifier.executed)
}

func TestSecretJobTryExecuteDueWarning(t *testing.T) {
	expiresAt := time.Now().UTC().Add(15 * 24 * time.Hour)
	setupMockSecret(true, expiresAt)
	defer services.SetMockSecretResult(nil)

	job := &CheckSecretJob{}
	notifier := &mockNotifier{}
	job.Init("* * * * *", "", 10000, secretList, &mockNotifier{})
	job.notifier = notifier
	job.tryExecute()

	assert.True(t, notifier.executed)
}

func TestSecretGetNotificationModelValidWithWarning(t *testing.T) {
	job := &CheckSecretJob{}
	job.Init("* * * * *", "", 30, secretList, &mockNotifier{})

	expiresAt := time.Now().AddDate(0, 0, 15)
	secret := &models.SecretCheckResult{
		Name:              "testfake.vault.azure.net/my-secret",
		IsValid:           true,
		ExpirationWarning: true,
		ExpiresOn:         &expiresAt,
		ValidationIssues:  []string{},
	}

	result := job.getNotificationModel(secret)

	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", result.Name)
	assert.True(t, len(result.Messages) > 0)
	assert.True(t, strings.Contains(result.Messages[0], "Secret expires in"))
	assert.True(t, strings.Contains(result.Messages[0], "days"))

	job.ticker.Stop()
}

func TestSecretGetNotificationModelValidNoExpiry(t *testing.T) {
	job := &CheckSecretJob{}
	job.Init("* * * * *", "", 30, secretList, &mockNotifier{})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
		IsValid:          true,
		ExpiresOn:        nil,
		ValidationIssues: []string{},
	}

	result := job.getNotificationModel(secret)

	assert.True(t, result.IsValid)
	assert.Empty(t, result.Messages)

	job.ticker.Stop()
}

func TestSecretGetNotificationModelInvalid(t *testing.T) {
	job := &CheckSecretJob{}
	job.Init("* * * * *", "", 30, secretList, &mockNotifier{})

	secret := &models.SecretCheckResult{
		Name:             "testfake.vault.azure.net/my-secret",
		IsValid:          false,
		ExpiresOn:        nil,
		ValidationIssues: []string{"Secret is disabled"},
	}

	result := job.getNotificationModel(secret)

	assert.False(t, result.IsValid)
	assert.Equal(t, 1, len(result.Messages))
	assert.Equal(t, "Secret is disabled", result.Messages[0])
	assert.False(t, strings.Contains(strings.Join(result.Messages, " "), "Secret expires in"))

	job.ticker.Stop()
}

