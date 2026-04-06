# sharp-cred-manager
This project aims to provide a simple tool to monitor TLS certificate and Azure Key Vault secret validity. It is entirely built using [GO](https://go.dev/).

![Demo frontend image](/docs/demo.jpeg)

Additionally, the app can be configured to run jobs at a given schedule. The jobs will check the configured websites and secrets and send a message to a Webhook with a summary of their validity.

Teams message:

![Demo teams message](/docs/TeamsDemo.jpg)

Slack message:

![Demo slack message](/docs/SlackDemo.jpg)

## Breaking Change
> ⚠️ `AZUREKEYVAULT_N` has been renamed to `AZUREKEYVAULTCERT_N`. Please update your configuration accordingly.

# Getting started
### Running webserver via Docker
> Note: replace docker with podman if needed.

The easiest way to get started is to run the Docker image published to [Docker Hub](https://hub.docker.com/repository/docker/jlucaspains/sharp-cred-manager/general). Replace the `SITE_1` parameter value with a website to monitor. To add other websites, just add parameters `SITE_n` where `n` is an integer.

```bash
docker run -it -p 8000:8000 \
    --env ENV=DEV \
    --env SITE_1=https://expired.badssl.com/ \
    --env AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert \
    --env AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret \
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
    --environment-variables ENV=DEV SITE_1=https://expired.badssl.com/ AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret
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
    --env-vars ENV=DEV SITE_1=https://expired.badssl.com/ AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret \
    --query properties.configuration.ingress.fqdn
```

## Jobs and Webhook Notifications
The app can be configured to run jobs at a given schedule. The jobs will check the configured websites/secrets and send a message to a Webhook. Currently, Teams and Slack are supported.

Adjust the `CHECK_CERT_JOB_SCHEDULE` cron to run certificate checks at the desired schedule, and `CHECK_SECRET_JOB_SCHEDULE` for secret checks.

The `WEBHOOK_URL` is the URL of the Teams/Slack Webhook to send the message to. Generate a webhook URL for Teams following [this guide](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook#add-an-incoming-webhook-to-a-teams-channel) and for Slack following [this guide](https://api.slack.com/messaging/webhooks).

```bash
docker run -it -p 8000:8000 `
    --env ENV=DEV `
    --env SITE_1=https://expired.badssl.com/ `
    --env CHECK_CERT_JOB_SCHEDULE=* * * * * `
    --env CHECK_SECRET_JOB_SCHEDULE=* * * * * `
    --env WEBHOOK_URL=ReplaceWithWebhookUrl `
    --env WEBHOOK_TYPE=teams `
    jlucaspains/sharp-cred-manager
```

## Secret Monitoring
The app can monitor **Azure Key Vault secrets** — checking their expiration date and enabled/active status — alongside TLS certificates. The dashboard shows a **Secrets tab** (alongside the existing Certificates tab) where each secret card displays its name, enabled status, and days until expiry.

### Configuring secrets
Set one or more `AZUREKEYVAULTSECRET_N` environment variables (where N starts at 1) to the URL of the secret to monitor:

- **Specific secret**: `https://mykeyvault.vault.azure.net/secrets/my-secret` — monitors only that secret.
- **Vault-only URL**: `https://mykeyvault.vault.azure.net` — lists and monitors **all** secrets in the vault.

When using a vault-only URL, the following filters apply:
- `SECRET_CHECK_INCLUDE_DISABLED` (default: `false`) — set to `true` to include disabled secrets.
- `SECRET_CHECK_REQUIRE_EXPIRE_DATE` (default: `true`) — set to `false` to also monitor secrets without an expiration date.

A secret is considered **valid** when:
1. Its `enabled` attribute is `true`.
2. Its expiration date (if set) has not passed.
3. Its expiration date is not within `SECRET_WARNING_VALIDITY_DAYS` days (warning state).

## All environment options
| Environment variable              | Description                                                                     | Default value                                 |
|-----------------------------------|---------------------------------------------------------------------------------|-----------------------------------------------|
| ENV                               | Environment name. Used to configure the app to run in different environments.   |                                               |
| SITE_1..SITE_N                    | Websites to monitor.                                                            |                                               |
| AZUREKEYVAULTCERT_1..N            | Azure Key Vault certificate URLs to monitor. Replaces the old `AZUREKEYVAULT_N`.  |                                               |
| AZUREKEYVAULTSECRET_1..N          | Azure Key Vault secret URLs to monitor. Use a vault-only URL to monitor all secrets in a vault. |                                 |
| CHECK_CERT_JOB_SCHEDULE           | Cron schedule to run the job that checks the certificates.                      |                                               |
| CHECK_SECRET_JOB_SCHEDULE         | Cron schedule to run the job that checks Key Vault secrets.                     |                                               |
| WEBHOOK_URL                       | Webhook URL to send the message to.                                             |                                               |
| MESSAGE_URL                       | URL to be used message action                                                   |                                               |
| MESSAGE_TITLE                     | Message  title                                                                  | Sharp Cert Manager Summary                    |
| MESSAGE_BODY                      | Message body body                                                               | The following credentials were checked on %s |
| WEB_HOST_PORT                     | Host and port the web server will listen on                                     | :8000                                         |
| WEBHOOK_TYPE                      | Defines whether teams or slack webhooks are used                                | teams                                         |
| TLS_CERT_FILE                     | Certificate used for TLS hosting                                                |                                               |
| TLS_CERT_KEY_FILE                 | Certificate key used for TLS hosting                                            |                                               |
| CERT_WARNING_VALIDITY_DAYS        | Defines how many days from today a cert need to have to prevent a warning       | 30                                            |
| SECRET_WARNING_VALIDITY_DAYS      | Defines how many days from today a secret needs to have before a warning is raised | 30                                         |
| SECRET_CHECK_INCLUDE_DISABLED     | When using a vault-only URL, include disabled secrets in monitoring             | false                                         |
| SECRET_CHECK_REQUIRE_EXPIRE_DATE  | When using a vault-only URL, only monitor secrets that have an expiration date  | true                                          |
| CHECK_CRED_JOB_NOTIFICATION_LEVEL | Defines minimum notification level for jobs (cert and secret). Values are Info, Warning, or Error | Warning              |
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
- [x] Azure Key Vault certificate integration
- [x] Azure Key Vault secret monitoring

## Headless Mode
The `HEADLESS` environment variable is used to determine if the web server should start. If `HEADLESS` is set to "true", the web server does not start. This can be useful for running the job task only once and exiting with a success code.

To run the job task only once and exit with a success code, set `HEADLESS` to "true" and `CHECK_CERT_JOB_SCHEDULE` to an empty value.

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
CERT_WARNING_VALIDITY_DAYS=90 `
HEADLESS=true `
WEBHOOK_TYPE=teams `
WEBHOOK_URL=<webhook-url> `
MESSAGE_MENTIONS=<user@domain.com>
CHECK_CRED_JOB_NOTIFICATION_LEVEL=Info
```
