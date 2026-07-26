# CI/CD Integration

`gplay` is a single static binary with JSON output and no interactive prompts — it drops into any CI/CD pipeline: GitHub Actions, GitLab CI, Jenkins, Bitrise, CircleCI.

## GitHub Actions

```yaml
name: Deploy to Play Store

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up gplay
        run: |
          curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Build app
        run: ./gradlew bundleRelease

      - name: Deploy to internal track
        env:
          GPLAY_SERVICE_ACCOUNT: ${{ secrets.PLAY_SERVICE_ACCOUNT }}
        run: |
          gplay release \
            --package com.example.app \
            --track internal \
            --bundle app/build/outputs/bundle/release/app-release.aab
```

## GitLab CI

```yaml
deploy:
  stage: deploy
  image: ubuntu:latest
  before_script:
    - curl -fsSL https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.sh | bash
    - export PATH="$HOME/.local/bin:$PATH"
  script:
    - gplay release --package $PACKAGE_NAME --track internal --bundle app.aab
  variables:
    GPLAY_SERVICE_ACCOUNT: $PLAY_SERVICE_ACCOUNT
```

## Useful CI gates

```bash
# Fail the build on compliance issues before uploading (offline, no credentials needed)
gplay preflight --file app.aab --max-size 100M --fail-on warning

# Block only on hard blockers, and check the store listing in the same pass
gplay preflight --file app.aab --listings-dir ./fastlane/metadata/android --fail-on error

# Narrow the gate to specific scanners (see `gplay preflight --list-scanners`)
gplay preflight --file app.aab --only manifest,permissions,native_libs,secrets --fail-on warning

# Fail on bundle size regression
gplay bundles compare --base old.aab --candidate new.aab --threshold 2M

# Google Checks privacy/policy gate — exits non-zero on high-priority failures
gplay checks analyze --account <account-id> --app <app-id> --binary app.aab \
  --code-ref "$GIT_SHA" --severity-threshold PRIORITY --checks-filter "state = FAILED"

# Notify the team after release
gplay notify send --webhook-url $SLACK_WEBHOOK --message "v1.2.3 → internal" --format slack
```

## Security in CI

- Store the service account key in your CI secrets manager, never in the repo
- Point `GPLAY_SERVICE_ACCOUNT` at the key file materialized at runtime
- Use separate service accounts per environment with least-privilege Play Console permissions
- Rotate keys regularly
