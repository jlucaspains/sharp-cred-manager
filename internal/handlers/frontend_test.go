package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
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
	assert.NotNil(t, indexTemplate.Lookup("secretsPanel.html"))
	assert.NotNil(t, indexTemplate.Lookup("secretItemLoaded.html"))
	assert.NotNil(t, indexTemplate.Lookup("secretItemModal.html"))
	assert.NotNil(t, indexTemplate.Lookup("appRegItem.html"))
	assert.NotNil(t, indexTemplate.Lookup("appRegsPanel.html"))
	assert.NotNil(t, indexTemplate.Lookup("appRegItemLoaded.html"))
	assert.NotNil(t, indexTemplate.Lookup("appRegItemModal.html"))
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
	assert.Contains(t, body, "id=\"tab-btn-appregs\"")
	assert.Contains(t, body, "id=\"panel-certs\"")
	assert.Contains(t, body, "id=\"panel-secrets\"")
	assert.Contains(t, body, "id=\"panel-appregs\"")
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

func TestRendersSecretsPanel(t *testing.T) {
	templatePath = "../../frontend"
	indexTemplate = nil
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secrets-panel", handlers.GetSecretsPanel)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secrets-panel", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "data-testid=\"result-item\"")
	assert.Contains(t, body, "hx-get=\"/secret-item?name=testfake.vault.azure.net/my-secret\"")
}

func TestRendersSecretItem(t *testing.T) {
	templatePath = "../../frontend"
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	enabled := true
	contentType := "text/plain"
	services.SetMockSecretResult(&azsecrets.SecretProperties{
		Attributes: &azsecrets.SecretAttributes{
			Enabled: &enabled,
			Expires: &expiresAt,
		},
		ContentType: &contentType,
	})
	defer services.SetMockSecretResult(nil)

	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item", handlers.GetSecretItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item?name=testfake.vault.azure.net/my-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "hx-get=\"/secret-item-detail?name=testfake.vault.azure.net/my-secret\"")
	assert.Contains(t, body, "<h2 class=\"text-white text-lg font-medium\">my-secret</h2>")
}

func TestRendersSecretItemError(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item", handlers.GetSecretItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item?name=testfake.vault.azure.net/my-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 500, code)
	assert.Contains(t, body, "Failed to process request")
}

func TestRendersSecretItemNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item", handlers.GetSecretItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "name is required")
}

func TestRendersSecretItemBadName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item", handlers.GetSecretItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item?name=unknown.vault.azure.net/bad-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "the provided secret name is not configured")
}

func TestRendersSecretItemDetail(t *testing.T) {
	templatePath = "../../frontend"
	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	enabled := true
	contentType := "text/plain"
	services.SetMockSecretResult(&azsecrets.SecretProperties{
		Attributes: &azsecrets.SecretAttributes{
			Enabled: &enabled,
			Expires: &expiresAt,
		},
		ContentType: &contentType,
	})
	defer services.SetMockSecretResult(nil)

	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item-detail", handlers.GetSecretItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item-detail?name=testfake.vault.azure.net/my-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "<td class=\"px-4 py-2\">testfake.vault.azure.net/my-secret</td>")
}

func TestRendersSecretItemDetailError(t *testing.T) {
	templatePath = "../frontend"
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item-detail", handlers.GetSecretItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item-detail?name=testfake.vault.azure.net/my-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 500, code)
	assert.Contains(t, body, "Failed to process request")
}

func TestRendersSecretItemDetailNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item-detail", handlers.GetSecretItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item-detail", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "name is required")
}

func TestRendersSecretItemDetailBadName(t *testing.T) {
	handlers := new(Handlers)
	handlers.SecretList = []models.CheckSecretItem{
		{Name: "testfake.vault.azure.net/my-secret", Url: "https://testfake.vault.azure.net/secrets/my-secret", Type: models.SecretCheckAzure, SecretName: "my-secret"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /secret-item-detail", handlers.GetSecretItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/secret-item-detail?name=unknown.vault.azure.net/bad-secret", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "the provided secret name is not configured")
}

func TestRendersAppRegsPanel(t *testing.T) {
	templatePath = "../../frontend"
	indexTemplate = nil
	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appregs-panel", handlers.GetAppRegsPanel)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appregs-panel", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "data-testid=\"result-item\"")
	assert.Contains(t, body, "hx-get=\"/appreg-item?name=tenant-id/app-id\"")
}

func TestRendersAppRegItem(t *testing.T) {
	templatePath = "../../frontend"
	indexTemplate = nil

	end := time.Now().UTC().Add(90 * 24 * time.Hour)
	services.SetMockAppRegResult(&models.AppRegCheckResult{
		Name:    "tenant-id/app-id",
		AppName: "TestApp",
		AppId:   "app-id",
		IsValid: true,
		Credentials: []models.AppRegCredentialResult{
			{KeyId: "k1", DisplayName: "CI Key", CredentialType: models.AppRegCredentialSecret, IsValid: true, HasExpiration: true, ValidityInDays: 90, EndDateTime: &end},
		},
	})
	defer services.SetMockAppRegResult(nil)

	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item", handlers.GetAppRegItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item?name=tenant-id/app-id", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "hx-get=\"/appreg-item-detail?name=tenant-id/app-id\"")
	assert.Contains(t, body, "<h2 class=\"text-white text-lg font-medium\">TestApp</h2>")
}

func TestRendersAppRegItemNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item", handlers.GetAppRegItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "name is required")
}

func TestRendersAppRegItemBadName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item", handlers.GetAppRegItem)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item?name=bad-tenant/bad-app", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "the provided app registration name is not configured")
}

func TestRendersAppRegItemDetail(t *testing.T) {
	templatePath = "../../frontend"
	indexTemplate = nil

	services.SetMockAppRegResult(&models.AppRegCheckResult{
		Name:     "tenant-id/app-id",
		AppName:  "TestApp",
		AppId:    "app-id",
		TenantId: "tenant-id",
		IsValid:  true,
		Credentials: []models.AppRegCredentialResult{
			{KeyId: "k1", DisplayName: "CI Key", CredentialType: models.AppRegCredentialSecret, IsValid: true, HasExpiration: false},
		},
	})
	defer services.SetMockAppRegResult(nil)

	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item-detail", handlers.GetAppRegItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item-detail?name=tenant-id/app-id", nil)

	assert.Nil(t, err)
	assert.Equal(t, 200, code)
	assert.Contains(t, body, "TestApp")
	assert.Contains(t, body, "app-id")
	assert.Contains(t, body, "tenant-id")
}

func TestRendersAppRegItemDetailNoName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item-detail", handlers.GetAppRegItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item-detail", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "name is required")
}

func TestRendersAppRegItemDetailBadName(t *testing.T) {
	handlers := new(Handlers)
	handlers.AppRegList = []models.CheckAppRegItem{
		{Name: "tenant-id/app-id", TenantId: "tenant-id", AppId: "app-id", AppObjectId: "obj-id", AppName: "TestApp"},
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /appreg-item-detail", handlers.GetAppRegItemDetail)

	code, _, body, _, err := makeRequest[string](router, "GET", "/appreg-item-detail?name=bad-tenant/bad-app", nil)

	assert.Nil(t, err)
	assert.Equal(t, 400, code)
	assert.Contains(t, body, "the provided app registration name is not configured")
}
