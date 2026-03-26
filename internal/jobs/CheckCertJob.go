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
	Notify(result []CertCheckNotification) error
	IsReady() bool
}

type CheckCertJob struct {
	cron        string
	ticker      *time.Ticker
	gron        *gronx.Gronx
	certList    []models.CheckCertItem
	running     bool
	notifier    Notifier
	level       Level
	warningDays int
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

type CertCheckNotification struct {
	Hostname          string
	IsValid           bool
	Messages          []string
	ExpirationWarning bool
}

func (c *CheckCertJob) Init(schedule string, level string, warningDays int, certList []models.CheckCertItem, notifier Notifier) error {
	c.gron = gronx.New()

	if schedule == "" || !c.gron.IsValid(schedule) {
		log.Printf("A valid cron schedule is required in the format e.g.: * * * * *")
		return fmt.Errorf("a valid cron schedule is required")
	}

	if notifier == nil || !notifier.IsReady() {
		log.Printf("A valid notifier is required")
		return fmt.Errorf("a valid notifier is required")
	}

	levelValue, ok := levels[level]
	if !ok {
		levelValue = Warning
	}

	if warningDays <= 0 {
		warningDays = 30
	}

	c.cron = schedule
	c.certList = certList
	c.ticker = time.NewTicker(time.Minute)
	c.notifier = notifier
	c.level = levelValue
	c.warningDays = warningDays

	return nil
}

func (c *CheckCertJob) RunNow() {
	c.execute()
}

func (c *CheckCertJob) Start() {
	c.running = true
	go func() {
		for range c.ticker.C {
			c.tryExecute()
		}
	}()
}

func (c *CheckCertJob) Stop() {
	c.running = false

	if c.ticker != nil {
		c.ticker.Stop()
	}
}

func (c *CheckCertJob) tryExecute() {
	due, _ := c.gron.IsDue(c.cron, time.Now().Truncate(time.Minute))

	log.Printf("tryExecute job, isDue: %t", due)

	if due {
		c.execute()
	}
}

func (c *CheckCertJob) execute() {
	result := []CertCheckNotification{}
	for _, item := range c.certList {
		checkStatus, err := services.CheckCertStatus(item, c.warningDays)

		if err != nil {
			log.Printf("Error checking cert status: %s", err)
			continue
		}

		log.Printf("Cert status for %s: %t", item.Name, checkStatus.IsValid)

		item := c.getNotificationModel(checkStatus)
		if c.shouldNotify(item) {
			result = append(result, item)
		}
	}

	err := c.notifier.Notify(result)

	if err != nil {
		log.Printf("Error sending notification: %s", err)
	}
}

func (c *CheckCertJob) shouldNotify(model CertCheckNotification) bool {
	return c.level == Info || !model.IsValid || (c.level == Warning && model.ExpirationWarning)
}

func (c *CheckCertJob) getNotificationModel(certificate *models.CertCheckResult) CertCheckNotification {
	result := CertCheckNotification{
		Hostname:          certificate.Hostname,
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
