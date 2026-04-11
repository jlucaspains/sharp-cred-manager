package models

import "time"

type SecretCheckType int

const (
	SecretCheckAzure SecretCheckType = iota
)

type SecretCheckResult struct {
	Name              string          `json:"name"`
	DisplayName       string          `json:"displayName"`
	Source            string          `json:"source"`
	Url               string          `json:"url"`
	Type              SecretCheckType `json:"type"`
	ContentType       string          `json:"contentType"`
	Enabled           bool            `json:"enabled"`
	ExpiresOn         *time.Time      `json:"expiresOn"`
	NotBefore         *time.Time      `json:"notBefore"`
	IsValid           bool            `json:"isValid"`
	ValidationIssues  []string        `json:"validationIssues"`
	ExpirationWarning bool            `json:"expirationWarning"`
	HasExpiration     bool            `json:"hasExpiration"`
	ValidityInDays    int             `json:"validityInDays"`
}

type CheckSecretItem struct {
	Name       string          `json:"name"`
	Url        string          `json:"url"`
	Type       SecretCheckType `json:"type"`
	SecretName string          `json:"secretName"`
}
