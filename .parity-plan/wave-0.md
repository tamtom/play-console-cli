# Wave 0

These tasks can run immediately in parallel.

## Issues

- [#198](https://github.com/tamtom/play-console-cli/issues/198) roadmap refresh
- [#194](https://github.com/tamtom/play-console-cli/issues/194) testing
  standards
- [#193](https://github.com/tamtom/play-console-cli/issues/193) Vitals
  integrations

## Ownership

- `#198` owns roadmap and sequencing only
- `#194` owns testing guidance and standards only
- `#193` owns `internal/cli/vitals/**`, `internal/reportingclient/**`, and
  direct tests for those areas

## Dependencies

- `#198`: none
- `#194`: none
- `#193`: none for initial work; coordinate with `#196` before any broader
  `reportingclient` refactor lands

## Merge Order

1. `#198`
2. `#194`
3. `#193`
