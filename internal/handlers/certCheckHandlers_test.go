package handlers

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

var certList = []models.CheckCertItem{
	{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
}

func TestGetCertList(t *testing.T) {
	godotenv.Load("../.test.env")
	handlers := new(Handlers)
	handlers.CertList = certList

	router := http.NewServeMux()
	router.HandleFunc("GET /cert-list", handlers.GetCertList)

	code, body, _, _, err := makeRequest[[]models.CheckCertItem](router, "GET", "/cert-list", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, 1, len(*body))
}

func TestGetCheckStatus(t *testing.T) {
	handlers := new(Handlers)
	handlers.CertList = certList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-cert", handlers.CheckCertStatus)

	url := fmt.Sprintf("/check-cert?name=%s", "blog.lpains.net")
	code, body, _, _, err := makeRequest[models.CertCheckResult](router, "GET", url, nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.True(t, body.IsValid)
	assert.LessOrEqual(t, body.CertStartDate, time.Now())
	assert.GreaterOrEqual(t, body.CertEndDate, time.Now())
	assert.Contains(t, body.Hostname, "blog.lpains.net")
	assert.Contains(t, body.CertDnsNames, "blog.lpains.net")
}

func TestGetCheckStatusNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.CertList = certList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-cert", handlers.CheckCertStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-cert", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "name is required", body.Errors[0])
}

func TestGetCheckStatusInvalidName(t *testing.T) {
	handlers := new(Handlers)
	handlers.CertList = certList

	router := http.NewServeMux()
	router.HandleFunc("GET /check-cert", handlers.CheckCertStatus)

	code, body, _, _, err := makeRequest[models.ErrorResult](router, "GET", "/check-cert?name=invalid", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Equal(t, "the provided cert name is not configured", body.Errors[0])
}
