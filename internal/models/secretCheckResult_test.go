package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecretCheckTypeConstants(t *testing.T) {
	assert.Equal(t, SecretCheckType(0), SecretCheckAzure)
}

func TestSecretCheckResult_JSONMarshal(t *testing.T) {
	expiresOn := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	notBefore := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	result := SecretCheckResult{
		Name:              "my-secret",
		Url:               "https://myvault.vault.azure.net/secrets/my-secret",
		Type:              SecretCheckAzure,
		ContentType:       "application/x-pkcs12",
		Enabled:           true,
		ExpiresOn:         &expiresOn,
		NotBefore:         &notBefore,
		IsValid:           true,
		ValidationIssues:  []string{},
		ExpirationWarning: false,
		ValidityInDays:    270,
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	var decoded SecretCheckResult
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, result.Name, decoded.Name)
	assert.Equal(t, result.Url, decoded.Url)
	assert.Equal(t, result.Type, decoded.Type)
	assert.Equal(t, result.ContentType, decoded.ContentType)
	assert.Equal(t, result.Enabled, decoded.Enabled)
	assert.Equal(t, result.IsValid, decoded.IsValid)
	assert.Equal(t, result.ExpirationWarning, decoded.ExpirationWarning)
	assert.Equal(t, result.ValidityInDays, decoded.ValidityInDays)
	assert.True(t, result.ExpiresOn.Equal(*decoded.ExpiresOn))
	assert.True(t, result.NotBefore.Equal(*decoded.NotBefore))
}

func TestSecretCheckResult_NilDates(t *testing.T) {
	result := SecretCheckResult{
		Name:    "no-expiry-secret",
		Enabled: true,
		IsValid: true,
	}

	data, err := json.Marshal(result)
	assert.NoError(t, err)

	var decoded SecretCheckResult
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Nil(t, decoded.ExpiresOn)
	assert.Nil(t, decoded.NotBefore)
}

func TestCheckSecretItem_JSONMarshal(t *testing.T) {
	item := CheckSecretItem{
		Name:       "vault-all",
		Url:        "https://myvault.vault.azure.net",
		Type:       SecretCheckAzure,
		SecretName: "",
	}

	data, err := json.Marshal(item)
	assert.NoError(t, err)

	var decoded CheckSecretItem
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, item.Name, decoded.Name)
	assert.Equal(t, item.Url, decoded.Url)
	assert.Equal(t, item.Type, decoded.Type)
	assert.Equal(t, item.SecretName, decoded.SecretName)
}

func TestCheckSecretItem_WithSecretName(t *testing.T) {
	item := CheckSecretItem{
		Name:       "specific-secret",
		Url:        "https://myvault.vault.azure.net/secrets/my-secret",
		Type:       SecretCheckAzure,
		SecretName: "my-secret",
	}

	assert.Equal(t, "my-secret", item.SecretName)
	assert.Equal(t, SecretCheckAzure, item.Type)
}
