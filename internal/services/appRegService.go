package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
)

// graphHTTPClient is a package-level HTTP client with a timeout, reused across all Graph requests.
var graphHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// mockGraphAppResult bypasses Graph HTTP calls for unit tests within this package.
var mockGraphAppResult *graphApplication = nil

// mockAppRegResult bypasses CheckAppRegStatus entirely; set via SetMockAppRegResult for use by other packages.
var mockAppRegResult *models.AppRegCheckResult = nil

func SetMockAppRegResult(mock *models.AppRegCheckResult) {
	mockAppRegResult = mock
}

type graphPasswordCredential struct {
	KeyId         string  `json:"keyId"`
	DisplayName   string  `json:"displayName"`
	StartDateTime *string `json:"startDateTime"`
	EndDateTime   *string `json:"endDateTime"`
}

type graphKeyCredential struct {
	KeyId         string  `json:"keyId"`
	DisplayName   string  `json:"displayName"`
	StartDateTime *string `json:"startDateTime"`
	EndDateTime   *string `json:"endDateTime"`
	Type          string  `json:"type"`
	Usage         string  `json:"usage"`
}

type graphApplication struct {
	ID                  string                    `json:"id"`
	DisplayName         string                    `json:"displayName"`
	PasswordCredentials []graphPasswordCredential `json:"passwordCredentials"`
	KeyCredentials      []graphKeyCredential      `json:"keyCredentials"`
}

type graphApplicationsResponse struct {
	Value []graphApplication `json:"value"`
}

func GetConfigAppRegs() []models.CheckAppRegItem {
	result := []models.CheckAppRegItem{}

	for i := 1; true; i++ {
		rawVal, ok := os.LookupEnv(fmt.Sprintf("APPREGISTRATION_%d", i))
		if !ok {
			break
		}

		parts := strings.SplitN(rawVal, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			log.Printf("Invalid APPREGISTRATION_%d value: %q (expected tenantId/appId)", i, rawVal)
			continue
		}

		tenantId, appId := parts[0], parts[1]
		name := tenantId + "/" + appId

		result = append(result, models.CheckAppRegItem{
			Name:        name,
			TenantId:    tenantId,
			AppId:       appId,
			AppObjectId: "", // Populated later in CheckAppRegStatus
			AppName:     "", // Populated later in CheckAppRegStatus
		})
	}

	return result
}

func CheckAppRegStatus(item models.CheckAppRegItem, warningDays int) (*models.AppRegCheckResult, error) {
	if mockAppRegResult != nil {
		return mockAppRegResult, nil
	}

	if item.Name == "" || item.AppId == "" {
		return nil, errors.New("name and appId are required")
	}

	app, err := getApplicationByAppId(item.AppId)
	if err != nil {
		return nil, err
	}

	return buildAppRegCheckResult(item, app, warningDays), nil
}

func buildAppRegCheckResult(item models.CheckAppRegItem, app *graphApplication, warningDays int) *models.AppRegCheckResult {
	result := &models.AppRegCheckResult{
		Name:        item.Name,
		AppName:     app.DisplayName,
		AppId:       item.AppId,
		TenantId:    item.TenantId,
		AppObjectId: app.ID,
		IsValid:     true,
		Credentials: []models.AppRegCredentialResult{},
	}

	for _, pc := range app.PasswordCredentials {
		cred := buildCredentialResult(pc.KeyId, pc.DisplayName, models.AppRegCredentialSecret, pc.StartDateTime, pc.EndDateTime, warningDays)
		if !cred.IsValid {
			result.IsValid = false
		}
		if cred.ExpirationWarning {
			result.ExpirationWarning = true
		}
		result.Credentials = append(result.Credentials, cred)
	}

	for _, kc := range app.KeyCredentials {
		cred := buildCredentialResult(kc.KeyId, kc.DisplayName, models.AppRegCredentialCertificate, kc.StartDateTime, kc.EndDateTime, warningDays)
		if !cred.IsValid {
			result.IsValid = false
		}
		if cred.ExpirationWarning {
			result.ExpirationWarning = true
		}
		result.Credentials = append(result.Credentials, cred)
	}

	return result
}

func buildCredentialResult(keyId, displayName string, credType models.AppRegCredentialType, start, end *string, warningDays int) models.AppRegCredentialResult {
	cred := models.AppRegCredentialResult{
		KeyId:            keyId,
		DisplayName:      displayName,
		CredentialType:   credType,
		IsValid:          true,
		ValidationIssues: []string{},
	}

	cred.StartDateTime = parseGraphDateTime(start)
	cred.EndDateTime = parseGraphDateTime(end)

	now := time.Now().UTC()

	if cred.StartDateTime != nil && cred.StartDateTime.After(now) {
		cred.IsValid = false
		cred.ValidationIssues = append(cred.ValidationIssues, "Credential is not yet active")
	}

	if cred.EndDateTime != nil {
		cred.HasExpiration = true
		endTime := cred.EndDateTime.UTC()
		if endTime.Before(now) {
			cred.IsValid = false
			cred.ValidationIssues = append(cred.ValidationIssues, "Credential is expired")
		} else {
			cred.ValidityInDays = int(endTime.Sub(now).Hours() / 24)
			if endTime.Before(now.AddDate(0, 0, warningDays)) {
				cred.ExpirationWarning = true
			}
		}
	}

	return cred
}

func parseGraphDateTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func getApplicationByAppId(appId string) (*graphApplication, error) {
	if mockGraphAppResult != nil {
		return mockGraphAppResult, nil
	}

	url := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/applications?$filter=appId%%20eq%%20'%s'&$select=id,displayName,passwordCredentials,keyCredentials",
		appId,
	)

	resp, err := makeGraphRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Graph API returned status %d for appId %s", resp.StatusCode, appId)
	}

	var result graphApplicationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Value) == 0 {
		return nil, fmt.Errorf("application with appId %s not found", appId)
	}

	return &result.Value[0], nil
}

func makeGraphRequest(url string) (*http.Response, error) {
	token, err := getGraphToken()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	return graphHTTPClient.Do(req)
}

func getGraphToken() (string, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", err
	}

	token, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	})
	if err != nil {
		return "", err
	}

	return token.Token, nil
}
