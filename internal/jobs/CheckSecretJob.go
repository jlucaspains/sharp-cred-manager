package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/adhocore/gronx"
	"github.com/jlucaspains/sharp-cred-manager/internal/models"
	"github.com/jlucaspains/sharp-cred-manager/internal/services"
)

type CheckSecretJob struct {
	cron        string
	ticker      *time.Ticker
	gron        *gronx.Gronx
	secretList  []models.CheckSecretItem
	running     bool
	notifier    Notifier
	level       Level
	warningDays int
}

func (c *CheckSecretJob) Init(schedule string, level string, warningDays int, secretList []models.CheckSecretItem, notifier Notifier) error {
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
	c.secretList = secretList
	c.ticker = time.NewTicker(time.Minute)
	c.notifier = notifier
	c.level = levelValue
	c.warningDays = warningDays

	return nil
}

func (c *CheckSecretJob) RunNow() {
	c.execute()
}

func (c *CheckSecretJob) Start() {
	c.running = true
	go func() {
		for range c.ticker.C {
			c.tryExecute()
		}
	}()
}

func (c *CheckSecretJob) Stop() {
	c.running = false

	if c.ticker != nil {
		c.ticker.Stop()
	}
}

func (c *CheckSecretJob) tryExecute() {
	due, _ := c.gron.IsDue(c.cron, time.Now().Truncate(time.Minute))

	log.Printf("tryExecute secret job, isDue: %t", due)

	if due {
		c.execute()
	}
}

func (c *CheckSecretJob) execute() {
	result := []CheckNotification{}
	for _, item := range c.secretList {
		checkStatus, err := services.CheckSecretStatus(item, c.warningDays)

		if err != nil {
			log.Printf("Error checking secret status: %s", err)
			continue
		}

		log.Printf("Secret status for %s: %t", item.Name, checkStatus.IsValid)

		notification := c.getNotificationModel(checkStatus)
		if c.shouldNotify(notification) {
			result = append(result, notification)
		}
	}

	err := c.notifier.Notify(result)

	if err != nil {
		log.Printf("Error sending notification: %s", err)
	}
}

func (c *CheckSecretJob) shouldNotify(model CheckNotification) bool {
	return c.level == Info || !model.IsValid || (c.level == Warning && model.ExpirationWarning)
}

func (c *CheckSecretJob) getNotificationModel(secret *models.SecretCheckResult) CheckNotification {
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
