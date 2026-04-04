# Wave 2

High-level product surfaces. These need a shared design pass before coding.

## Issues

- [#190](https://github.com/tamtom/play-console-cli/issues/190) canonical
  `validate`
- [#188](https://github.com/tamtom/play-console-cli/issues/188) canonical
  `publish`
- [#192](https://github.com/tamtom/play-console-cli/issues/192) screenshots and
  store-media pipeline
- [#189](https://github.com/tamtom/play-console-cli/issues/189) top-level
  `status`

## Shared Design Decisions

Agree on these before implementation:

- canonical command naming
- shared release and rollout terminology
- reusable data types for release health and readiness
- media directory conventions
- whether `publish` calls `validate` directly or shares a lower-level readiness
  package

## Coordination Rules

- `#190` defines readiness semantics
- `#188` consumes readiness semantics and defines the canonical publish path
- `#192` aligns with the media conventions and optional publish integration
- `#189` consumes the final shared release and vitals models

## Dependencies

- all Wave 2 issues depend on Wave 1
- `#188` should not finalize before `#190`
- `#189` should not finalize before `#190` and `#193`

## Merge Order

1. `#190`
2. `#188`
3. `#192`
4. `#189`
