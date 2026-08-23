# Policy-safe automation boundary

Gplay uses documented Google APIs for supported operations and explicit manual
Play Console handoffs for unsupported initial setup. It does not attempt to
work around the public API boundary.

## Prohibited implementations

Gplay must not:

- call reverse-engineered or private Play Console RPC endpoints;
- import, retain, or transmit authenticated Google browser cookies;
- drive authenticated Play Console forms through CDP, WebDriver, or DOM scripts;
- automate acceptance of agreements, declarations, or other legally meaningful text;
- silently fall back from a documented API failure to Console automation.

## Initial app setup

`gplay bootstrap plan` is deliberately non-executing. It validates a local AAB,
produces a stable plan ID, and identifies the steps that the account owner must
complete directly in Play Console. It does not authenticate, contact Google,
open a browser, upload an artifact, submit a release, or accept an agreement.

After the initial app record and first artifact exist, use documented API-backed
commands such as `gplay publish track` for later release workflows.

Use `gplay capabilities` to inspect whether a workflow is classified as
`official`, `manual`, or `unsupported`.

## Testing boundary

Capability and bootstrap tests are offline. They use temporary local files and
in-process CLI execution, and they make no Google requests. No personal or
production Play Console account is required or permitted for these tests.

Any separately enabled integration test for an existing official API must use a
dedicated test resource and explicit opt-in. It is not part of bootstrap testing.
