# 2026-05-16 classic-plus-0.2.0 local API test report

## Change

Added the local Stage A execution report for `classic-plus-0.2.0-dev`:

- `.ai/L5#Knowledge/caf-classic-plus-0-2-0-local-api-report.md`

## Result

Local Stage A is marked `FAIL`.

Main blocker:

- Relay can return HTTP 200 to the client while billing/log settlement records the same request as 499 failure and does not create a consume log.

The report also records passed log API checks, normal user/token setup, and secondary local environment issues.

## Follow-up fix retest

The P0 was fixed in code and retested locally:

- successful `/v1/responses` now creates a `consume` log and charges quota;
- successful `/v1/chat/completions` through SFT now ends in `consume`;
- admin stats, user quota, token quota, and native type filters are consistent;
- `go test ./service` passes.

The report now marks the original P0 as fixed for core local API flows, with a remaining pending connected client-abort integration sample.

## Connected abort retest

The pending abort sample was completed with a Node HTTP client running outside the sandbox:

- TCP connection established;
- request body written;
- client socket destroyed before upstream response;
- server logged native `type=5`, `trace_role=error_visible`, status `499`;
- user quota/token quota/stat totals stayed unchanged.

Additional API checks were added for `/api/data`, `/api/data/users`, `/api/data/self`, grouped log filters, and deprecated `/api/log/search` behavior.

## Full test expansion

Ran broader verification:

- `bun run build` in `web/` passed.
- first `go test ./...` exposed Claude file conversion and stream scanner status regressions.
- fixed both regressions.
- second `go test ./...` passed.

The local backend was rebuilt to `/private/tmp/nacp-main` and restarted in tmux session `nacp-backend`.

## Frontend lint follow-up

`bun run eslint` fails globally because it scans generated `web/dist` assets and unrelated existing source files with header/blank-line violations.

For the files touched by the usage-log work:

- added the missing license header to `TraceExpandRender.jsx`;
- targeted ESLint for usage-log files passed.
