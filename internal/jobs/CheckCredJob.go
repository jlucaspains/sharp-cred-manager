package jobs

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/adhocore/gronx"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

type Notifier interface {
	Notify(groups []CheckNotificationGroup) error
	IsReady() bool
}

type RawNotifier interface {
	Notifier
	NotifyRaw(
		certs   []*models.CertCheckResult,
		secrets []*models.SecretCheckResult,
		appRegs []*models.AppRegCheckResult,
	) error
}

type Level int

const (
	Info Level = iota
	Warning
	Error
)

var levels = map[string]Level{
	"Info":    Info,
	"Warning": Warning,
	"Error":   Error,
}

type CheckNotification struct {
	Name              string
	Source            string
	IsValid           bool
	Messages          []string
	ExpirationWarning bool
}

type CheckNotificationGroup struct {
	Label      string
	Items      []CheckNotification
	ShowSource bool
}

type CheckCredJob struct {
	cron              string
	ticker            *time.Ticker
	gron              *gronx.Gronx
	certList          []models.CheckCertItem
	secretList        []models.CheckSecretItem
	appRegList        []models.CheckAppRegItem
	running           bool
	notifier          Notifier
	level             Level
	certWarningDays   int
	secretWarningDays int
	appRegWarningDays int
}

type CheckCredJobConfig struct {
	Schedule          string
	Level             string
	CertWarningDays   int
	SecretWarningDays int
	AppRegWarningDays int
	CertList          []models.CheckCertItem
	SecretList        []models.CheckSecretItem
	AppRegList        []models.CheckAppRegItem
	Notifier          Notifier
}

func (c *CheckCredJob) Init(config CheckCredJobConfig) error {
	c.gron = gronx.New()

	if config.Schedule == "" || !c.gron.IsValid(config.Schedule) {
		log.Printf("A valid cron schedule is required in the format e.g.: * * * * *")
		return fmt.Errorf("a valid cron schedule is required")
	}

	if config.Notifier == nil || !config.Notifier.IsReady() {
		log.Printf("A valid notifier is required")
		return fmt.Errorf("a valid notifier is required")
	}

	levelValue, ok := levels[config.Level]
	if !ok {
		levelValue = Warning
	}

	if config.CertWarningDays <= 0 {
		config.CertWarningDays = 30
	}

	if config.SecretWarningDays <= 0 {
		config.SecretWarningDays = 30
	}

	if config.AppRegWarningDays <= 0 {
		config.AppRegWarningDays = 30
	}

	c.cron = config.Schedule
	c.certList = config.CertList
	c.secretList = config.SecretList
	c.appRegList = config.AppRegList
	c.ticker = time.NewTicker(time.Minute)
	c.notifier = config.Notifier
	c.level = levelValue
	c.certWarningDays = config.CertWarningDays
	c.secretWarningDays = config.SecretWarningDays
	c.appRegWarningDays = config.AppRegWarningDays

	return nil
}

func (c *CheckCredJob) RunNow() {
	c.execute()
}

func (c *CheckCredJob) Start() {
	c.running = true
	go func() {
		for range c.ticker.C {
			c.tryExecute()
		}
	}()
}

func (c *CheckCredJob) Stop() {
	c.running = false

	if c.ticker != nil {
		c.ticker.Stop()
	}
}

func (c *CheckCredJob) tryExecute() {
	due, _ := c.gron.IsDue(c.cron, time.Now().Truncate(time.Minute))

	log.Printf("tryExecute job, isDue: %t", due)

	if due {
		c.execute()
	}
}

func (c *CheckCredJob) execute() {
	certResults := c.checkCerts()
	secretResults := c.checkSecrets()
	appRegResults := c.checkAppRegs()

	if raw, ok := c.notifier.(RawNotifier); ok {
		if err := raw.NotifyRaw(
			c.filterCertResults(certResults),
			c.filterSecretResults(secretResults),
			c.filterAppRegResults(appRegResults),
		); err != nil {
			log.Printf("Error sending notification: %s", err)
		}
		return
	}

	groups := []CheckNotificationGroup{}
	if certGroup := c.buildCertGroup(certResults); len(certGroup.Items) > 0 {
		groups = append(groups, certGroup)
	}
	if secretGroup := c.buildSecretGroup(secretResults); len(secretGroup.Items) > 0 {
		groups = append(groups, secretGroup)
	}
	if appRegGroup := c.buildAppRegGroup(appRegResults); len(appRegGroup.Items) > 0 {
		groups = append(groups, appRegGroup)
	}

	if err := c.notifier.Notify(groups); err != nil {
		log.Printf("Error sending notification: %s", err)
	}
}

