# sharp-cred-manager
This project aims to provide a simple tool to monitor TLS certificates and secret validity. For certificate monitoring, both direct website connection and Azure Key Vault are supported. For secret monitoring, Azure Key Vault is supported. It is entirely built using [GO](https://go.dev/).

![Demo frontend image 1](/docs/demo.png)

![Demo frontend image 2](/docs/demo2.png)

![Demo frontend image 3](/docs/demo3.png)

Additionally, the app can be configured to run jobs at a given schedule. The jobs will check the configured websites and secrets and send a message to a Webhook with a summary of their validity.

Teams message:

![Demo teams message](/docs/TeamsDemo.png)

Slack message:

![Demo slack message](/docs/SlackDemo.png)

## V2
V2 is a new major version that introduces:
1. **Azure Key Vault secret monitoring** — monitor secrets' expiration and enabled/active status alongside certificates
2. **Secrets dashboard tab** — the web UI now has a Secrets tab alongside the existing Certificates tab
3. **App registration dashboard tab** - the web UI now has a App Registrations tab alongside the existing certificates tab
3. **Breaking Change**: The app was renamed from Sharp Cert Manager to **Sharp Cred Manager** as it now monitors credentials beyond certificates

### Migrate from v1.x to v2.x

**Renamed/removed environment variables:**

| v1 variable | v2 variable | Notes |
|---|---|---|
| `AZUREKEYVAULT_N` | `AZUREKEYVAULTCERT_N` | Renamed to distinguish certificate URLs from secret URLs |
| `CHECK_CERT_JOB_SCHEDULE` | `CHECK_CRED_JOB_SCHEDULE` | Certs and secrets now run on a single shared schedule |

**New optional environment variables:**

| Variable | Description | Default |
|---|---|---|
| `AZUREKEYVAULTSECRET_1..N` | Azure Key Vault secret URLs to monitor. Use a vault-only URL to monitor all secrets in a vault. | |
| `SECRET_WARNING_VALIDITY_DAYS` | Days before expiry to trigger a warning for secrets | 30 |
| `SECRET_CHECK_INCLUDE_DISABLED` | When using a vault-only URL, include disabled secrets | false |
| `SECRET_CHECK_REQUIRE_EXPIRE_DATE` | When using a vault-only URL, only monitor secrets that have an expiration date | true |
| `APPREGISTRATION_1..N` | Azure App Registrations to monitor. Includes app secrets and app certificates. Use the App Id of the registration. | |
| `APP_REG_WARNING_VALIDITY_DAYS` | Days before expiry to trigger a warning for app registration credentials | 30 |

**Azure Key Vault permissions**

To monitor Key Vault secrets, the Key Vault Reader role or equivalent is required. The Reader role grants access to list the properties of secrets, but not the value. See [DefaultAzureCredential Class](https://learn.microsoft.com/en-us/dotnet/api/azure.identity.defaultazurecredential?view=azure-dotnet) for the list of possible ways to authenticate. It is not required nor recommended to allow sharp-cred-manager to read secret values.

**Azure Graph permissions**

To monitor app registrations, the user running the application (managed identity, service principal, or user in AZ CLI) needs read permission to the app registrations you setup to monitor. This permission can be granted a number of ways including granting `Application.Read.All` to the app registration being used (if any) or the `Directory Reader` Directory role. See [DefaultAzureCredential Class](https://learn.microsoft.com/en-us/dotnet/api/azure.identity.defaultazurecredential?view=azure-dotnet) for the list of possible ways to authenticate

# Getting started

### Running webserver via Docker
The easiest way to get started is to run the Docker image published to [Docker Hub](https://hub.docker.com/repository/docker/jlucaspains/sharp-cred-manager/general). Replace the `SITE_1` parameter value with a website to monitor. To add other websites, just add parameters `SITE_n` where `n` is an integer.

```bash
docker/podman run -it -p 8000:8000 \
    --env ENV=DEV \
    --env SITE_1=https://expired.badssl.com/ \
    --env AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert \
    --env AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret \
    --env APPREGISTRATION_1=<appId> \
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

### Clone the repo
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
    --environment-variables ENV=DEV SITE_1=https://expired.badssl.com/ AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret APPREGISTRATION_1=<appId>
```

### Azure Container App
> Note: While more expensive, an ACA is a better option for production environments as it provides a more robust and scalable environment.

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
    --env-vars ENV=DEV SITE_1=https://expired.badssl.com/ AZUREKEYVAULTCERT_1=https://mykeyvault.vault.azure.net/certificates/my-cert AZUREKEYVAULTSECRET_1=https://mykeyvault.vault.azure.net/secrets/my-secret APPREGISTRATION_1=<appId> \
    --query properties.configuration.ingress.fqdn
```

## Jobs and Webhook Notifications
The app can be configured to run jobs at a given schedule. The jobs will check all configured websites and secrets together and send a **single combined message** to a Webhook. Currently, Teams and Slack are supported.

Set `CHECK_CRED_JOB_SCHEDULE` to a cron expression to run both certificate and secret checks on the same schedule.

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

## App Registration Monitoring
The app can monitor **Azure App Registration credentials** — both client secrets (`passwordCredentials`) and certificates (`keyCredentials`) — checking their expiration dates. The dashboard shows an **App Registrations tab** (alongside Certificates and Secrets) where each app registration card lists all its credentials with their type and days until expiry. Clicking a card opens a detail modal.

### Configuring app registrations
Set one or more `APPREGISTRATION_N` environment variables (where N starts at 1) to `<appId>`:

```
APPREGISTRATION_1=yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy
```

At startup, sharp-cred-manager reads the configured `APPREGISTRATION_N` entries. The attached client secrets and certificates are then fetched from Microsoft Graph on demand when app registration status is checked, and monitored together.

### Required Azure permission
The identity running sharp-cred-manager needs the **`Application.Read.All`** application permission granted in Entra ID (Microsoft Graph) or equivalent for the user running the application. This allows the app to read application properties including `passwordCredentials` and `keyCredentials`.

> Note: `Application.Read.All` is broader than Key Vault permissions. Grant it as an application permission (not delegated) and have it admin-consented in your tenant.

### Validity rules
A credential is considered **valid** when:
1. Its `endDateTime` (if set) has not passed.
2. Its `startDateTime` (if set) is not in the future.
3. Its `endDateTime` is not within `APP_REG_WARNING_VALIDITY_DAYS` days (warning state).

Credentials with no `endDateTime` are treated as valid with no expiration.

The app registration card is shown as **invalid** (red) if any credential is invalid, and as a **warning** if any credential is within the warning threshold.

## All environment options
| Environment variable              | Description                                                                     | Default value                                 |
|-----------------------------------|---------------------------------------------------------------------------------|-----------------------------------------------|
| ENV                               | Environment name. Used to configure the app to run in different environments.   |                                               |
| SITE_1..SITE_N                    | Websites to monitor.                                                            |                                               |
| AZUREKEYVAULTCERT_1..N            | Azure Key Vault certificate URLs to monitor.                                    |                                               |
| AZUREKEYVAULTSECRET_1..N          | Azure Key Vault secret URLs to monitor. Use a vault-only URL to monitor all secrets in a vault. |                                 |
| CHECK_CRED_JOB_SCHEDULE           | Cron schedule to run the job that checks both certificates and secrets together. |                                               |
| WEBHOOK_URL                       | Webhook URL to send the message to.                                             |                                               |
| MESSAGE_URL                       | URL to be used message action                                                   |                                               |
| MESSAGE_TITLE                     | Message  title                                                                  | Sharp Cred Manager Summary                    |
| MESSAGE_BODY                      | Message body                                                                    | The following credentials were checked on %s |
| WEB_HOST_PORT                     | Host and port the web server will listen on                                     | :8000                                         |
| WEBHOOK_TYPE                      | Defines whether teams or slack webhooks are used                                | teams                                         |
| TLS_CERT_FILE                     | Certificate used for TLS hosting                                                |                                               |
| TLS_CERT_KEY_FILE                 | Certificate key used for TLS hosting                                            |                                               |
| CERT_WARNING_VALIDITY_DAYS        | Defines how many days from today a cert need to have to prevent a warning       | 30                                            |
| SECRET_WARNING_VALIDITY_DAYS      | Defines how many days from today a secret needs to have before a warning is raised | 30                                         |
| SECRET_CHECK_INCLUDE_DISABLED     | When using a vault-only URL, include disabled secrets in monitoring             | false                                         |
| SECRET_CHECK_REQUIRE_EXPIRE_DATE  | When using a vault-only URL, only monitor secrets that have an expiration date  | true                                          |
| APPREGISTRATION_1..N              | Azure App Registration app id to monitor. All client secrets and certificates on the registration are monitored. | |
| APP_REG_WARNING_VALIDITY_DAYS     | Defines how many days from today an app registration credential needs to have before a warning is raised | 30              |
| MESSAGE_MENTIONS                  | Comma-separated list of email addresses to mention in webhook notifications.    |                                               |
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
- [x] Azure Key Vault certificate monitoring
- [x] Azure Key Vault secret monitoring
- [x] Azure App Registration credential monitoring (client secrets and certificates)

## Headless Mode
The `HEADLESS` environment variable is used to determine if the web server should start. If `HEADLESS` is set to "true", the web server does not start. This can be useful for running the job task only once and exiting with a success code.

To run the job task only once and exit with a success code, set `HEADLESS` to "true" and leave `CHECK_CRED_JOB_SCHEDULE` unset.

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
