# Official Google Play API coverage

The reviewed discovery manifest covers 218 methods across eight Play-specific
Google APIs. Cloud Storage is intentionally limited to object listing/download
for Play financial and statistics reports.

| API | Methods | Gplay coverage |
|---|---:|---|
| Android Publisher v3 | 145 | All methods; third-party-store and enterprise-KMS methods are scope-gated |
| Play Developer Reporting v1beta1 | 25 | All methods through typed commands or `vitals metric-sets` |
| Checks v1alpha | 15 | All methods, including repository scans |
| Games Configuration | 10 | All methods |
| Games Management | 18 | All methods; global resets require exact application confirmation |
| Custom App Publishing | 1 | Complete |
| Play Integrity v1 | 3 | Complete; Device Recall requires restricted-use acknowledgement |
| Android Developer ID Status v1 | 1 | Complete, read-only |

Run `make check-api-drift` to compare the reviewed manifest with Google's live
official discovery documents. A credential-free scheduled workflow runs the
same check weekly. After reviewing a legitimate API change and updating the
implementation/tests, run `make update-api-manifest`.

## Specialized namespaces

- `gplay app-stores` is only for organizations registered in Google's
  third-party app-store program. It is not a normal Play publishing namespace.
- `gplay app-signing` is only for enterprise self-hosted Google Cloud KMS key
  custody. Commands require an exact `--confirm-package` match. Ordinary
  Google-managed signing enrollment remains a manual Console workflow.
- `gplay integrity device-recall-write` is restricted to security, fraud, and
  abuse prevention and requires an explicit acknowledgement.
- Play Games global reset commands delete tester progress across every player
  and require `--application-id`, a matching `--confirm-application-id`, and
  `--confirm`. Before resetting, gplay verifies that the authenticated caller
  can read that exact Play Games application through the Configuration API.

## Testing

Unit and CLI contract tests use local fixtures and `httptest`; they do not call
Google or require an account. Live integration tests are opt-in and use only
documented official APIs. High-impact operations (refund review, enterprise key
enrollment/rotation, Device Recall writes, repository scan generation, and
global Games resets) remain mocked unless a dedicated disposable test resource
is explicitly configured.
