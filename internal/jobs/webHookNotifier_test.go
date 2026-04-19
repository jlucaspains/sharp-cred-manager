package jobs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebHookNotifierExplicitInit(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Teams, ts.URL, "title", "body", "url", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"type\": \"message\",\n\"attachments\": [{\n\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\"content\": {\n\"type\": \"AdaptiveCard\",\n\"version\": \"1.5\",\n\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\"msteams\": {\n\"width\": \"full\"\n},\n\"body\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"title\",\n\"size\": \"large\",\n\"weight\": \"bolder\",\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"body\",\n\"isSubtle\": true,\n\"wrap\": true\n},\n{\n\"type\": \"Table\",\n\"columns\": [\n{\n\"width\": 2\n},\n{\n\"width\": 4\n}\n],\n\"rows\": [\n{\n\"type\": \"TableRow\",\n\"cells\": [\n{\n\"type\": \"TableCell\",\n\"items\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"No items to display\"\n}\n]\n}\n]\n}\n]\n}\n],\n\"actions\": [\n{\n\"type\": \"Action.OpenUrl\",\n\"title\": \"View Details\",\n\"url\": \"url\"\n}\n]\n}\n}]\n}", result)
}

func TestWebHookNotifierWithMentionsExplicitInit(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Teams, ts.URL, "title", "body", "url", "me@lpains.net")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"type\": \"message\",\n\"attachments\": [{\n\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\"content\": {\n\"type\": \"AdaptiveCard\",\n\"version\": \"1.5\",\n\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\"msteams\": {\n\"width\": \"full\",\n\"entities\": [\n{\n\"type\": \"mention\",\n\"text\": \"<at>me@lpains.net</at>\",\n\"mentioned\": {\n\"id\": \"me@lpains.net\",\n\"name\": \"me\"\n}\n}\n]\n},\n\"body\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"title\",\n\"size\": \"large\",\n\"weight\": \"bolder\",\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"Attention: <at>me@lpains.net</at>\",\n\"isSubtle\": true,\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"body\",\n\"isSubtle\": true,\n\"wrap\": true\n},\n{\n\"type\": \"Table\",\n\"columns\": [\n{\n\"width\": 2\n},\n{\n\"width\": 4\n}\n],\n\"rows\": [\n{\n\"type\": \"TableRow\",\n\"cells\": [\n{\n\"type\": \"TableCell\",\n\"items\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"No items to display\"\n}\n]\n}\n]\n}\n]\n}\n],\n\"actions\": [\n{\n\"type\": \"Action.OpenUrl\",\n\"title\": \"View Details\",\n\"url\": \"url\"\n}\n]\n}\n}]\n}", result)
}

func TestWebHookNotifierImplicitInit(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Teams, ts.URL, "", "The following credentials were checked on today", "", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"type\": \"message\",\n\"attachments\": [{\n\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\"content\": {\n\"type\": \"AdaptiveCard\",\n\"version\": \"1.5\",\n\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\"msteams\": {\n\"width\": \"full\"\n},\n\"body\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"Sharp Cred Manager Summary\",\n\"size\": \"large\",\n\"weight\": \"bolder\",\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"The following credentials were checked on today\",\n\"isSubtle\": true,\n\"wrap\": true\n},\n{\n\"type\": \"Table\",\n\"columns\": [\n{\n\"width\": 2\n},\n{\n\"width\": 4\n}\n],\n\"rows\": [\n{\n\"type\": \"TableRow\",\n\"cells\": [\n{\n\"type\": \"TableCell\",\n\"items\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"No items to display\"\n}\n]\n}\n]\n}\n]\n}\n]\n}\n}]\n}", result)
}

func TestTeamsWebHookNotifierWithData(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Teams, ts.URL, "", "The following credentials were checked on today", "", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{
		{Label: "Certificates", Items: []CheckNotification{{Name: "host1", IsValid: true}}},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"type\": \"message\",\n\"attachments\": [{\n\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\"content\": {\n\"type\": \"AdaptiveCard\",\n\"version\": \"1.5\",\n\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\"msteams\": {\n\"width\": \"full\"\n},\n\"body\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"Sharp Cred Manager Summary\",\n\"size\": \"large\",\n\"weight\": \"bolder\",\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"The following credentials were checked on today\",\n\"isSubtle\": true,\n\"wrap\": true\n},\n{\n\"type\": \"TextBlock\",\n\"text\": \"**Certificates**\",\n\"weight\": \"bolder\",\n\"separator\": true,\n\"wrap\": true\n},\n{\n\"type\": \"Table\",\n\"columns\": [\n{\n\"width\": 2\n},\n{\n\"width\": 4\n}\n],\n\"rows\": [\n{\n\"type\": \"TableRow\",\n\"cells\": [\n{\n\"type\": \"TableCell\",\n\"items\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"host1\",\n\"wrap\": true\n}\n]\n},\n{\n\"type\": \"TableCell\",\n\"items\": [\n{\n\"type\": \"TextBlock\",\n\"text\": \"✔️\",\n\"wrap\": true\n}\n]\n}\n]\n}\n]\n}\n]\n}\n}]\n}", result)
}

func TestSlackWebHookNotifierWithData(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Slack, ts.URL, "", "The following credentials were checked on today", "", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{
		{Label: "Certificates", Items: []CheckNotification{{Name: "host1", IsValid: true}}},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\",\n\"blocks\": [\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\"\n}\n},\n{\n\"type\": \"divider\"\n},\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \"*Certificates*\"\n}\n},\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \":white_check_mark:\\t*host1*\\n\"\n}\n},\n{\n\"type\": \"divider\"\n}\n]\n}", result)
}

func TestSlackWebHookNotifierWithNotificationUrl(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	var result string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		defer r.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		result = buf.String()
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Slack, ts.URL, "", "The following credentials were checked on today", "https://example.com", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{
		{Label: "Certificates", Items: []CheckNotification{{Name: "host1", IsValid: true}}},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\",\n\"blocks\": [\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\"\n}\n},\n{\n\"type\": \"divider\"\n},\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \"*Certificates*\"\n}\n},\n{\n\"type\": \"section\",\n\"text\": {\n\"type\": \"mrkdwn\",\n\"text\": \":white_check_mark:\\t*host1*\\n\"\n}\n},\n{\n\"type\": \"divider\"\n},\n{\n\"type\": \"actions\",\n\"elements\": [\n{\n\"type\": \"button\",\n\"text\": {\n\"type\": \"plain_text\",\n\"text\": \"View details\",\n\"emoji\": true\n},\n\"value\": \"click_me_123\",\n\"url\": \"https://example.com\"\n}\n]\n}\n]\n}", result)
}

func TestWebHookNotifierBadResponseCode(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	ts.Start()
	defer ts.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	WebHookNotifier := &WebHookNotifier{}
	WebHookNotifier.Init(Teams, ts.URL, "", "", "", "")
	err := WebHookNotifier.Notify([]CheckNotificationGroup{})
	assert.Equal(t, "error sending notification to Teams. Error 400 Bad Request", err.Error())
}
