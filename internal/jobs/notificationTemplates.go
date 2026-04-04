package jobs

const slackMessageTemplate = `{
	"blocks": [
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "{{ .Title }}\n{{ .Description }}"
			}
		},
		{
			"type": "divider"
		},
		{{- if gt (len .Items) 0}}
		{{- $max := len (slice .Items 1)}}
		{{- range $i, $item := .Items}}
		{
			{{- $itemName := replace ($item.Name) "/" " / "}}
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "{{if not $item.IsValid}}:x:{{else if $item.ExpirationWarning}}:warning:{{else}}:white_check_mark:{{end}}\t*{{$itemName}}*\n{{ range $index, $element := $item.Messages}}{{if $index}}, {{end}}{{$element}}{{end}}"
			}
		},
		{{- end}}
		{{- else}}
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "No items to display"
			}
		},
		{{- end}}
		{
			"type": "divider"
		},
		{{- if gt (len .NotificationUrl) 0}}
		{
			"type": "actions",
			"elements": [
				{
					"type": "button",
					"text": {
						"type": "plain_text",
						"text": "View details",
						"emoji": true
					},
					"value": "click_me_123",
					"url": "{{ .NotificationUrl }}"
				}
			]
		}
		{{- end}}
	]
}`

const teamsMessageTemplate = `{
	"type": "message",
	"attachments": [{
		"contentType": "application/vnd.microsoft.card.adaptive",
		"content": {
			"type": "AdaptiveCard",
			"version": "1.5",
			"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
			"body": [
				{
					"type": "TextBlock",
					"text": "{{ .Title }}",
					"size": "large",
					"weight": "bolder",
					"wrap": true
				},{{- if gt (len .Mentions) 0}}
				{
					"type": "TextBlock",
					"text": "Attention: {{ range $index, $element := .Mentions}}{{if $index}}, {{end}}<at>{{$element}}</at>{{end}}",
					"isSubtle": true,
					"wrap": true
				},{{- end}}
				{
					"type": "TextBlock",
					"text": "{{ .Description }}",
					"isSubtle": true,
					"wrap": true
				},
				{
					"type": "Table",
					"columns": [
						{
							"width": 2
						},
						{
							"width": 4
						}
					],
					"rows": [
						{{- if gt (len .Items) 0}}
						{{- $max := len (slice .Items 1)}}
						{{- range $i, $item := .Items}}
						{
							"type": "TableRow",
							"cells": [
								{
									"type": "TableCell",
									"items": [
										{
										"type": "TextBlock",
										"text": "{{if not $item.IsValid}}❌{{else if $item.ExpirationWarning}}⚠️{{else}}✔️{{end}}{{$item.Name}}"
										}
									]
								},
								{
									"type": "TableCell",
									"items": [
										{
										"type": "TextBlock",
										"text": "{{ range $index, $element := $item.Messages}}{{if $index}}, {{end}}{{$element}}{{end}}",
										"wrap": true
										}
									]
								}
							]
						}{{if lt $i $max}},{{end}}
						{{- end}}
						{{- else}}
						{
							"type": "TableRow",
							"cells": [
								{
									"type": "TableCell",
									"items": [
										{
										"type": "TextBlock",
										"text": "No items to display"
										}
									]
								}
							]
						}
						{{- end}}
					]
				}
			]{{if .NotificationUrl}},
			"actions": [
				{
					"type": "Action.OpenUrl",
					"title": "View Details",
					"url": "{{ .NotificationUrl }}"
				}
			]{{end}}{{if gt (len .Mentions) 0}},
			"msteams": {
                "entities": [
					{{- $max := len (slice .Mentions 1)}}
					{{- range $i, $item := .Mentions}}
					{{- $shortNames := split $item "@"}}
                    {
						"type": "mention",
						"text": "<at>{{$item}}</at>",
						"mentioned": {
							"id": "{{$item}}",
							"name": "{{ index $shortNames 0 }}"
						}
					}{{if lt $i $max}},{{end}}
					{{- end}}
				]
			}
			{{- end}}
		}
	}]
}`

type NotifierType int

const (
	Teams NotifierType = iota
	Slack
)

func (n NotifierType) String() string {
	return [...]string{"Teams", "Slack"}[n]
}

var Notifiers = map[string]NotifierType{
	"teams": Teams,
	"slack": Slack,
}

var NotificationTemplates = map[NotifierType]string{
	Teams: teamsMessageTemplate,
	Slack: slackMessageTemplate,
}
