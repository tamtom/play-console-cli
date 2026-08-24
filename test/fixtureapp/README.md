# Fixture app for internal-track release tests

This directory holds a minimal Android app for live release testing. The CI
job builds it as an AAB, uploads it with the compiled `gplay` binary, and
publishes it ONLY to the internal track of the fixture app
`com.itdeveapps.stepsshare`. A readback then verifies the release.

## Safety rules

- The package name is fixed. The workflow takes no package input.
- The release goes to the internal track only. The script refuses every
  other track.
- The version code comes from a live readback: `max(existing) + 1`. This
  keeps the version sequence of the app intact.
- The workflow concurrency group blocks parallel live runs.

## Required GitHub secrets

The job stays inert until these secrets exist:

| Secret | Content |
|--------|---------|
| `GPLAY_UPLOAD_KEYSTORE_B64` | The upload keystore, base64-encoded |
| `GPLAY_UPLOAD_KEYSTORE_PASSWORD` | The keystore password |
| `GPLAY_UPLOAD_KEY_ALIAS` | The key alias |
| `GPLAY_UPLOAD_KEY_PASSWORD` | The key password |

Create the base64 value with:

```bash
base64 -i upload.keystore | pbcopy
```

The keystore must hold the upload key that Play App Signing expects for
`com.itdeveapps.stepsshare`.

## Local build

```bash
cd test/fixtureapp
gradle bundleRelease -PfixtureVersionCode=123
```
