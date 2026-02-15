# Google Play Android Publisher API Notes

## API Reference

- **REST Reference**: https://developers.google.com/android-publisher/api-ref/rest
- **Overview**: https://developers.google.com/android-publisher

## Edit Lifecycle

All metadata changes go through an edit workflow:

1. **Create edit** — `edits.insert()` returns an edit ID
2. **Make changes** — listings, tracks, images, etc. within the edit
3. **Validate** — `edits.validate()` checks for errors
4. **Commit** — `edits.commit()` applies changes atomically
5. **Delete** — `edits.delete()` discards uncommitted changes

Edits expire after a period of inactivity. Always commit or delete when done.

## Common Patterns

- Package name (applicationId) is required for almost every API call
- Track names: `internal`, `alpha`, `beta`, `production`, and custom tracks
- Image types: `icon`, `featureGraphic`, `phoneScreenshots`, `sevenInchScreenshots`, `tenInchScreenshots`, `tvScreenshots`, `wearScreenshots`
- Release status values: `draft`, `inProgress`, `halted`, `completed`

## Rate Limits

The API uses quota-based rate limiting. Default quotas are generous for typical usage. Use exponential backoff on 429 responses.

## Common Error Codes

| Code | Meaning |
|------|---------|
| 400 | Invalid request (check field values) |
| 401 | Authentication failed (check service account) |
| 403 | Insufficient permissions |
| 404 | Resource not found (check package name, edit ID) |
| 409 | Conflict (concurrent edit, stale data) |
| 429 | Rate limited (back off and retry) |
