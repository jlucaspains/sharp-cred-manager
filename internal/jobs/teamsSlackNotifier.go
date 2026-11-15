package jobs

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"
)

type TeamsSlackNotifier struct {
	NotifierType      NotifierType
	WebhookUrl        string
	NotificationTitle string
	NotificationBody  string
	NotificationUrl   string
	Mentions          []string
	parsedTemplate    *template.Template
	httpClient        *http.Client
	retryPolicy       retryPolicy
}

type TeamsSlackNotificationCard struct {
	Title           string
	Description     string
	NotificationUrl string
	Groups          []CheckNotificationGroup
	Mentions        []string
}

func (m *TeamsSlackNotifier) Init(notifierType NotifierType, webhookUrl string, notificationTitle string, notificationBody string, notificationUrl string, messageMentions string) {
	if notificationTitle == "" {
		notificationTitle = "Sharp Cred Manager Summary"
	}

	if notificationBody == "" {
		notificationBody = fmt.Sprintf("The following credentials were checked on %s", time.Now().Format("01/02/2006"))
	}

	m.NotifierType = notifierType
	m.NotificationTitle = notificationTitle
	m.NotificationBody = notificationBody
	m.NotificationUrl = notificationUrl
	m.WebhookUrl = webhookUrl
	m.Mentions = parseMentions(messageMentions)
}

func parseMentions(mentions string) []string {
	if mentions == "" {
		return []string{}
	}

	return strings.Split(mentions, ",")
}

func (m *TeamsSlackNotifier) Notify(groups []CheckNotificationGroup) error {
	if m.retryPolicy.maxAttempts == 0 {
		m.retryPolicy = defaultRetryPolicy
	}
	client := m.getClient()
	parsedTemplate := m.getTemplate()
	card := TeamsSlackNotificationCard{
		Title:           m.NotificationTitle,
		Description:     m.NotificationBody,
		NotificationUrl: m.NotificationUrl,
		Groups:          groups,
		Mentions:        m.Mentions,
	}

	var templateBody bytes.Buffer
	err := parsedTemplate.Execute(&templateBody, card)

	if err != nil {
		return err
	}

	body := templateBody.Bytes()
	fmt.Println(string(body))

	newRequest := func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, m.WebhookUrl, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	return postWithRetry(client, newRequest, m.NotifierType.String(), m.retryPolicy)
}

func (m *TeamsSlackNotifier) getTemplate() *template.Template {
	if m.parsedTemplate == nil {
		m.parsedTemplate, _ = template.New("template").Funcs(template.FuncMap{
			"split": func(s, sep string) []string {
				return strings.Split(s, sep)
			},
			"replace": func(input, from, to string) string {
				return strings.ReplaceAll(input, from, to)
			},
		}).Parse(NotificationTemplates[m.NotifierType])
	}

	return m.parsedTemplate
}

func (m *TeamsSlackNotifier) getClient() *http.Client {
	if m.httpClient == nil {
		m.httpClient = &http.Client{Timeout: webhookClientTimeout}
	}

	return m.httpClient
}

func (m *TeamsSlackNotifier) IsReady() bool {
	return m.WebhookUrl != ""
}
