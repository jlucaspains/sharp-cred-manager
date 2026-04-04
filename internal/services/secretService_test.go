package services

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func createMockSecret(enabled bool, expiresAt time.Time, notBefore *time.Time) *azsecrets.SecretProperties {
	contentType := "text/plain"
	return &azsecrets.SecretProperties{
		Attributes: &azsecrets.SecretAttributes{
			Enabled:   &enabled,
			Expires:   &expiresAt,
			NotBefore: notBefore,
		},
		ContentType: &contentType,
	}
}

func TestCheckSecretStatus_Valid(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	mockSecretResult = createMockSecret(true, expiresAt, nil)
	defer func() { mockSecretResult = nil }()

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	result, err := CheckSecretStatus(item, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.True(t, result.Enabled)
	assert.Empty(t, result.ValidationIssues)
	assert.False(t, result.ExpirationWarning)
	assert.Greater(t, result.ValidityInDays, 0)
	assert.Equal(t, "text/plain", result.ContentType)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", result.Name)
	assert.Equal(t, "https://testfake.vault.azure.net/secrets/my-secret", result.Url)
}

func TestCheckSecretStatus_Disabled(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	mockSecretResult = createMockSecret(false, expiresAt, nil)
	defer func() { mockSecretResult = nil }()

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	result, err := CheckSecretStatus(item, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.False(t, result.Enabled)
	assert.Contains(t, result.ValidationIssues, "Secret is disabled")
}

func TestCheckSecretStatus_Expired(t *testing.T) {
	expiresAt := time.Now().UTC().Add(-24 * time.Hour)
	mockSecretResult = createMockSecret(true, expiresAt, nil)
	defer func() { mockSecretResult = nil }()

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	result, err := CheckSecretStatus(item, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.Contains(t, result.ValidationIssues, "Secret is expired")
}

func TestCheckSecretStatus_ExpirationWarning(t *testing.T) {
	expiresAt := time.Now().UTC().Add(15 * 24 * time.Hour)
	mockSecretResult = createMockSecret(true, expiresAt, nil)
	defer func() { mockSecretResult = nil }()

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	result, err := CheckSecretStatus(item, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.Greater(t, result.ValidityInDays, 0)
}

func TestCheckSecretStatus_NotYetActive(t *testing.T) {
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	notBefore := time.Now().UTC().Add(24 * time.Hour)
	mockSecretResult = createMockSecret(true, expiresAt, &notBefore)
	defer func() { mockSecretResult = nil }()

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	result, err := CheckSecretStatus(item, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.Contains(t, result.ValidationIssues, "Secret is not yet active")
}

func TestCheckSecretStatus_InvalidAzure(t *testing.T) {
	mockSecretResult = nil

	item := models.CheckSecretItem{
		Name:       "testfake.vault.azure.net/my-secret",
		Url:        "https://testfake.vault.azure.net/secrets/my-secret",
		Type:       models.SecretCheckAzure,
		SecretName: "my-secret",
	}

	_, err := CheckSecretStatus(item, 30)

	assert.NotNil(t, err)
}

func TestCheckSecretStatus_EmptyName(t *testing.T) {
	item := models.CheckSecretItem{Name: "", Url: "", Type: models.SecretCheckAzure}

	_, err := CheckSecretStatus(item, 30)

	assert.NotNil(t, err)
	assert.Equal(t, "name and url are required", err.Error())
}

func TestCheckSecretStatus_InvalidType(t *testing.T) {
	item := models.CheckSecretItem{
		Name: "test",
		Url:  "https://testfake.vault.azure.net/secrets/my-secret",
		Type: models.SecretCheckType(99),
	}

	_, err := CheckSecretStatus(item, 30)

	assert.NotNil(t, err)
	assert.Equal(t, "invalid type", err.Error())
}

func TestGetConfigSecrets_SpecificSecret(t *testing.T) {
	t.Setenv("AZUREKEYVAULTSECRET_1", "https://testfake.vault.azure.net/secrets/my-secret")

	secrets := GetConfigSecrets()

	assert.Len(t, secrets, 1)
	assert.Equal(t, "testfake.vault.azure.net/my-secret", secrets[0].Name)
	assert.Equal(t, "https://testfake.vault.azure.net/secrets/my-secret", secrets[0].Url)
	assert.Equal(t, models.SecretCheckAzure, secrets[0].Type)
	assert.Equal(t, "my-secret", secrets[0].SecretName)
}

func TestGetConfigSecrets_VaultOnly(t *testing.T) {
	t.Setenv("AZUREKEYVAULTSECRET_1", "https://testfake.vault.azure.net")

	id := azsecrets.ID("https://testfake.vault.azure.net/secrets/vault-secret/someversion")
	mockSecretListResult = []*azsecrets.SecretProperties{
		{ID: &id},
	}
	defer func() { mockSecretListResult = nil }()

	secrets := GetConfigSecrets()

	assert.Len(t, secrets, 1)
	assert.Equal(t, "testfake.vault.azure.net/vault-secret", secrets[0].Name)
	assert.Equal(t, "https://testfake.vault.azure.net/secrets/vault-secret", secrets[0].Url)
	assert.Equal(t, models.SecretCheckAzure, secrets[0].Type)
	assert.Equal(t, "vault-secret", secrets[0].SecretName)
}
