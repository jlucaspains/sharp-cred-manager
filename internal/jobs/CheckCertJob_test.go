package jobs

import (
	"strings"
	"testing"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	executed bool
}

func (m *mockNotifier) Notify(result []CertCheckNotification) error {
	m.executed = true
	return nil
}

func (m *mockNotifier) IsReady() bool {
	return true
}

var certList = []models.CheckCertItem{
	{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
}

func TestJobInit(t *testing.T) {
	checkCertJob := &CheckCertJob{}

	checkCertJob.Init("* * * * *", "", 1, certList, &mockNotifier{})

	assert.Equal(t, "* * * * *", checkCertJob.cron)
	assert.Equal(t, "https://blog.lpains.net", checkCertJob.certList[0].Url)
	assert.Equal(t, "blog.lpains.net", checkCertJob.certList[0].Name)
	assert.Equal(t, models.CertCheckURL, checkCertJob.certList[0].Type)

	checkCertJob.ticker.Stop()
}

func TestJobInitDefaultWarningDays(t *testing.T) {
	checkCertJob := &CheckCertJob{}

	checkCertJob.Init("* * * * *", "", 0, certList, &mockNotifier{})

	assert.Equal(t, "* * * * *", checkCertJob.cron)
	assert.Equal(t, "blog.lpains.net", checkCertJob.certList[0].Name)
	assert.Equal(t, "https://blog.lpains.net", checkCertJob.certList[0].Url)
	assert.Equal(t, models.CertCheckURL, checkCertJob.certList[0].Type)
	assert.Equal(t, 30, checkCertJob.warningDays)

	checkCertJob.ticker.Stop()
}

func TestJobInitBadCron(t *testing.T) {
	checkCertJob := &CheckCertJob{}

	err := checkCertJob.Init("* * * *", "", 0, certList, &mockNotifier{})

	assert.Equal(t, "a valid cron schedule is required", err.Error())
}

func TestJobInitBadNotifier(t *testing.T) {
	checkCertJob := &CheckCertJob{}

	err := checkCertJob.Init("* * * * *", "", 0, certList, nil)

	assert.Equal(t, "a valid notifier is required", err.Error())
}

func TestJobStartStop(t *testing.T) {
	checkCertJob := &CheckCertJob{}

	err := checkCertJob.Init("* * * * *", "", 0, certList, &mockNotifier{})
	assert.Nil(t, err)
	checkCertJob.Start()
	assert.True(t, checkCertJob.running)
	checkCertJob.Stop()
	assert.False(t, checkCertJob.running)
}

func TestTryExecuteNotDue(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	notifier := &mockNotifier{}
	checkCertJob.Init("0 0 1 1 1", "", 0, certList, &mockNotifier{})
	checkCertJob.notifier = notifier
	checkCertJob.tryExecute()

	assert.False(t, notifier.executed)
}

func TestTryExecuteDue(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	notifier := &mockNotifier{}
	checkCertJob.Init("* * * * *", "", 0, certList, &mockNotifier{})
	checkCertJob.notifier = notifier
	checkCertJob.tryExecute()

	assert.True(t, notifier.executed)
}

func TestExecuteNow(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	notifier := &mockNotifier{}
	checkCertJob.Init("* * * * *", "", 0, certList, &mockNotifier{})
	checkCertJob.notifier = notifier
	checkCertJob.RunNow()

	assert.True(t, notifier.executed)
}

func TestTryExecuteDueWarning(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	notifier := &mockNotifier{}
	checkCertJob.Init("* * * * *", "", 10000, certList, &mockNotifier{})
	checkCertJob.notifier = notifier
	checkCertJob.tryExecute()

	assert.True(t, notifier.executed)
}

func TestGetNotificationModelValidCertWithWarning(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	checkCertJob.Init("* * * * *", "", 30, certList, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 0, 15)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           true,
		ExpirationWarning: true,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{},
	}

	result := checkCertJob.getNotificationModel(cert)

	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Equal(t, "test.example.com", result.Hostname)
	assert.True(t, len(result.Messages) > 0, "Messages should contain expiration date")
	assert.True(t, strings.Contains(result.Messages[0], "Certificate expires in"), "Should contain expiration message")
	assert.True(t, strings.Contains(result.Messages[0], "days"), "Should contain 'days' in message")
}

func TestGetNotificationModelValidCertWithoutWarning(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	checkCertJob.Init("* * * * *", "", 30, certList, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 1, 0)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           true,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{},
	}

	result := checkCertJob.getNotificationModel(cert)

	assert.True(t, result.IsValid)
	assert.False(t, result.ExpirationWarning)
	assert.Equal(t, "test.example.com", result.Hostname)
	assert.True(t, len(result.Messages) > 0, "Messages should contain expiration date even without warning")
	assert.True(t, strings.Contains(result.Messages[0], "Certificate expires in"), "Should contain expiration message")
}

func TestGetNotificationModelValidCertWithValidationIssues(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	checkCertJob.Init("* * * * *", "", 30, certList, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 1, 0)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           true,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{"Issue 1", "Issue 2"},
	}

	result := checkCertJob.getNotificationModel(cert)

	assert.True(t, result.IsValid)
	assert.Equal(t, 3, len(result.Messages), "Should have 2 validation issues + 1 expiration message")
	assert.Equal(t, "Issue 1", result.Messages[0])
	assert.Equal(t, "Issue 2", result.Messages[1])
	assert.True(t, strings.Contains(result.Messages[2], "Certificate expires in"), "Third message should be expiration")
}

func TestGetNotificationModelInvalidCert(t *testing.T) {
	checkCertJob := &CheckCertJob{}
	checkCertJob.Init("* * * * *", "", 30, certList, &mockNotifier{})

	expirationDate := time.Now().AddDate(0, 0, -5)
	cert := &models.CertCheckResult{
		Hostname:          "test.example.com",
		IsValid:           false,
		ExpirationWarning: false,
		CertEndDate:       expirationDate,
		ValidationIssues:  []string{"Certificate expired"},
	}

	result := checkCertJob.getNotificationModel(cert)

	assert.False(t, result.IsValid)
	assert.Equal(t, 1, len(result.Messages), "Should only have validation issue, no expiration message for invalid certs")
	assert.Equal(t, "Certificate expired", result.Messages[0])
	assert.False(t, strings.Contains(strings.Join(result.Messages, " "), "Certificate expires in"))
}
