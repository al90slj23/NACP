# CAF classic-plus-0.2.0 Local API Test Report

> Date: 2026-05-16
> Target: local Stage A, `http://localhost:5173/` for dashboard APIs, backend relay at `/api/status.server_address`
> Version file: `classic-plus-0.2.0-dev`
> Gate result: FAIL initially; P0 fixed and core local API retest passed

## Scope

This report records the local API-first execution of the `classic-plus-0.2.0-dev` test plan before any online test-server deployment.

The main focus was the 0.2.0 SFT log/trace changes:

- `/api/log/grouped`
- `/api/log/traces`
- `/api/log/trace`
- compatibility with native NewAPI `logs.type`
- normal user -> token -> relay request -> admin log verification
- billing, usage, and statistics not being polluted by SFT probe/intercept rows

## Passed Checks

| Area | Result | Evidence |
|---|---:|---|
| Local frontend API reachability | PASS with note | `GET /api/status` returned success through `localhost:5173` |
| Backend unit tests | PASS | `go test ./service -run 'TestGroupedLogs|TestTrace'` passed |
| Admin login | PASS | admin session login returned role `100`, id `1` |
| Admin header protection | PASS | `/api/log/grouped` without `New-API-User` returned unauthorized |
| `/api/log/grouped` default list | PASS | returned 25 rows, standalone requestless probe rows were not surfaced |
| `/api/log/traces` grouping | PASS | returned 2 request summaries, no duplicate request IDs |
| Virtual type compatibility | PASS | `type=20/21/29/50/51/52/59` all returned `total=0`; these are display-derived types, not stored `logs.type` |
| Success chain detail | PASS | request `202605151634024149938738268d9d6m3vEE6rT` ended with `trace_role=consume`, quota `111`, prompt `121`, completion `20` |
| Failure chain detail | PASS | request `202605151641575233889398268d9d6I356YgeP` ended with `trace_role=error_visible`, quota/tokens all `0` |
| Native type filter | PASS | `type=2` returned only consume logs; `type=5` returned error logs; `type=4` had no current local samples |
| Admin log stat | PASS | quota stat `489` matched the sum of existing local consume rows |
| Normal user creation | PASS | created local user `nacp_stagea_0516_001`, id `13` |
| Normal user quota grant | PASS | added quota to user id `13` |
| Normal user token creation | PASS | created token `stagea-token-0516`, id `13`; real key was retrieved for local testing only |

## Blocking Finding

### P0: Relay returns HTTP 200 to client but logs and billing mark the request as failed

This is a release blocker because the external client receives a successful response, while NACP records the same request as failure and does not create a consume log or charge usage.

Reproduced twice with the normal user/token created during this test.

| Case | Client result | Log result | Billing/stat result |
|---|---|---|---|
| `/v1/responses`, model `gpt-5.3-codex` | HTTP 200, body status `completed`, output `OK` | request `202605152023061702890008268d9d6ZnFfvIXQ` logged as `error_intercepted -> error_visible`, status 499 | user stat quota `0`; user quota unchanged after refund |
| `/v1/chat/completions`, model `claude-haiku-4-5-20251001` | HTTP 200, assistant content `OK.` | request `202605152026551197700008268d9d6fRdmev4T` logged as failed, final `error_visible` | user stat quota `0`; no consume log |

Backend runtime evidence for `/v1/responses`:

- pre-consume happened: user `13` pre-consumed `317` quota
- response route finished as `GIN ... relay ... 200 ... POST /v1/responses`
- settlement was skipped with `skip text quota settlement because request context is done: context canceled`
- error log was recorded as `status_code=499, client request closed: context canceled`
- pre-consume was refunded

Code path currently implicated:

| File | Line | Behavior |
|---|---:|---|
| `service/text_quota.go` | 320 | `PostTextConsumeQuota` aborts settlement when `RequestContextErr(ctx)` is non-nil |
| `service/billing.go` | 34 | `SettleBilling` also aborts when the request context is done |
| `controller/relay.go` | 89 | if billing was skipped because of cancellation, a request-canceled API error is returned after relay handler execution |
| `service/request_context.go` | 16 | cancellation is read directly from `c.Request.Context().Err()` |

Current hypothesis:

