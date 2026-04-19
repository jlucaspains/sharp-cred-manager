package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/adhocore/gronx"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

type Notifier interface {
	Notify(groups []CheckNotificationGroup) error
	IsReady() bool
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
	IsValid           bool
	Messages          []string
	ExpirationWarning bool
}

type CheckNotificationGroup struct {
	Label string
	Items []CheckNotification
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
	certGroup := CheckNotificationGroup{Label: "Certificates"}
	for _, item := range c.certList {
		checkStatus, err := services.CheckCertStatus(item, c.certWarningDays)

		if err != nil {
			log.Printf("Error checking cert status: %s", err)
			continue
		}

		log.Printf("Cert status for %s: %t", item.Name, checkStatus.IsValid)

		n := c.getCertNotification(checkStatus)
		if c.shouldNotify(n) {
			certGroup.Items = append(certGroup.Items, n)
		}
	}

	secretGroup := CheckNotificationGroup{Label: "Secrets"}
	for _, item := range c.secretList {
		checkStatus, err := services.CheckSecretStatus(item, c.secretWarningDays)

		if err != nil {
			log.Printf("Error checking secret status: %s", err)
			continue
		}

		log.Printf("Secret status for %s: %t", item.Name, checkStatus.IsValid)

		n := c.getSecretNotification(checkStatus)
		if c.shouldNotify(n) {
			secretGroup.Items = append(secretGroup.Items, n)
		}
	}

	appRegGroup := CheckNotificationGroup{Label: "App Registrations"}
	for _, item := range c.appRegList {
		checkStatus, err := services.CheckAppRegStatus(item, c.appRegWarningDays)

		if err != nil {
			log.Printf("Error checking app registration status: %s", err)
			continue
		}

		log.Printf("App registration status for %s: %t", item.Name, checkStatus.IsValid)

		n := c.getAppRegNotification(checkStatus)
		if c.shouldNotify(n) {
			appRegGroup.Items = append(appRegGroup.Items, n)
		}
	}

	groups := []CheckNotificationGroup{}
	if len(certGroup.Items) > 0 {
		groups = append(groups, certGroup)
	}
	if len(secretGroup.Items) > 0 {
		groups = append(groups, secretGroup)
	}
	if len(appRegGroup.Items) > 0 {
		groups = append(groups, appRegGroup)
	}

	if err := c.notifier.Notify(groups); err != nil {
		log.Printf("Error sending notification: %s", err)
	}
}

func (c *CheckCredJob) shouldNotify(model CheckNotification) bool {
	return c.level == Info || !model.IsValid || (c.level == Warning && model.ExpirationWarning)
}

func (c *CheckCredJob) getCertNotification(certificate *models.CertCheckResult) CheckNotification {
	result := CheckNotification{
		Name:              certificate.Hostname,
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
	result := CheckNotification{
		Name:              secret.Name,
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

func (c *CheckCredJob) getAppRegNotification(appReg *models.AppRegCheckResult) CheckNotification {
	result := CheckNotification{
		Name:              appReg.AppName,
		IsValid:           appReg.IsValid,
		ExpirationWarning: appReg.ExpirationWarning,
		Messages:          []string{},
	}

	for _, cred := range appReg.Credentials {
		credType := "Secret"
		if cred.CredentialType == models.AppRegCredentialCertificate {
			credType = "Certificate"
		}
		label := fmt.Sprintf("%s (%s)", cred.DisplayName, credType)

		if !cred.IsValid {
			for _, issue := range cred.ValidationIssues {
				result.Messages = append(result.Messages, fmt.Sprintf("%s: %s", label, issue))
			}
		} else if cred.ExpirationWarning {
			result.Messages = append(result.Messages, fmt.Sprintf("%s: expires in %d days", label, cred.ValidityInDays))
		}
	}

	return result
}
