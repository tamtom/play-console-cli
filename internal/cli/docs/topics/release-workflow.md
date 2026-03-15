# Release Workflow

## Step-by-Step Release Process

### 1. Prepare Your Build

Build your Android App Bundle (.aab) file using your build system:

```bash
./gradlew bundleRelease
```

### 2. Validate Before Upload

```bash
# Validate the bundle
gplay validate bundle --file app-release.aab

# Validate store listing metadata
gplay validate listing --dir ./metadata
```

### 3. Upload and Release

```bash
# Create a release on the internal track
gplay release --package com.example.app \
  --track internal \
  --bundle app-release.aab \
  --release-notes "Bug fixes and performance improvements"

# Or with JSON release notes for multiple locales
gplay release --package com.example.app \
  --track internal \
  --bundle app-release.aab \
  --release-notes '[{"language":"en-US","text":"Bug fixes"},{"language":"ja-JP","text":"..."}]'
```

### 4. Promote Through Tracks

```bash
# Promote from internal to alpha
gplay promote --package com.example.app --from internal --to alpha

# Promote from alpha to beta with staged rollout
gplay promote --package com.example.app --from alpha --to beta --rollout 0.1

# Promote to production with 10% rollout
gplay promote --package com.example.app --from beta --to production --rollout 0.1
```

### 5. Manage Rollout

```bash
# Increase rollout to 50%
gplay rollout update --package com.example.app --track production --rollout 0.5

# Complete the rollout (100%)
gplay rollout complete --package com.example.app --track production

# Halt rollout if issues are found
gplay rollout halt --package com.example.app --track production
```

### 6. Monitor

```bash
# Check crash reports
gplay vitals crashes --package com.example.app

# Check ANR rates
gplay vitals errors --package com.example.app

# Check performance metrics
gplay vitals performance --package com.example.app
```

## CI/CD Integration

For CI/CD pipelines, use environment variables and the `--output json` flag:

```bash
export GPLAY_SERVICE_ACCOUNT=/path/to/key.json
export GPLAY_PACKAGE=com.example.app

gplay release \
  --track internal \
  --bundle app-release.aab \
  --release-notes "Build $BUILD_NUMBER" \
  --output json
```