The request context check is too broad for successful non-stream relay responses. After the adaptor has already received upstream usage and written the successful response body, the client/request context can be done by the time quota settlement runs. The current logic treats that as client cancellation, skips settlement, returns a synthetic 499 error to the outer relay flow, records failure logs, and refunds pre-consume, even though the client already received HTTP 200.

This means the protection intended for real client-aborted requests is also catching completed successful responses.

## Secondary Findings

### P1: `localhost:5173` proxies `/api/*`, but not `/v1/*`

`POST http://localhost:5173/v1/responses` returned `404 0` and did not enter relay. The dashboard APIs are valid through `5173`, but relay calls currently need to use the backend address from `/api/status`, which was `http://localhost:3000` in this local run.

This should be standardized in CAF local testing:

- dashboard/admin/user APIs: `http://localhost:5173/api/*`
- relay APIs: `server_address` from `/api/status`, unless Vite proxy explicitly supports `/v1/*`

### P1: `/api/status` reports `v0.0.0` while `VERSION` is `classic-plus-0.2.0-dev`

Local `/api/status` returned:

- `success=true`
- `system_name=MSRL-NACP`
- `server_address=http://localhost:3000`
- `version=v0.0.0`

The repository `VERSION` file is `classic-plus-0.2.0-dev`. This looks like local build-time version injection is missing. It does not invalidate log API tests, but it weakens deployment/version verification.

## Gate Decision

Local Stage A is not passed.

Do not proceed to online test server deployment or production deployment until the P0 relay settlement/log mismatch is fixed and re-tested.

## Required Retest After Fix

After fixing the P0 issue, rerun at minimum:

1. Normal user token `/v1/responses` success.
2. Normal user token `/v1/chat/completions` success.
3. Admin `/api/log/grouped` for that user must show a consume row or a 20 summary with final consume.
4. `/api/log/trace` must end in `trace_role=consume`, not `error_visible`.
5. `/api/log/stat?username=...` quota must equal the consume log quota.
6. user `quota`, `used_quota`, `request_count`, token `remain_quota`, and token `used_quota` must reflect the actual charge.
7. Real client abort still must log 499 and must refund pre-consume.
8. SFT failure chain still must end in `error_visible` and remain uncharged.

## Fix Retest

Date: 2026-05-16

Patch summary:

- Removed the early `RequestContextErr(ctx)` abort from successful text quota settlement.
- Removed the same abort from `SettleBilling`, so a successful response with known usage still settles even if the HTTP request context has ended.
- Applied the same successful-settlement rule to audio and realtime quota paths.
- Added `TestSettleBillingStillRunsAfterRequestContextDone` to lock the intended behavior.

Code paths changed:

| File | Behavior after fix |
|---|---|
| `service/text_quota.go` | `PostTextConsumeQuota` continues settlement after usage is available |
| `service/billing.go` | `SettleBilling` no longer treats context-done as settlement failure |
| `service/quota.go` | audio/realtime successful usage settlement follows the same rule |
| `service/billing_context_test.go` | regression test for successful response settlement after context done |

Automated test result:

| Command | Result |
|---|---:|
| `GOCACHE=/private/tmp/nacp-gocache go test ./service -run 'TestSettleBillingStillRunsAfterRequestContextDone|TestGroupedLogs|TestTrace'` | PASS |
| `GOCACHE=/private/tmp/nacp-gocache go test ./service` | PASS |

Local API retest results:

| Case | Client result | Log/trace result | Billing/stat result |
|---|---|---|---|
| `/v1/responses`, model `gpt-5.3-codex` | HTTP 200, response status `completed`, output `OK` | request `202605152036545270330008268d9d668r3e3lF` has final `trace_role=consume` | quota `246`, prompt `353`, completion `5` |
| `/v1/chat/completions`, model `claude-haiku-4-5-20251001` | HTTP 200, assistant content `OK` | request `202605152037101790700008268d9d6PyTBwsUe` has SFT chain ending in `trace_role=consume` | quota `16`, prompt `12`, completion `4` |
| Admin stat for `nacp_stagea_0516_001` | PASS | `quota=262`, `rpm=2`, `tpm=374` | equals consume logs |
| User quota | PASS | `quota=999738`, `used_quota=262`, `request_count=2` | initial granted quota was `1000000` |
| Token quota | PASS | `remain_quota=99738`, `used_quota=262` | initial token quota was `100000` |
| Native type filter `type=2` | PASS | returned the two new consume rows only | no intercepted/probe rows included |
| Virtual type filter `20/21/29/50/51/52/59` | PASS | all returned `total=0` | virtual display types did not pollute `logs.type` |
| User self logs | PASS | normal user can see own consume/intercept rows | admin-only `admin_info` is not exposed in self log output |

