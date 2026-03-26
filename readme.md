# sharp-cred-manager
This project aims to provide a simple tool to monitor certificate validity. It is entirely built using [GO](https://go.dev/).

![Demo frontend image](/docs/demo.jpeg)

Additionally, the app can be configured to run a job at a given schedule. The job will check the configured websites and send a message to a Webhook with a summary of the websites and their certificate validity.

Teams message:

![Demo teams message](/docs/TeamsDemo.jpg)

Slack message:

![Demo slack message](/docs/SlackDemo.jpg)

# Getting started
### Running webserver via Docker
> Note: replace docker with podman if needed.

The easiest way to get started is to run the Docker image published to [Docker Hub](https://hub.docker.com/repository/docker/jlucaspains/sharp-cred-manager/general). Replace the `SITE_1` parameter value with a website to monitor. To add other websites, just add parameters `SITE_n` where `n` is an integer.

```bash
docker run -it -p 8000:8000 \
    --env ENV=DEV \
    --env SITE_1=https://expired.badssl.com/ \
    jlucaspains/sharp-cred-manager
```

### Running CLI
```bash
go install github.com/jlucaspains/sharp-cred-manager/cmd/sharp-cred-manager@latest
sharp-cred-manager check --url https://expired.badssl.com/
```

## Running locally
### Prerequisites
* Go 1.24+
* Tailwindcss CLI

### CLone the repo
```bash
git clone https://github.com/jlucaspains/sharp-cred-manager.git
```

### Install dependencies
```bash
cd sharp-cred-manager
go mod download
```

### Run web server
Generate CSS using Tailwindcss CLI:

```bash
tailwindcss.exe -i ./frontend/styles.css -o ./public/styles.css --minify
```

Create a dev `.env` file:
```bash
echo "ENV=local\nSITE_1=https://expired.badssl.com/" > .env
```

### Run CLI
```bash
go run .\cmd\sharp-cred-manager\ check --url https://expired.badssl.com/
```

## Running in Azure
### Azure Container Instance
Create an ACI resource via Azure CLI. The following parameters may be adjusted
1. `--resource-group`: resource group to be used
2. `--name`: name of the ACI resource
3. `--dns-name-label`: DNS to expose the ACI under
4. `--environment-variables`
   1. `SITE_1..SITE_N`: monitored websites.

```bash
az container create \
    --resource-group rg-sharpcredmanager-001 \
    --name aci-sharpcredmanager-001 \
    --image jlucaspains/sharp-cred-manager \
    --dns-name-label sharp-cred-manager \
    --ports 8000 \
    --environment-variables ENV=DEV SITE_1=https://expired.badssl.com/
```

### Azure Container App
> While more expensive, an ACA is a better option for production environments as it provides a more robust and scalable environment.

First, create an ACA environment using Azure CLI:

```bash
az containerapp env create \
    --name ace-sharpcredmanager-001 \
    --resource-group rg-experiments-soutchcentralus-001
```

Now, create the actual ACA. The following parameters may be adjusted:
1. `-g`: resource group to be used
2. `-n`: name of the app
4. `--env-vars`
   1. `SITE_1..SITE_N`: monitored websites.

```bash
az containerapp create \
    -n aca-sharpcredmanager-001 \
    -g rg-experiments-soutchcentralus-001 \
    --image jlucaspains/sharp-cred-manager \
    --environment ace-sharpcredmanager-001 \
    --ingress external --target-port 8000 \
    --env-vars ENV=DEV SITE_1=https://expired.badssl.com/ \
    --query properties.configuration.ingress.fqdn
```

## Jobs and Webhook Notifications
The app can be configured to run a job at a given schedule. The job will check the configured websites and send a message to a Webhook with a summary of the websites and their certificate validity. Currently, Teams and Slack are supported.

Adjust the `CHECK_CRED_JOB_SCHEDULE` cron to run at the desired schedule.

The `WEBHOOK_URL` is the URL of the Teams/Slack Webhook to send the message to. Generate a webhook URL for Teams following [this guide](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook#add-an-incoming-webhook-to-a-teams-channel) and for Slack following [this guide](https://api.slack.com/messaging/webhooks).

```bash
docker run -it -p 8000:8000 `
    --env ENV=DEV `
    --env SITE_1=https://expired.badssl.com/ `
    --env CHECK_CRED_JOB_SCHEDULE=* * * * * `
    --env WEBHOOK_URL=ReplaceWithWebhookUrl `
    --env WEBHOOK_TYPE=teams `
    jlucaspains/sharp-cred-manager
```

## All environment options
| Environment variable              | Description                                                                     | Default value                                 |
|-----------------------------------|---------------------------------------------------------------------------------|-----------------------------------------------|
| ENV                               | Environment name. Used to configure the app to run in different environments.   |                                               |
| SITE_1..SITE_N                    | Websites to monitor.                                                            |                                               |
| AZUREKEYVAULT_1..AZUREKEYVAULT_N  | Azure key vault certificates URLs to monitor.                                   |                                               |
| CHECK_CRED_JOB_SCHEDULE           | Cron schedule to run the job that checks the certificates.                      |                                               |
| WEBHOOK_URL                       | Webhook URL to send the message to.                                             |                                               |
| MESSAGE_URL                       | URL to be used message action                                                   |                                               |
| MESSAGE_TITLE                     | Message  title                                                                  | Sharp Cert Manager Summary                    |
| MESSAGE_BODY                      | Message body body                                                               | The following certificates were checked on %s |
| WEB_HOST_PORT                     | Host and port the web server will listen on                                     | :8000                                         |
| WEBHOOK_TYPE                      | Defines whether teams or slack webhooks are used                                | teams                                         |
| TLS_CERT_FILE                     | Certificate used for TLS hosting                                                |                                               |
| TLS_CERT_KEY_FILE                 | Certificate key used for TLS hosting                                            |                                               |
| CRED_WARNING_VALIDITY_DAYS        | Defines how many days from today a cert need to have to prevent a warning       | 30                                            |
| CHECK_CRED_JOB_NOTIFICATION_LEVEL | Defines minimum notification level for jobs. Values are Info, Warning, or Error | Warning                                       |
| HEADLESS                          | If set to "true", the web server does not start.                                |                                               |

## Security considerations
This app is intended to run in private environments or at a minimum be behind a secure gateway with proper TLS and authentication to ensure it is not improperly used.

The app will allow unsecured requests to the configured websites. It will perform a get and discard any data returned. All information used is derived from the connection and certificate negotiated between the http client and the web server being monitored.

## Features
Below features are currentl being evaluated and/or built. If you have a suggestion, please create an issue.

- [x] Display list of monitored certificates
- [x] Display certificate details
- [x] Monitor certificate in background
- [x] Teams WebHook integration
- [x] Slack WebHook integration
- [x] Azure Key Vault integration

## Headless Mode
The `HEADLESS` environment variable is used to determine if the web server should start. If `HEADLESS` is set to "true", the web server does not start. This can be useful for running the job task only once and exiting with a success code.

To run the job task only once and exit with a success code, set `HEADLESS` to "true" and `CHECK_CRED_JOB_SCHEDULE` to an empty value.

Example: Running as a container app job using az cli
```bash
az containerapp job create `
    --name sharp-cred-manager `
    --resource-group <resource-group> `
    --image jlucaspains/sharp-cred-manager `
    --trigger-type "Schedule" `
    --replica-timeout 1800 `
    --cpu "0.25" --memory "0.5Gi" `
    --cron-expression "0 8 * * 1" `
    --replica-retry-limit 1 `
    --parallelism 1 `
    --replica-completion-count 1 `
    --env-vars ENV=DEV `
SITE_1=https://blog.lpains.net/ `
CRED_WARNING_VALIDITY_DAYS=90 `
HEADLESS=true `
WEBHOOK_TYPE=teams `
WEBHOOK_URL=<webhook-url> `
MESSAGE_MENTIONS=<user@domain.com>
CHECK_CRED_JOB_NOTIFICATION_LEVEL=Info
```
