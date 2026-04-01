package handlers

import (
	"net/http"
	"testing"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
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

func TestGetSecretList(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = secretList

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-list", handlers.GetSecretList)

	code, body, _, _, err := makeRequest[[]models.CheckSecretItem](router, "GET", "/secret-list", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, 1, len(*body))
}

func TestCheckSecretStatusNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = secretList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-secret", handlers.CheckSecretStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "name is required", body.Errors[0])
}

func TestCheckSecretStatusInvalidName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = secretList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-secret", handlers.CheckSecretStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-secret?name=invalid", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "the provided secret name is not configured", body.Errors[0])
}