Retest note:

- `/api/log/traces` returns SFT/request-chain summaries. The direct one-step success request `202605152036545270330008268d9d668r3e3lF` is visible via `/api/log/grouped` and `/api/log/trace?request_id=...`, but not listed as a chain summary. This matches the current UI decision that direct type `2` successes do not need expansion.
- Real connected client-abort was retested with a Node HTTP client running outside the sandbox. The client opened a TCP connection, wrote the request body, then destroyed the socket before upstream response.

Abort retest evidence:

| Item | Result |
|---|---:|
| Client output | `CONNECTED`, then `ABORTING_CONNECTED_REQUEST`, then local client error |
| request_id | `202605152048077554230008268d9d6HVlyXOHL` |
| trace final role | `error_visible` |
| stored log type | native `type=5` |
| status/content | `499`, `client request closed: context canceled` |
| quota/tokens | all `0` |
| user quota after abort | unchanged: `quota=999738`, `used_quota=262`, `request_count=2` |
| token quota after abort | unchanged: `remain_quota=99738`, `used_quota=262` |

Expanded API checks after the fix:

| API | Result |
|---|---:|
| `/api/data/` | PASS; model aggregates include `gpt-5.3-codex` quota `246` and `claude-haiku-4-5-20251001` quota `16` for the test hour |
| `/api/data/users` | PASS; user aggregate for `nacp_stagea_0516_001` is `token_used=374`, `count=2`, `quota=262` |
| `/api/data/self` | PASS with valid time window; returns the two model aggregates for user id `13` |
| `/api/log/grouped?token_name=stagea-token-0516` | PASS; returns only logs for that token |
| `/api/log/grouped?model_name=gpt-5.3-codex` | PASS; returns Codex consume/error rows |
| `/api/log/grouped?request_id=202605152037101790700008268d9d6PyTBwsUe` | PASS; returns the exact SFT chain including probe rows and final consume |
| `/api/log/search` | Expected deprecated; returns `该接口已废弃` |

Build and full test checks:

| Command | Result |
|---|---:|
| `bun run build` in `web/` | PASS; production build completed, with existing chunk-size / lottie eval warnings |
| `GOCACHE=/private/tmp/nacp-gocache go test ./...` | PASS after fixing two additional regressions found by the first full run |
| `bun run eslint` in `web/` | FAIL globally because the script scans generated `web/dist` assets and unrelated existing header/blank-line issues |
| targeted ESLint for changed usage-log files | PASS |

Additional regressions found by `go test ./...` and fixed:

| Area | Symptom | Fix |
|---|---|---|
| Claude file conversion | OpenAI `file` content without explicit MIME was converted as Claude `image`; `.pdf`, `.txt`, and unsupported `.bin` semantics were wrong | preserve MIME from filename extension, convert PDF to `document`, image MIME to `image`, text MIME to Claude text, skip unsupported file types |
| Stream scanner status | `StreamScannerHandler` always replaced a pre-initialized `StreamStatus`, dropping pre-existing soft errors | only create `StreamStatus` when it is nil |
| Usage log frontend lint | touched `TraceExpandRender.jsx` was missing the project license header | added the same protected QuantumNous/AGPL header pattern used by adjacent usage-log files |

Local server state after rebuild:

- backend rebuilt to `/private/tmp/nacp-main`;
- backend restarted in tmux session `nacp-backend`;
- `127.0.0.1:5173/api/status` returns HTTP 200 through the existing Vite dev server;
- a temporary 5174 dev server was started during port diagnosis and then stopped.

Updated gate:

The original P0 is fixed. Core local API retest, connected-abort retest, log filters, quota/stat consistency, and data aggregation checks have passed.
