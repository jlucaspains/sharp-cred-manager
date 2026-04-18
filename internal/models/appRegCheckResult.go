package models

import "time"

type AppRegCredentialType int

const (
	AppRegCredentialSecret      AppRegCredentialType = iota
	AppRegCredentialCertificate AppRegCredentialType = iota
)

type AppRegCredentialResult struct {
	KeyId             string               `json:"keyId"`
	DisplayName       string               `json:"displayName"`
	CredentialType    AppRegCredentialType `json:"credentialType"`
	StartDateTime     *time.Time           `json:"startDateTime"`
	EndDateTime       *time.Time           `json:"endDateTime"`
	IsValid           bool                 `json:"isValid"`
	ValidationIssues  []string             `json:"validationIssues"`
	ExpirationWarning bool                 `json:"expirationWarning"`
	HasExpiration     bool                 `json:"hasExpiration"`
	ValidityInDays    int                  `json:"validityInDays"`
}

type AppRegCheckResult struct {
	Name              string                   `json:"name"`
	AppName           string                   `json:"appName"`
	AppId             string                   `json:"appId"`
	TenantId          string                   `json:"tenantId"`
	AppObjectId       string                   `json:"appObjectId"`
	IsValid           bool                     `json:"isValid"`
	ExpirationWarning bool                     `json:"expirationWarning"`
	Credentials       []AppRegCredentialResult `json:"credentials"`
}

type CheckAppRegItem struct {
	Name        string `json:"name"`
	TenantId    string `json:"tenantId"`
	AppId       string `json:"appId"`
	AppObjectId string `json:"appObjectId"`
	AppName     string `json:"appName"`
}
