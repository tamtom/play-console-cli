# Troubleshooting

## Authentication Errors

### "403 Forbidden" or "Permission denied"

**Cause**: The service account lacks required permissions.

**Fix**:
1. Go to Play Console > Users and permissions
2. Find your service account email
3. Ensure it has the necessary permissions for the operation
4. Wait a few minutes for permission changes to propagate

```bash
gplay auth doctor --fix --confirm
```

### "Invalid credentials" or "Could not load credentials"

**Cause**: The service account JSON file is missing or invalid.

**Fix**:
```bash
# Check current auth status
gplay auth status

# Re-login with the correct file
gplay auth login --service-account /path/to/key.json
```

### "API not enabled"

**Cause**: The Google Play Android Developer API is not enabled.

**Fix**:
1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/library)
2. Search for "Google Play Android Developer API"
3. Click **Enable**

## Release Errors

### "Changes cannot be sent for review"

**Cause**: There are pending changes that need to be reviewed first.

**Fix**: Use `--changes-not-sent-for-review` flag or resolve pending changes in Play Console.

### "Version code already exists"

**Cause**: A bundle with the same version code has already been uploaded.

**Fix**: Increment the `versionCode` in your app's build configuration and rebuild.

### "Edit has been deleted"

**Cause**: The edit session expired or was deleted.

**Fix**: The CLI creates and commits edits automatically. If using manual edits, ensure you commit within the timeout period.

## Network Errors

### "Timeout" errors

**Cause**: The request took too long.

**Fix**:
```bash
# Increase timeout
export GPLAY_TIMEOUT=120s

# For uploads, increase upload timeout
export GPLAY_UPLOAD_TIMEOUT=10m
```

### "Connection refused" or DNS errors

**Cause**: Network connectivity issues.

**Fix**: Check your network connection and proxy settings.

## Common Issues

### Output is not what I expected

Use `--output table` for human-readable output or `--output json --pretty` for formatted JSON:

```bash
gplay tracks list --package com.example.app --output table
gplay tracks list --package com.example.app --output json --pretty
```

### Debug logging

Enable debug mode to see HTTP requests and responses:

```bash
export GPLAY_DEBUG=api
gplay tracks list --package com.example.app
```

### Config not loading

Check config file locations:
- Global: `~/.gplay/config.json`
- Local: `./.gplay/config.json` (takes precedence)

```bash
gplay auth status
```
