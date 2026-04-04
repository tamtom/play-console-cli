# Wave 1

Foundation architecture. Run in parallel, merge carefully.

## Issues

- [#196](https://github.com/tamtom/play-console-cli/issues/196) split
  `playclient` and `reportingclient` by domain
- [#197](https://github.com/tamtom/play-console-cli/issues/197) introduce
  runtime package and thin command constructors

## Coordination Rules

- `#196` sets the long-term domain boundaries for API clients.
- `#197` sets the long-term root/runtime wiring pattern.
- If both need to touch the same call site, prefer `#196` first so `#197` can
  bind to the post-split structure.

## Dependencies

- `#196`: none
- `#197`: can start in parallel, but final wiring should follow the agreed
  client package boundaries from `#196`

## Merge Order

1. `#196`
2. `#197`
