# Authentication

`gplay` authenticates to the Google Play Android Developer API with a **service account**. This guide covers the automated setup, the manual setup, profiles, and troubleshooting.

## Automated setup (recommended)

```bash
gplay setup --auto
```

That's it. This drives the whole flow via `gcloud`:
- Installs `gcloud` if needed (Homebrew on macOS, curl on Linux)
- Logs you into Google Cloud (`gcloud auth login`) if you aren't already
- Enables the `androidpublisher` API
- Creates a service account and downloads its JSON key
- Wires the profile into `~/.gplay/config.json`
- Opens Play Console for the one manual step (granting the account access)

`gplay auth setup --auto` is an alias of the same command.

Useful flags:

```bash
gplay setup --auto --project <id>     # if gcloud has no default project
gplay setup --auto --dry-run          # preview the gcloud commands, run nothing
gplay setup --auto --no-browser       # CI/agent friendly: never open a browser
gplay setup --auto --sa-name my-ci    # custom service-account name
gplay setup --auto --output json      # machine-readable result

# Verify
gplay auth doctor
```

`--no-browser` requires that `gcloud` is already installed and authenticated (it won't launch the interactive login), which makes it the right choice for CI and headless agents.

## Manual setup

### 1. Create a Google Cloud project & enable the API

```
Google Cloud Console → Create Project → APIs & Services → Library
→ Search "Google Play Android Developer API" → Enable
```

### 2. Create a service account & download a key

```
IAM & Admin → Service Accounts → Create Service Account
→ Name it (e.g., "gplay-cli") → Create → Done
→ Click the account → Keys → Add Key → Create new key → JSON
→ Save the JSON file securely (never commit to git!)
```

### 3. Grant access in Play Console

```
Play Console → Users and permissions → Invite new users
→ Paste service account email (from JSON: "client_email" field)
→ Set permissions (Admin, or per-app access)
→ Invite user
```

### 4. Configure gplay

```bash
# Option A: Login command (saves to profile)
gplay auth login --service-account /path/to/service-account.json

# Option B: Environment variable
export GPLAY_SERVICE_ACCOUNT=/path/to/service-account.json

# Verify setup
gplay auth doctor
```

## Profiles

Use profiles to switch between multiple developer accounts or apps:

```bash
# Add profiles for different accounts/apps
gplay auth login --profile work --service-account /path/to/work-sa.json
gplay auth login --profile personal --service-account /path/to/personal-sa.json

# Switch default profile
gplay auth switch --profile work

# Check current status
gplay auth status

# Use a specific profile for a single command
GPLAY_PROFILE=personal gplay tracks list --package com.example.app
```

## Diagnostics

```bash
# Validate auth setup (16 checks: gcloud, config, SA key, DNS, disk, clock, ...)
gplay auth doctor

# Auto-fix issues (with confirmation)
gplay auth doctor --fix --confirm
```

## Web sessions (browser cookies)

A few features — app creation, for example — are not covered by the official
Android Publisher API. For those, gplay can drive Play Console's internal web
RPCs using cookies captured from a signed-in browser. This is a separate auth
mechanism from the service-account profiles above; most users never need it.

```bash
# On macOS, import the signed-in Google Chrome session automatically.
# --email selects the matching Google account when Chrome has several.
gplay web auth login --email you@example.com

# Manual fallback: copy the "Cookie:" header of a play.google.com request.
gplay web auth login --email you@example.com --cookies "SID=...; SAPISID=...; ..."

# Or pipe a cookie file / exporter JSON via stdin (keeps cookies out of shell history):
pbpaste | gplay web auth login --email you@example.com --cookies-file -

# List sessions and validate them against the real console:
gplay web auth status --check

# Delete a session (or all of them):
gplay web auth logout --account you@example.com --confirm
gplay web auth logout --all --confirm
```

Automatic import reads Chrome's current cookie database and asks macOS
Keychain for `Chrome Safe Storage` access. Sign in at
<https://play.google.com/console> first. Other platforms can use either manual
cookie option.

On macOS, sessions are stored in the macOS Keychain (service `gplay web
session`) — cookies never touch disk; `~/.gplay/web/` holds only the
`last.json`/`index.json` metadata files (0600 files, 0700 dir). On other
platforms, or when `GPLAY_WEB_SESSION_DIR` is set, sessions are stored as
per-account files under `~/.gplay/web/` instead; the variable is also the
escape hatch when the Keychain is locked or unavailable (headless CI).
Sessions are validated against the real console at login. Cookies are
secrets — treat them like passwords, never commit them, and expect them to
expire; re-run `web auth login` when commands start failing with auth errors.

## Security best practices

- **Never commit service account keys** to version control
- **Never commit web session cookies** — on macOS they live in the Keychain, but with file storage (`GPLAY_WEB_SESSION_DIR` or other platforms) they sit under `~/.gplay/web/`; either way they grant full console access
- **Use environment variables** or secrets management in CI/CD
- **Limit service account permissions** to only what's needed
- **Rotate keys regularly**
- **Use separate service accounts** for different environments

Credentials are stored in config as a file path reference only — never the key content itself.