func (c *CheckCredJob) checkCerts() []*models.CertCheckResult {
	results := []*models.CertCheckResult{}
	for _, item := range c.certList {
		checkStatus, err := services.CheckCertStatus(item, c.certWarningDays)
		if err != nil {
			log.Printf("Error checking cert status: %s", err)
			continue
		}
		log.Printf("Cert status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}

func (c *CheckCredJob) checkSecrets() []*models.SecretCheckResult {
	results := []*models.SecretCheckResult{}
	for _, item := range c.secretList {
		checkStatus, err := services.CheckSecretStatus(item, c.secretWarningDays)
		if err != nil {
			log.Printf("Error checking secret status: %s", err)
			continue
		}
		log.Printf("Secret status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}

func (c *CheckCredJob) checkAppRegs() []*models.AppRegCheckResult {
	results := []*models.AppRegCheckResult{}
	for _, item := range c.appRegList {
		checkStatus, err := services.CheckAppRegStatus(item, c.appRegWarningDays)
		if err != nil {
			log.Printf("Error checking app registration status: %s", err)
			continue
		}
		log.Printf("App registration status for %s: %t", item.Name, checkStatus.IsValid)
		results = append(results, checkStatus)
	}
	return results
}

func (c *CheckCredJob) filterCertResults(results []*models.CertCheckResult) []*models.CertCheckResult {
	filtered := []*models.CertCheckResult{}
	for _, r := range results {
		if c.shouldNotify(CheckNotification{IsValid: r.IsValid, ExpirationWarning: r.ExpirationWarning}) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (c *CheckCredJob) filterSecretResults(results []*models.SecretCheckResult) []*models.SecretCheckResult {
	filtered := []*models.SecretCheckResult{}
	for _, r := range results {
		if c.shouldNotify(CheckNotification{IsValid: r.IsValid, ExpirationWarning: r.ExpirationWarning}) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (c *CheckCredJob) filterAppRegResults(results []*models.AppRegCheckResult) []*models.AppRegCheckResult {
	filtered := []*models.AppRegCheckResult{}
	for _, r := range results {
		filteredCreds := []models.AppRegCredentialResult{}
		for _, cred := range r.Credentials {
			if c.shouldNotify(CheckNotification{IsValid: cred.IsValid, ExpirationWarning: cred.ExpirationWarning}) {
				filteredCreds = append(filteredCreds, cred)
			}
		}
		if len(filteredCreds) > 0 {
			filtered = append(filtered, &models.AppRegCheckResult{
				Name:              r.Name,
				AppName:           r.AppName,
				AppId:             r.AppId,
				AppObjectId:       r.AppObjectId,
				IsValid:           r.IsValid,
				ExpirationWarning: r.ExpirationWarning,
				Credentials:       filteredCreds,
			})
		}
	}
	return filtered
}

func (c *CheckCredJob) buildCertGroup(results []*models.CertCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "Certificates", ShowSource: true}
	for _, checkStatus := range results {
		n := c.getCertNotification(checkStatus)
		if c.shouldNotify(n) {
			group.Items = append(group.Items, n)
		}
	}
	return group
}

func (c *CheckCredJob) buildSecretGroup(results []*models.SecretCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "Secrets", ShowSource: true}
	for _, checkStatus := range results {
		n := c.getSecretNotification(checkStatus)
		if c.shouldNotify(n) {
			group.Items = append(group.Items, n)
		}
	}
	return group
}

func (c *CheckCredJob) buildAppRegGroup(results []*models.AppRegCheckResult) CheckNotificationGroup {
	group := CheckNotificationGroup{Label: "App Registrations", ShowSource: true}
	for _, checkStatus := range results {
		for _, cred := range checkStatus.Credentials {
			n := c.getCredentialNotification(checkStatus.AppName, cred)
			if c.shouldNotify(n) {
				group.Items = append(group.Items, n)
			}
		}
	}
	return group
}

func (c *CheckCredJob) shouldNotify(model CheckNotification) bool {
	return c.level == Info || !model.IsValid || (c.level == Warning && model.ExpirationWarning)
}

func (c *CheckCredJob) getCertNotification(certificate *models.CertCheckResult) CheckNotification {
	source := strings.TrimPrefix(certificate.Source, "https://")
	source = strings.TrimPrefix(source, "http://")
	source = strings.TrimSuffix(source, ".vault.azure.net")

	result := CheckNotification{
		Name:              certificate.DisplayName,
		Source:            source,
		IsValid:           certificate.IsValid,
		ExpirationWarning: certificate.ExpirationWarning,
		Messages:          certificate.ValidationIssues,
	}

	if certificate.IsValid {
		days := int(time.Until(certificate.CertEndDate).Hours() / 24)
		result.Messages = append(result.Messages, fmt.Sprintf("Certificate expires in %d days", days))
	}

	return result
}

func (c *CheckCredJob) getSecretNotification(secret *models.SecretCheckResult) CheckNotification {
	source := strings.TrimSuffix(strings.Replace(secret.Source, ".vault.azure.net", "", 1), "/")

	result := CheckNotification{
		Name:              secret.DisplayName,
		Source:            source,
		IsValid:           secret.IsValid,
		ExpirationWarning: secret.ExpirationWarning,
		Messages:          secret.ValidationIssues,
	}

	if secret.IsValid && secret.ExpiresOn != nil {
		days := int(time.Until(*secret.ExpiresOn).Hours() / 24)
		result.Messages = append(result.Messages, fmt.Sprintf("Secret expires in %d days", days))
	}

	return result
}

func (c *CheckCredJob) getCredentialNotification(appName string, cred models.AppRegCredentialResult) CheckNotification {
	icon := "🔑"
	if cred.CredentialType == models.AppRegCredentialCertificate {
		icon = "📜"
	}

	result := CheckNotification{
		Source:            appName,
		Name:              icon + " " + cred.DisplayName,
		IsValid:           cred.IsValid,
		ExpirationWarning: cred.ExpirationWarning,
		Messages:          []string{},
	}

	if !cred.IsValid {
		result.Messages = append(result.Messages, cred.ValidationIssues...)
	} else if cred.ExpirationWarning {
		result.Messages = []string{fmt.Sprintf("expires in %d days", cred.ValidityInDays)}
	}

	return result
}
