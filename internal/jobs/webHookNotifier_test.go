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
	err := WebHookNotifier.Notify([]CheckNotification{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"type\": \"message\",\n\t\"attachments\": [{\n\t\t\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\t\t\"content\": {\n\t\t\t\"type\": \"AdaptiveCard\",\n\t\t\t\"version\": \"1.5\",\n\t\t\t\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\t\t\t\"body\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"title\",\n\t\t\t\t\t\"size\": \"large\",\n\t\t\t\t\t\"weight\": \"bolder\",\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"body\",\n\t\t\t\t\t\"isSubtle\": true,\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Table\",\n\t\t\t\t\t\"columns\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 2\n\t\t\t\t\t\t},\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 4\n\t\t\t\t\t\t}\n\t\t\t\t\t],\n\t\t\t\t\t\"rows\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"type\": \"TableRow\",\n\t\t\t\t\t\t\t\"cells\": [\n\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\"type\": \"TableCell\",\n\t\t\t\t\t\t\t\t\t\"items\": [\n\t\t\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\t\t\t\t\t\"text\": \"No items to display\"\n\t\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t\t]\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t]\n\t\t\t\t\t\t}\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"actions\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Action.OpenUrl\",\n\t\t\t\t\t\"title\": \"View Details\",\n\t\t\t\t\t\"url\": \"url\"\n\t\t\t\t}\n\t\t\t]\n\t\t}\n\t}]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"type\": \"message\",\n\t\"attachments\": [{\n\t\t\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\t\t\"content\": {\n\t\t\t\"type\": \"AdaptiveCard\",\n\t\t\t\"version\": \"1.5\",\n\t\t\t\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\t\t\t\"body\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"title\",\n\t\t\t\t\t\"size\": \"large\",\n\t\t\t\t\t\"weight\": \"bolder\",\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"Attention: <at>me@lpains.net</at>\",\n\t\t\t\t\t\"isSubtle\": true,\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"body\",\n\t\t\t\t\t\"isSubtle\": true,\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Table\",\n\t\t\t\t\t\"columns\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 2\n\t\t\t\t\t\t},\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 4\n\t\t\t\t\t\t}\n\t\t\t\t\t],\n\t\t\t\t\t\"rows\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"type\": \"TableRow\",\n\t\t\t\t\t\t\t\"cells\": [\n\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\"type\": \"TableCell\",\n\t\t\t\t\t\t\t\t\t\"items\": [\n\t\t\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\t\t\t\t\t\"text\": \"No items to display\"\n\t\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t\t]\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t]\n\t\t\t\t\t\t}\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"actions\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Action.OpenUrl\",\n\t\t\t\t\t\"title\": \"View Details\",\n\t\t\t\t\t\"url\": \"url\"\n\t\t\t\t}\n\t\t\t],\n\t\t\t\"msteams\": {\n                \"entities\": [\n                    {\n\t\t\t\t\t\t\"type\": \"mention\",\n\t\t\t\t\t\t\"text\": \"<at>me@lpains.net</at>\",\n\t\t\t\t\t\t\"mentioned\": {\n\t\t\t\t\t\t\t\"id\": \"me@lpains.net\",\n\t\t\t\t\t\t\t\"name\": \"me\"\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t]\n\t\t\t}\n\t\t}\n\t}]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"type\": \"message\",\n\t\"attachments\": [{\n\t\t\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\t\t\"content\": {\n\t\t\t\"type\": \"AdaptiveCard\",\n\t\t\t\"version\": \"1.5\",\n\t\t\t\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\t\t\t\"body\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"Sharp Cred Manager Summary\",\n\t\t\t\t\t\"size\": \"large\",\n\t\t\t\t\t\"weight\": \"bolder\",\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"The following credentials were checked on today\",\n\t\t\t\t\t\"isSubtle\": true,\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Table\",\n\t\t\t\t\t\"columns\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 2\n\t\t\t\t\t\t},\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 4\n\t\t\t\t\t\t}\n\t\t\t\t\t],\n\t\t\t\t\t\"rows\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"type\": \"TableRow\",\n\t\t\t\t\t\t\t\"cells\": [\n\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\"type\": \"TableCell\",\n\t\t\t\t\t\t\t\t\t\"items\": [\n\t\t\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\t\t\t\t\t\"text\": \"No items to display\"\n\t\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t\t]\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t]\n\t\t\t\t\t\t}\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t]\n\t\t}\n\t}]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{
		{Name: "host1", IsValid: true},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"type\": \"message\",\n\t\"attachments\": [{\n\t\t\"contentType\": \"application/vnd.microsoft.card.adaptive\",\n\t\t\"content\": {\n\t\t\t\"type\": \"AdaptiveCard\",\n\t\t\t\"version\": \"1.5\",\n\t\t\t\"$schema\": \"http://adaptivecards.io/schemas/adaptive-card.json\",\n\t\t\t\"body\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"Sharp Cred Manager Summary\",\n\t\t\t\t\t\"size\": \"large\",\n\t\t\t\t\t\"weight\": \"bolder\",\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\"text\": \"The following credentials were checked on today\",\n\t\t\t\t\t\"isSubtle\": true,\n\t\t\t\t\t\"wrap\": true\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"Table\",\n\t\t\t\t\t\"columns\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 2\n\t\t\t\t\t\t},\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"width\": 4\n\t\t\t\t\t\t}\n\t\t\t\t\t],\n\t\t\t\t\t\"rows\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"type\": \"TableRow\",\n\t\t\t\t\t\t\t\"cells\": [\n\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\"type\": \"TableCell\",\n\t\t\t\t\t\t\t\t\t\"items\": [\n\t\t\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\t\t\t\t\t\"text\": \"✔️host1\"\n\t\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t\t]\n\t\t\t\t\t\t\t\t},\n\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\"type\": \"TableCell\",\n\t\t\t\t\t\t\t\t\t\"items\": [\n\t\t\t\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\t\t\"type\": \"TextBlock\",\n\t\t\t\t\t\t\t\t\t\t\"text\": \"\",\n\t\t\t\t\t\t\t\t\t\t\"wrap\": true\n\t\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t\t]\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t]\n\t\t\t\t\t\t}\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t]\n\t\t}\n\t}]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{
		{Name: "host1", IsValid: true},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"blocks\": [\n\t\t{\n\t\t\t\"type\": \"section\",\n\t\t\t\"text\": {\n\t\t\t\t\"type\": \"mrkdwn\",\n\t\t\t\t\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\"\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"type\": \"divider\"\n\t\t},\n\t\t{\n\t\t\t\"type\": \"section\",\n\t\t\t\"text\": {\n\t\t\t\t\"type\": \"mrkdwn\",\n\t\t\t\t\"text\": \":white_check_mark:\\t*host1*\\n\"\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"type\": \"divider\"\n\t\t},\n\t]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{
		{Name: "host1", IsValid: true},
	})
	assert.Nil(t, err)
	assert.Equal(t, "{\n\t\"blocks\": [\n\t\t{\n\t\t\t\"type\": \"section\",\n\t\t\t\"text\": {\n\t\t\t\t\"type\": \"mrkdwn\",\n\t\t\t\t\"text\": \"Sharp Cred Manager Summary\\nThe following credentials were checked on today\"\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"type\": \"divider\"\n\t\t},\n\t\t{\n\t\t\t\"type\": \"section\",\n\t\t\t\"text\": {\n\t\t\t\t\"type\": \"mrkdwn\",\n\t\t\t\t\"text\": \":white_check_mark:\\t*host1*\\n\"\n\t\t\t}\n\t\t},\n\t\t{\n\t\t\t\"type\": \"divider\"\n\t\t},\n\t\t{\n\t\t\t\"type\": \"actions\",\n\t\t\t\"elements\": [\n\t\t\t\t{\n\t\t\t\t\t\"type\": \"button\",\n\t\t\t\t\t\"text\": {\n\t\t\t\t\t\t\"type\": \"plain_text\",\n\t\t\t\t\t\t\"text\": \"View details\",\n\t\t\t\t\t\t\"emoji\": true\n\t\t\t\t\t},\n\t\t\t\t\t\"value\": \"click_me_123\",\n\t\t\t\t\t\"url\": \"https://example.com\"\n\t\t\t\t}\n\t\t\t]\n\t\t}\n\t]\n}", result)
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
	err := WebHookNotifier.Notify([]CheckNotification{})
	assert.Equal(t, "error sending notification to Teams. Error 400 Bad Request", err.Error())
}
