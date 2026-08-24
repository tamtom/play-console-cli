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

## Store-listing experiments

Google's reviewed public Android Publisher discovery document currently has no
store-listing experiment lifecycle or results resource. `gplay experiments
support` reports that boundary offline from the checked-in schema. Creating an
experiment, reading its measurements, and choosing a winner therefore remain a
manual Play Console workflow.

After a person chooses the winner, `gplay experiments apply-winner` can apply
that winner's local listing text and images through the documented
`edits.listings` and `edits.images` APIs. It requires the winner name twice and
uses the sealed, resumable sync transaction. It never reads or infers results,
uses browser credentials, or calls a private Console interface.

## Local-only helpers

`gplay android build`, `gplay android signing`, `gplay android screenshots`,
`gplay validate --offline`, and `gplay install-skills` operate locally. They do
not authenticate to Google. The skills installer is pinned to a reviewed Git
commit, verifies every skill tree, executes no repository code, and installs
transactionally.
