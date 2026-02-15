# Google Play Android Developer API — Quirks & Notes

This document captures observed behaviors, gotchas, and implementation-relevant
details of the [Google Play Android Developer API (v3)](https://developers.google.com/android-publisher/api-ref/rest).
It serves as institutional knowledge for contributors and helps AI agents reason
about API constraints when implementing new features.

> **Last verified**: February 2026 against API v3.

---

## Table of Contents

1. [Edit Lifecycle](#edit-lifecycle)
2. [Image Types and Dimensions](#image-types-and-dimensions)
3. [Track Status Transitions](#track-status-transitions)
4. [Rate Limits and Quota](#rate-limits-and-quota)
5. [Error Response Formats](#error-response-formats)
6. [Pagination Behavior](#pagination-behavior)
7. [Upload Behavior](#upload-behavior)

---

## Edit Lifecycle

**Reference**: [Edits API](https://developers.google.com/android-publisher/api-ref/rest/v3/edits)

The Play Developer API uses an **edit-based transactional model**. Every
modification to an app's listing, tracks, or store assets must happen inside an
edit session.

### Flow

```
edits.insert  →  modify resources  →  edits.validate  →  edits.commit
                                                            │
                                                    (or edits.delete)
```

1. **`edits.insert`** — Creates a new edit and returns an `editId`. All
   subsequent calls reference this ID.
2. **Modify** — Call any number of mutation endpoints (tracks, listings, images,
   etc.) using the `editId`.
3. **`edits.validate`** — Dry-run validation. Returns errors without committing.
   Always call this before commit to surface problems early.
4. **`edits.commit`** — Atomically applies all changes. After commit the
   `editId` is no longer valid.
5. **`edits.delete`** — Discards an uncommitted edit, releasing the lock.

### Quirks

| Behavior | Detail |
|---|---|
| **One active edit per app** | Creating a second edit while one is already open returns `409 Conflict`. You must commit or delete the existing edit first. |
| **Edit expiry** | Uncommitted edits expire after approximately **2 hours** of inactivity. The API does not return the exact TTL. |
| **Stale edit** | Using an expired or committed `editId` returns `404 Not Found`. Handle this by creating a fresh edit and retrying. |
| **Commit is atomic** | Either all changes in the edit succeed or none do. There is no partial commit. |
| **Validate first** | `edits.validate` catches most errors that `edits.commit` would. Calling validate is cheap and avoids wasting quota on a doomed commit. |
| **Edit ID is opaque** | The `editId` is a server-generated string. Do not parse or cache it across sessions. |

### CLI Implication

Every mutating CLI command (e.g., `gplay tracks update`, `gplay listings update`)
should follow the full create-modify-validate-commit cycle within a single
invocation. If the command fails after creating an edit, it should attempt to
delete the edit to avoid blocking subsequent commands.

---

## Image Types and Dimensions

**Reference**: [Images API](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.images)

### Supported Image Types

| Image Type | Enum Value | Dimensions (px) | Notes |
|---|---|---|---|
| App icon | `icon` | 512 × 512 | Required. PNG or JPEG, 32-bit with alpha. |
| Feature graphic | `featureGraphic` | 1024 × 500 | Displayed on the Play Store listing page. |
| Phone screenshots | `phoneScreenshots` | Min 320px, max 3840px on any side | 2–8 screenshots required for phone form factor. |
| 7-inch tablet screenshots | `sevenInchScreenshots` | Min 320px, max 3840px on any side | Optional; recommended for tablet-optimized apps. |
| 10-inch tablet screenshots | `tenInchScreenshots` | Min 320px, max 3840px on any side | Optional. |
| TV screenshots | `tvScreenshots` | 1280 × 720 (min) | Required for Android TV apps. |
| Wear screenshots | `wearScreenshots` | Min 320px, max 3840px on any side | Required for Wear OS apps. |
| TV banner | `tvBanner` | 1280 × 720 | Required for Android TV apps. |

### Format & Size

- **Accepted formats**: JPEG, PNG (24-bit or 32-bit).
- **Maximum file size**: 15 MB per image.
- **MIME type**: Must be set correctly in the upload `Content-Type` header.
  The API rejects uploads with mismatched or missing MIME types. Detect the
  type from the file extension (`.png` → `image/png`, `.jpg`/`.jpeg` →
  `image/jpeg`).

### Screenshot Limits

- Minimum **2** screenshots per active device type.
- Maximum **8** screenshots per image type per language.
- Uploading beyond the limit returns a `400` error.

### CLI Implication

The `gplay listings images upload` command must detect MIME type from the file
extension and set the `Content-Type` header accordingly. See commit `d289d21`
for the fix that addressed this.

---

## Track Status Transitions

**Reference**: [Tracks API](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.tracks)

A **release** on a track has a `status` field that governs its lifecycle.

### Valid Statuses

| Status | Meaning |
|---|---|
| `draft` | Release is being prepared. Not visible to any users. |
| `inProgress` | Staged rollout is active at a specified percentage. |
| `completed` | Release is fully rolled out to 100% of users. |
| `halted` | Rollout was paused. Can be resumed to `inProgress` or promoted to `completed`. |

### Allowed Transitions

```
draft  ──────►  inProgress  ──────►  completed
  │                 │    ▲                ▲
  │                 ▼    │                │
  │              halted ─┘                │
  │                 │                     │
  │                 └─────────────────────┘
  └──────────────────────────────────────►
```

| From | To | Notes |
|---|---|---|
| `draft` | `inProgress` | Begins staged rollout. Must include `userFraction`. |
| `draft` | `completed` | Skips staged rollout; goes to 100%. |
| `inProgress` | `completed` | Promotes rollout to 100%. |
| `inProgress` | `halted` | Pauses the rollout. Users who already received the update keep it. |
| `halted` | `inProgress` | Resumes rollout. Can change `userFraction`. |
| `halted` | `completed` | Promotes directly to 100%. |
| `completed` | *(none)* | Terminal state. Cannot revert to `draft` or `inProgress`. |

### Staged Rollout Rules

- `userFraction` must be between `0.0` (exclusive) and `1.0` (exclusive) for
  `inProgress` status. A value of `1.0` means 100% — use `completed` instead.
- When updating `userFraction`, the new value should typically be **higher**
  than the previous value. The API does allow decreasing it, but the users who
  already received the update will not be rolled back.
- Setting status to `completed` automatically clears `userFraction`.

### Track Names

Standard tracks: `production`, `beta`, `alpha`, `internal`.
Custom tracks are also supported and are referenced by their track name string.

---

## Rate Limits and Quota

**Reference**: [Usage Limits](https://developers.google.com/android-publisher/quotas)

### Default Quota

| Limit | Value |
|---|---|
| Queries per day | 200,000 |
| Queries per 100 seconds per user | 100 (approximately) |

- Quota is applied **per Google Cloud project**, not per app or per service
  account key. Multiple service account keys from the same project share the
  same quota pool.
- Upload operations (bundles, APKs, images) may consume more quota units than
  simple read operations.

### 429 Too Many Requests

When quota is exceeded the API returns HTTP `429` with a `Retry-After` header
(not always present) or an error body indicating the quota limit.

**Retry strategy**:

1. Use **exponential backoff** starting at 1 second: 1s → 2s → 4s → 8s → …
2. Add **jitter** (±20%) to avoid thundering herd.
3. Respect `Retry-After` header when present.
4. Cap retries at a configurable maximum (default: 3, via `GPLAY_MAX_RETRIES`).

### CLI Implication

The `playclient` HTTP layer implements automatic retry with exponential backoff
for `429` and `5xx` responses. The base delay is configurable via
`GPLAY_RETRY_DELAY` (default `1s`), and maximum retries via `GPLAY_MAX_RETRIES`
(default `3`).

---

## Error Response Formats

### Standard Error JSON

All API errors follow the Google Cloud error format:

```json
{
  "error": {
    "code": 403,
    "message": "The caller does not have permission",
    "status": "PERMISSION_DENIED",
    "errors": [
      {
        "message": "The caller does not have permission",
        "domain": "androidpublisher",
        "reason": "permissionDenied"
      }
    ]
  }
}
```

### Common Error Codes

| HTTP Code | Meaning | Common Cause |
|---|---|---|
| `400` | Bad Request | Validation failure — missing required field, invalid image dimensions, invalid release status transition. |
| `401` | Unauthorized | Expired or invalid access token. Service account token refresh may have failed. |
| `403` | Forbidden | Service account lacks permissions in Play Console. Check that the account has the correct app-level or account-level access. |
| `404` | Not Found | Invalid `editId` (expired or already committed), nonexistent package name, or nonexistent track. |
| `409` | Conflict | Another edit is already open for this app. Delete or commit the existing edit first. |
| `429` | Too Many Requests | Quota exceeded. Apply exponential backoff and retry. |
| `500` | Internal Server Error | Transient server-side failure. Safe to retry with backoff. |
| `503` | Service Unavailable | Temporary overload or maintenance. Retry with backoff. |

### Extracting Actionable Information

- The top-level `message` field usually contains a human-readable description.
- The nested `errors[].reason` field provides a machine-readable error reason
  (e.g., `editAlreadyCommitted`, `permissionDenied`, `quotaExceeded`).
- For `400` validation errors, the `message` often contains specific field-level
  details (e.g., "Release status must be 'draft' or 'completed'").

### CLI Implication

The CLI's error handling should:
1. Parse the error JSON and display the `message` field to the user.
2. For known error codes (409, 429), add contextual guidance (e.g., "Another
   edit is active. Use `gplay edits delete` to clean it up.").
3. In `--debug` mode, print the full error JSON for troubleshooting.

---

## Pagination Behavior

**Reference**: Applies to list endpoints such as
[reviews.list](https://developers.google.com/android-publisher/api-ref/rest/v3/reviews/list).

### Token-Based Pagination

The API uses opaque **page tokens** for pagination:

- Request: `?pageToken=<token>&maxResults=<n>`
- Response includes `tokenPagination.nextPageToken` if more results exist.
- When `nextPageToken` is absent or empty, all results have been fetched.

### Default Page Sizes

| Endpoint | Default `maxResults` | Maximum `maxResults` |
|---|---|---|
| `reviews.list` | 10 | 100 |
| Other list endpoints | Varies | Varies |

Page sizes vary by endpoint. The API silently clamps values that exceed the
maximum rather than returning an error.

### Quirks

| Behavior | Detail |
|---|---|
| **Token expiry** | Page tokens can expire if too much time passes between requests. Restart pagination from the beginning if you receive a `400` with an invalid token error. |
| **No total count** | Most list endpoints do not return a total result count. You must paginate until `nextPageToken` is absent. |
| **Ordering** | Result ordering is determined by the API and cannot be customized via query parameters for most endpoints. |

### CLI Implication

The `--paginate` flag causes the CLI to automatically follow `nextPageToken`
until all pages are consumed, aggregating results into a single JSON array (or
table). Without `--paginate`, only the first page is returned.

---

## Upload Behavior

**Reference**: [Bundles](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.bundles),
[APKs](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.apks),
[Images](https://developers.google.com/android-publisher/api-ref/rest/v3/edits.images)

### Upload Types

| Resource | Upload Protocol | Typical Size |
|---|---|---|
| App bundles (`.aab`) | Resumable upload | 10 MB – 150 MB+ |
| APKs (`.apk`) | Resumable upload | 10 MB – 150 MB+ |
| Images | Simple (single-request) upload | < 15 MB |

### Resumable Uploads (Bundles & APKs)

Resumable uploads follow the [Google resumable upload protocol](https://developers.google.com/android-publisher/upload):

1. **Initiate**: `POST` to the upload URI with `X-Upload-Content-Type` and
   `X-Upload-Content-Length` headers. Returns a resumable session URI.
2. **Upload chunks**: `PUT` data to the session URI. Can be a single request
   for small files or multiple chunks for large files.
3. **Completion**: The final chunk returns the created resource.

If a resumable upload is interrupted, it can be resumed by querying the session
URI for the last received byte and continuing from there.

### Simple Uploads (Images)

Image uploads use a single multipart `POST` request:

- `Content-Type` must match the image format (`image/png` or `image/jpeg`).
- Maximum size: 15 MB.
- The image is sent as the request body (not multipart form data for the
  `edits.images.upload` endpoint — it uses media upload).

### Timeout Considerations

- Bundle/APK uploads can take several minutes on slow connections. The CLI uses
  a separate **upload timeout** (`GPLAY_UPLOAD_TIMEOUT`, default `5m`) that is
  longer than the standard request timeout (`GPLAY_TIMEOUT`, default `120s`).
- Use `shared.ContextWithUploadTimeout` for upload operations and
  `shared.ContextWithTimeout` for all other API calls.

### Content-Type Requirements

| File Type | Required Content-Type |
|---|---|
| `.aab` | `application/octet-stream` |
| `.apk` | `application/vnd.android.package-archive` |
| `.png` | `image/png` |
| `.jpg` / `.jpeg` | `image/jpeg` |

Omitting or mismatching the `Content-Type` causes a `400` error with an
unhelpful message. Always set it explicitly based on file extension.

---

## Further Reading

- [Google Play Developer API Overview](https://developers.google.com/android-publisher)
- [REST API Reference (v3)](https://developers.google.com/android-publisher/api-ref/rest)
- [Getting Started Guide](https://developers.google.com/android-publisher/getting_started)
- [Upload Guide](https://developers.google.com/android-publisher/upload)
- [Quota & Rate Limits](https://developers.google.com/android-publisher/quotas)
