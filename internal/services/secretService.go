package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
)

var mockSecretResult *azsecrets.Secret = nil
var mockSecretListResult []*azsecrets.SecretProperties = nil

// SetMockSecretResult injects a mock secret for use in tests across packages.
func SetMockSecretResult(mock *azsecrets.Secret) {
	mockSecretResult = mock
}

func GetConfigSecrets() []models.CheckSecretItem {
	result := []models.CheckSecretItem{}
	for i := 1; true; i++ {
		rawUrl, ok := os.LookupEnv(fmt.Sprintf("AZUREKEYVAULTSECRET_%d", i))
		if !ok {
			break
		}

		parsedUrl, err := url.Parse(rawUrl)
		if err != nil {
			continue
		}

		pathParts := strings.Split(parsedUrl.Path, "/")
		// pathParts for "/secrets/my-secret" = ["", "secrets", "my-secret"]
		if len(pathParts) >= 3 && pathParts[2] != "" {
			secretName := pathParts[2]
			name := parsedUrl.Hostname() + "/" + secretName
			result = append(result, models.CheckSecretItem{
				Name:       name,
				Url:        rawUrl,
				Type:       models.SecretCheckAzure,
				SecretName: secretName,
			})
		} else {
			vaultUrl := parsedUrl.Scheme + "://" + parsedUrl.Host
			secrets, err := listSecretsFromVault(vaultUrl)
			if err != nil {
				log.Printf("Error listing secrets from vault %s: %s", vaultUrl, err)
				continue
			}
			for _, s := range secrets {
				if s.ID == nil {
					continue
				}
				secretUrl := string(*s.ID)
				parsedSecretUrl, err := url.Parse(secretUrl)
				if err != nil {
					continue
				}
				secretPathParts := strings.Split(parsedSecretUrl.Path, "/")
				if len(secretPathParts) < 3 || secretPathParts[2] == "" {
					continue
				}
				secretName := secretPathParts[2]
				name := parsedUrl.Hostname() + "/" + secretName
				cleanUrl := parsedSecretUrl.Scheme + "://" + parsedSecretUrl.Host + "/secrets/" + secretName
				result = append(result, models.CheckSecretItem{
					Name:       name,
					Url:        cleanUrl,
					Type:       models.SecretCheckAzure,
					SecretName: secretName,
				})
			}
		}
	}
	return result
}

func CheckSecretStatus(item models.CheckSecretItem, warningDays int) (*models.SecretCheckResult, error) {
	if item.Name == "" || item.Url == "" {
		return nil, errors.New("name and url are required")
	}

	switch item.Type {
	case models.SecretCheckAzure:
		return checkAzureSecretStatus(item, warningDays)
	}

	return nil, errors.New("invalid type")
}

func checkAzureSecretStatus(item models.CheckSecretItem, warningDays int) (*models.SecretCheckResult, error) {
	parsedUrl, _ := url.Parse(item.Url)
	vaultUrl := parsedUrl.Scheme + "://" + parsedUrl.Host

	secret, err := getSecretFromKeyVault(vaultUrl, item.SecretName)
	if err != nil {
		return nil, err
	}

	return buildSecretCheckResult(item, secret, warningDays), nil
}

func buildSecretCheckResult(item models.CheckSecretItem, secret *azsecrets.Secret, warningDays int) *models.SecretCheckResult {
	result := &models.SecretCheckResult{
		Name:             item.Name,
		Url:              item.Url,
		Type:             item.Type,
		IsValid:          true,
		ValidationIssues: []string{},
	}

	if secret.ContentType != nil {
		result.ContentType = *secret.ContentType
	}

	if secret.Attributes != nil {
		if secret.Attributes.Enabled != nil {
			result.Enabled = *secret.Attributes.Enabled
			if !*secret.Attributes.Enabled {
				result.IsValid = false
				result.ValidationIssues = append(result.ValidationIssues, "Secret is disabled")
			}
		}

		result.ExpiresOn = secret.Attributes.Expires
		result.NotBefore = secret.Attributes.NotBefore

		now := time.Now().UTC()

		if secret.Attributes.NotBefore != nil && secret.Attributes.NotBefore.After(now) {
			result.IsValid = false
			result.ValidationIssues = append(result.ValidationIssues, "Secret is not yet active")
		}

		if secret.Attributes.Expires != nil {
			expiresOn := secret.Attributes.Expires.UTC()
			if expiresOn.Before(now) {
				result.IsValid = false
				result.ValidationIssues = append(result.ValidationIssues, "Secret is expired")
			} else {
				result.ValidityInDays = int(expiresOn.Sub(now).Hours() / 24)
				if expiresOn.Before(now.AddDate(0, 0, warningDays)) {
					result.ExpirationWarning = true
				}
			}
		}
	}

	return result
}

func getSecretFromKeyVault(vaultUrl, secretName string) (*azsecrets.Secret, error) {
	if mockSecretResult != nil {
		return mockSecretResult, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	secretClient, err := azsecrets.NewClient(vaultUrl, cred, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("Getting secret from Azure Key Vault: %s", secretName)

	response, err := secretClient.GetSecret(context.Background(), secretName, "", nil)
	if err != nil {
		return nil, err
	}

	secret := response.Secret
	return &secret, nil
}

func listSecretsFromVault(vaultUrl string) ([]*azsecrets.SecretProperties, error) {
	if mockSecretListResult != nil {
		return mockSecretListResult, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	secretClient, err := azsecrets.NewClient(vaultUrl, cred, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("Listing secrets from Azure Key Vault: %s", vaultUrl)

	pager := secretClient.NewListSecretPropertiesPager(nil)
	var results []*azsecrets.SecretProperties
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		results = append(results, page.Value...)
	}

	return results, nil
}
