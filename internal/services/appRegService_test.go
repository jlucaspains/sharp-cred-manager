package services

import (
	"testing"
	"time"

	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func makeTimePtr(t time.Time) *string {
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func makeGraphApp(secrets []graphPasswordCredential, certs []graphKeyCredential) *graphApplication {
	return &graphApplication{
		ID:                  "object-id-1",
		DisplayName:         "TestApp",
		PasswordCredentials: secrets,
		KeyCredentials:      certs,
	}
}

func makeSecret(keyId, name string, start, end *string) graphPasswordCredential {
	return graphPasswordCredential{KeyId: keyId, DisplayName: name, StartDateTime: start, EndDateTime: end}
}

func makeCert(keyId, name string, start, end *string) graphKeyCredential {
	return graphKeyCredential{KeyId: keyId, DisplayName: name, StartDateTime: start, EndDateTime: end}
}

var testItem = models.CheckAppRegItem{
	Name:        "tenant-id/app-id",
	TenantId:    "tenant-id",
	AppId:       "app-id",
	AppObjectId: "object-id-1",
	AppName:     "TestApp",
}

func TestCheckAppRegStatus_ValidSecret(t *testing.T) {
	end := makeTimePtr(time.Now().UTC().Add(90 * 24 * time.Hour))
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{makeSecret("key-1", "CI Key", nil, end)}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.False(t, result.ExpirationWarning)
	assert.Len(t, result.Credentials, 1)
	assert.True(t, result.Credentials[0].IsValid)
	assert.Equal(t, models.AppRegCredentialSecret, result.Credentials[0].CredentialType)
	assert.Greater(t, result.Credentials[0].ValidityInDays, 0)
}

func TestCheckAppRegStatus_ValidCertificate(t *testing.T) {
	end := makeTimePtr(time.Now().UTC().Add(90 * 24 * time.Hour))
	mockGraphAppResult = makeGraphApp(nil, []graphKeyCredential{makeCert("key-2", "Auth Cert", nil, end)})
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.Len(t, result.Credentials, 1)
	assert.Equal(t, models.AppRegCredentialCertificate, result.Credentials[0].CredentialType)
}

func TestCheckAppRegStatus_MixedCredentials(t *testing.T) {
	end := makeTimePtr(time.Now().UTC().Add(90 * 24 * time.Hour))
	mockGraphAppResult = makeGraphApp(
		[]graphPasswordCredential{makeSecret("key-1", "CI Key", nil, end)},
		[]graphKeyCredential{makeCert("key-2", "Auth Cert", nil, end)},
	)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.Len(t, result.Credentials, 2)
}

func TestCheckAppRegStatus_ExpiredSecret(t *testing.T) {
	end := makeTimePtr(time.Now().UTC().Add(-24 * time.Hour))
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{makeSecret("key-1", "Old Key", nil, end)}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.False(t, result.Credentials[0].IsValid)
	assert.Contains(t, result.Credentials[0].ValidationIssues, "Credential is expired")
}

func TestCheckAppRegStatus_ExpirationWarning(t *testing.T) {
	end := makeTimePtr(time.Now().UTC().Add(15 * 24 * time.Hour))
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{makeSecret("key-1", "CI Key", nil, end)}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.True(t, result.ExpirationWarning)
	assert.True(t, result.Credentials[0].ExpirationWarning)
	assert.Greater(t, result.Credentials[0].ValidityInDays, 0)
}

func TestCheckAppRegStatus_NotYetActive(t *testing.T) {
	start := makeTimePtr(time.Now().UTC().Add(24 * time.Hour))
	end := makeTimePtr(time.Now().UTC().Add(90 * 24 * time.Hour))
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{makeSecret("key-1", "Future Key", start, end)}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.Contains(t, result.Credentials[0].ValidationIssues, "Credential is not yet active")
}

func TestCheckAppRegStatus_NoExpiration(t *testing.T) {
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{makeSecret("key-1", "No Expiry Key", nil, nil)}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.True(t, result.IsValid)
	assert.False(t, result.Credentials[0].HasExpiration)
	assert.Equal(t, 0, result.Credentials[0].ValidityInDays)
}

func TestCheckAppRegStatus_InvalidSecretInvalidatesApp(t *testing.T) {
	end1 := makeTimePtr(time.Now().UTC().Add(90 * 24 * time.Hour))
	end2 := makeTimePtr(time.Now().UTC().Add(-24 * time.Hour))
	mockGraphAppResult = makeGraphApp([]graphPasswordCredential{
		makeSecret("key-1", "Good Key", nil, end1),
		makeSecret("key-2", "Expired Key", nil, end2),
	}, nil)
	defer func() { mockGraphAppResult = nil }()

	result, err := CheckAppRegStatus(testItem, 30)

	assert.Nil(t, err)
	assert.False(t, result.IsValid)
	assert.True(t, result.Credentials[0].IsValid)
	assert.False(t, result.Credentials[1].IsValid)
}

func TestCheckAppRegStatus_EmptyItem(t *testing.T) {
	item := models.CheckAppRegItem{}

	_, err := CheckAppRegStatus(item, 30)

	assert.NotNil(t, err)
	assert.Equal(t, "name and appId are required", err.Error())
}

func TestCheckAppRegStatus_GraphError(t *testing.T) {
	mockGraphAppResult = nil

	_, err := CheckAppRegStatus(testItem, 30)

	assert.NotNil(t, err)
}

func TestGetConfigAppRegs_ValidEntry(t *testing.T) {
	t.Setenv("APPREGISTRATION_1", "my-tenant/my-app-id")

	mockGraphAppResult = &graphApplication{ID: "obj-id", DisplayName: "MyApp"}
	defer func() { mockGraphAppResult = nil }()

	items := GetConfigAppRegs()

	assert.Len(t, items, 1)
	assert.Equal(t, "my-tenant/my-app-id", items[0].Name)
	assert.Equal(t, "my-tenant", items[0].TenantId)
	assert.Equal(t, "my-app-id", items[0].AppId)
	assert.Equal(t, "obj-id", items[0].AppObjectId)
	assert.Equal(t, "MyApp", items[0].AppName)
}

func TestGetConfigAppRegs_InvalidFormat(t *testing.T) {
	t.Setenv("APPREGISTRATION_1", "no-slash-here")

	items := GetConfigAppRegs()

	assert.Len(t, items, 0)
}

func TestGetConfigAppRegs_GraphError(t *testing.T) {
	t.Setenv("APPREGISTRATION_1", "my-tenant/my-app-id")
	mockGraphAppResult = nil

	items := GetConfigAppRegs()

	assert.Len(t, items, 0)
}

func TestGetConfigAppRegs_NoEnvVars(t *testing.T) {
	items := GetConfigAppRegs()

	assert.Len(t, items, 0)
}

func TestGetConfigAppRegs_MultipleEntries(t *testing.T) {
	t.Setenv("APPREGISTRATION_1", "tenant-a/app-1")
	t.Setenv("APPREGISTRATION_2", "tenant-b/app-2")

	mockGraphAppResult = &graphApplication{ID: "obj-id", DisplayName: "SomeApp"}
	defer func() { mockGraphAppResult = nil }()

	items := GetConfigAppRegs()

	assert.Len(t, items, 2)
}
