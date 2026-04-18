package handlers

import (
	"net/http"
	"testing"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

var appRegList = []models.CheckAppRegItem{
	{
		Name:        "tenant-id/app-id",
		TenantId:    "tenant-id",
		AppId:       "app-id",
		AppObjectId: "object-id-1",
		AppName:     "TestApp",
	},
}

func TestGetAppRegList(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = appRegList

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-list", handlers.GetAppRegList)

	code, body, _, _, err := makeRequest[[]models.CheckAppRegItem](router, "GET", "/appreg-list", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, 1, len(*body))
}

func TestCheckAppRegStatusNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = appRegList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-appreg", handlers.CheckAppRegStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-appreg", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "name is required", body.Errors[0])
}

func TestCheckAppRegStatusInvalidName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = appRegList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-appreg", handlers.CheckAppRegStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-appreg?name=unknown-tenant/unknown-app", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "the provided app registration name is not configured", body.Errors[0])
}
