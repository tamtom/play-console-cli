# GPlay Parity Roadmap

This roadmap replaces the earlier local parity notes with a tracker that matches
the current repository state and the live GitHub issues.

## Principles

- Copy ASC's product quality and engineering discipline, not Apple-only
  workflows.
- Prefer canonical Play-specific flows over raw API-shaped command taxonomies.
- Keep architecture work ahead of command-surface expansion when the two are in
  tension.
- Keep issue scope single-purpose so work can run in parallel with worktrees.

## Applicable Vs Not Applicable

### Applicable

- Canonical publish workflow for Google Play releases
- Aggregated status dashboard
- Canonical release-readiness validation
- Workflow engine improvements
- Vitals and reporting integrations
- Screenshots and store-media sync
- Runtime and client architecture cleanup
- Stronger testing standards and black-box CLI coverage

### Not Applicable To Copy Literally

- App Store submission and App Review resource flows
- TestFlight-specific concepts
- Xcode, notarization, or Apple signing workflows
- ASC Studio and other Apple-platform-only app surfaces

For Play, the right goal is feature quality parity, not command-count parity.

## Active Tracker Issues

- [#188](https://github.com/tamtom/play-console-cli/issues/188) canonical
  publish command family
- [#189](https://github.com/tamtom/play-console-cli/issues/189) top-level
  status dashboard
- [#190](https://github.com/tamtom/play-console-cli/issues/190) canonical
  validate command
- [#191](https://github.com/tamtom/play-console-cli/issues/191) workflow engine
  upgrade
- [#192](https://github.com/tamtom/play-console-cli/issues/192) screenshots and
  store-media pipeline
- [#193](https://github.com/tamtom/play-console-cli/issues/193) Vitals reporting
  integrations
- [#194](https://github.com/tamtom/play-console-cli/issues/194) testing
  standards
- [#195](https://github.com/tamtom/play-console-cli/issues/195) black-box CLI
  coverage
- [#196](https://github.com/tamtom/play-console-cli/issues/196) client domain
  split
- [#197](https://github.com/tamtom/play-console-cli/issues/197) runtime package
- [#198](https://github.com/tamtom/play-console-cli/issues/198) roadmap refresh

## Wave Overview

- Wave 0: planning, test standards, and isolated reporting work
- Wave 1: foundation architecture
- Wave 2: high-level product surfaces
- Wave 3: automation and stabilization

Detailed plans live in:

- `.parity-plan/wave-0.md`
- `.parity-plan/wave-1.md`
- `.parity-plan/wave-2.md`
- `.parity-plan/wave-3.md`

## Merge Order

1. Wave 0: `#198`, `#194`, `#193`
2. Wave 1: `#196`, `#197`
3. Wave 2: `#190`, `#188`, `#192`, `#189`
4. Wave 3: `#191`, `#195`

## Branch Naming

Use one branch and one worktree per issue:

- `codex/issue-188-canonical-publish`
- `codex/issue-189-status-dashboard`
- `codex/issue-190-canonical-validate`
- `codex/issue-191-workflow-upgrade`
- `codex/issue-192-media-pipeline`
- `codex/issue-193-vitals-reporting`
- `codex/issue-194-testing-standards`
- `codex/issue-195-cli-blackbox-expansion`
- `codex/issue-196-client-domain-split`
- `codex/issue-197-runtime-package`
- `codex/issue-198-roadmap-refresh`

## Contributor Rules

- Do not reopen old ASC parity assumptions blindly; use the active tracker
  issues above.
- Do not combine multiple tracker issues into one PR.
- When in doubt, align new command surfaces with `publish`, `status`, and
  `validate` rather than adding more low-level one-off commands.
