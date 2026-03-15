# Authentication Setup

## Create a Service Account

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select or create a project
3. Navigate to **IAM & Admin > Service Accounts**
4. Click **Create Service Account**
5. Give it a name (e.g., `gplay-cli`)
6. Click **Create and Continue**
7. Skip granting project access (not needed)
8. Click **Done**
9. Click on the service account you just created
10. Go to the **Keys** tab
11. Click **Add Key > Create new key > JSON**
12. Save the downloaded JSON file securely

## Grant Access in Play Console

1. Go to [Google Play Console](https://play.google.com/console)
2. Navigate to **Users and permissions**
3. Click **Invite new users**
4. Enter the service account email (from the JSON file)
5. Set permissions:
   - For full access: **Admin**
   - For read-only: **View app information and download bulk reports**
   - For releases: **Manage production releases** or **Manage testing track releases**
6. Click **Invite user**
7. Wait for the invitation to be accepted (automatic for service accounts)

## Configure gplay

```bash
# Login with the service account key
gplay auth login --service-account /path/to/service-account.json

# Verify the setup
gplay auth doctor

# Set a default package name
gplay init --package com.example.app
```

## Environment Variables

You can also configure authentication via environment variables:

```bash
export GPLAY_SERVICE_ACCOUNT=/path/to/service-account.json
export GPLAY_PACKAGE=com.example.app
```

## Troubleshooting

- **403 Forbidden**: The service account doesn't have the required permissions in Play Console
- **Invalid credentials**: Check the JSON file path and ensure it's a valid service account key
- **API not enabled**: Enable the Google Play Android Developer API in your Google Cloud project
