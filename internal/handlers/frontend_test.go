package handlers

import (
	"net/http"
	"testing"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestInitTemplate(t *testing.T) {
	templatePath = "../../frontend"
	indexTemplate = nil
	initTemplates()

	assert.NotNil(t, indexTemplate.Lookup("index.html"))
	assert.NotNil(t, indexTemplate.Lookup("body.html"))
	assert.NotNil(t, indexTemplate.Lookup("head.html"))
	assert.NotNil(t, indexTemplate.Lookup("certItem.html"))
	assert.NotNil(t, indexTemplate.Lookup("itemLoaded.html"))
	assert.NotNil(t, indexTemplate.Lookup("itemModal.html"))
	assert.NotNil(t, indexTemplate.Lookup("secretItem.html"))
	assert.NotNil(t, indexTemplate.Lookup("secretItemLoaded.html"))
	assert.NotNil(t, indexTemplate.Lookup("secretItemModal.html"))
}

func TestRendersIndex(t *testing.T) {
	templatePath = "../../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /", handlers.Index)

	code, _, body, _, err := makeRequest[string](router, "GET", "/", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	// Testing the full HTML content is not practical, so we check for specific elements
	assert.Contains(t, body, "data-testid=\"result-item\"")
	assert.Contains(t, body, "hx-get=\"/item?name=blog.lpains.net\"")
	assert.Contains(t, body, "id=\"tab-btn-certs\"")
	assert.Contains(t, body, "id=\"tab-btn-secrets\"")
	assert.Contains(t, body, "id=\"panel-certs\"")
	assert.Contains(t, body, "id=\"panel-secrets\"")
}

func TestRendersItem(t *testing.T) {
	templatePath = "../../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /item", handlers.GetItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/item?name=blog.lpains.net", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	// Testing the full HTML content is not practical, so we check for specific elements
	assert.Contains(t, body, "hx-get=\"/itemDetail?name=blog.lpains.net\" hx-trigger=\"click, keyup[key=='Enter']\" hx-target=\"#modal\"")
	assert.Contains(t, body, "<h2 class=\"text-white text-lg font-medium\">blog.lpains.net</h2>")
}

func TestRendersItemError(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blo.lpains.net", Url: "https://blo.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /item", handlers.GetItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/item?name=blo.lpains.net", nil)

	assert.Nil(t, err)
	assert.Equal(t, 500, code)
	assert.Contains(t, body, "Failed to process request")
}

func TestRendersItemNoName(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /item", handlers.GetItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/item", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "name is required")
}

func TestRendersItemBadName(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /item", handlers.GetItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/item?name=badname.bad.com", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "the provided cert name is not configured")
}

func TestRendersItemDetail(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /itemDetail", handlers.GetItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/itemDetail?name=blog.lpains.net", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	// Testing the full HTML content is not practical, so we check for specific elements
	assert.Contains(t, body, "<td class=\"px-4 py-2\">blog.lpains.net</td>")
}

func TestRendersItemDetailError(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blo.lpains.net", Url: "https://blo.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /itemDetail", handlers.GetItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/itemDetail?name=blo.lpains.net", nil)

	assert.Nil(t, err)
	assert.Equal(t, 500, code)
	assert.Contains(t, body, "Failed to process request")
}

func TestRendersItemDetailNoName(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /itemDetail", handlers.GetItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/itemDetail", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	// Testing the full HTML content is not practical, so we check for specific elements
	assert.Contains(t, body, "name is required")
}

func TestRendersItemDetailBadName(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.CertList = []models.CheckCertItem{
		{Name: "blog.lpains.net", Url: "https://blog.lpains.net", Type: models.CertCheckURL},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /itemDetail", handlers.GetItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/itemDetail?name=bad.name.com", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	// Testing the full HTML content is not practical, so we check for specific elements
	assert.Contains(t, body, "the provided cert name is not configured")
}

func TestRendersEmpty(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)

	router := http.NewServeMux()
	router.HandleFunc("GET /empty", handlers.GetEmpty)

	code, _, body, _, err := makeRequest[string](router, "GET", "/empty", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Equal(t, body, "")
}
