# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**This repo is a starter kit**, not a product. It is the Go backend skeleton copied into every new web app: HTTP API + RabbitMQ event consumers + cron jobs in one binary (`cmd/server`). Postgres via sqlc (queries) and goose (migrations). Brevo sends transactional email. Firebase Storage holds uploaded files; libvips (cgo, via govips) makes image previews — `brew install vips` before building or testing.

What ships out of the box: email/password signup + signin, Sign in with Google (signup, signin, adding a password to a Google account or Google to a password account), email verification with rate-limited resend, password reset by emailed link, email + password change, account deletion, JWT access tokens with rotating refresh tokens (HttpOnly cookie), event log + RabbitMQ dispatch with dead-lettering, transactional email log, file uploads with previews (preview-only purposes drop the original), user profiles with an optional picture, cron cleanup jobs (expired refresh tokens, stale pending uploads, orphaned attached uploads). Ready-but-unused seams for later domains: `internal/pagination` + page query params, `handlers.PatchField`, `input.validateTaggedStruct`.

The app name placeholder is `web_app_starter` (module `github.com/pauliusgin/web_app_starter-be`). Changes made here are meant to be generic — nothing domain-specific to one app belongs in the kit.

## Starting a new project from this kit

1. Copy the directory (without `.git`), then `git init` and `git config core.hooksPath .githooks`.
2. Replace every `web_app_starter` with the new app name: `grep -rl --exclude-dir=.git web_app_starter . | xargs sed -i '' 's/web_app_starter/<new_name>/g'`. This covers the module path in `go.mod` + imports, Makefile container/DB/vhost names, the RabbitMQ exchange (`rabbitmq.EventExchange`, `DeadLetterExchange`) and queue names (`consumers.AuthChallengeEmailQueue`), and test fixtures. Then `go build ./... && go test ./...`.
3. `cp .env.example .env` and fill it in (see Config below).
4. `make db-build`, `make mq-build`, create the RabbitMQ vhost named in `MQ_VHOST`, then `make migrate-all`.
5. Add app-specific upload purposes in `internal/domains/uploads/write/models/upload_rules.go` (`profile_picture` and `document_attachment` are the generic defaults); every new column pointing at an upload gets a `UNION ALL` branch in the `upload_references` view, or the orphaned upload cleanup deletes what it points at. Grow domains under `internal/domains/`.
6. Rewrite the intro of this file for the new app and drop the "Starting a new project" section.

## Commands

```sh
make db-build / db-start          # Postgres 18 in Docker
make mq-build / mq-start          # RabbitMQ 4 + management UI on :15672
make mq-reset-queues              # delete consumer queues; server redeclares on start

# To run the starter, use "make db-build", "make mq-build" and populate the .env
make run                          # go run ./cmd/server
make build                        # → ./bin/server
#- To run tests, `make db-build` and `make mq-build` are enough
make test                         # go test ./... (integration tests need Docker running)
go test ./internal/test/integration/   # use cases over real Postgres in a testcontainers postgres:18
go test ./internal/auth/refreshtoken -run TestRotateRefreshToken   # single package / test

make sqlc                         # regenerate internal/adapters/repositories/postgres/sqlc from sql/
export GOOSE_DBSTRING=<db url>    # required before migrate-* targets
make migrate-all | migrate-one | migrate-down
```

Git hooks live in `.githooks/` (`git config core.hooksPath .githooks`):

- pre-commit: `gofmt -w` on staged `.go` files.
- pre-push: `sqlc diff` (generated code must match `sql/`) + `go test ./...`.

Tests touching live Postgres/RabbitMQ (`cmd/server/server_test.go`, `internal/adapters/brokers/rabbitmq/connection_test.go`) `t.Skip` when the service is down — start containers to run them. `docs/testing.md` lists every test needing an external service.

`internal/test/integration` starts its own Postgres via testcontainers-go (`internal/test/support/postgrestest`): one container per run, migrations from the embedded `sql/schema` (`schema.Migrations`, goose library), and each test gets a fresh database cloned from the migrated template (`testPostgres.NewTestDatabase(t)`). Without Docker those tests skip; `INTEGRATION_REQUIRED=1` turns the skip into a failure (for CI).

## Config

Env-only (`internal/config`), read from `.env` if present; `.env.example` lists every var. Required: `DATABASE_URL`, `RABBITMQ_URL`, `JWT_SECRET`, `FRONTEND_BASE_URL`, `APP_ENV`, `BREVO_API_KEY`, `BREVO_SENDER_EMAIL`, `BREVO_EMAIL_VERIFICATION_TEMPLATE_ID`, `BREVO_PASSWORD_RESET_TEMPLATE_ID`, `GOOGLE_FIREBASE_CREDENTIALS` (base64 service account JSON), `GOOGLE_FIREBASE_STORAGE_BUCKET`, `GOOGLE_CLIENT_ID` (OAuth client Sign in with Google ID tokens must be issued for). `APP_ENV` must be one of `config.AppEnv` (`local`, `dev`, `prod`). `SERVER_HOST` defaults to 0.0.0.0, `SERVER_PORT` to 3000. `config.Load` reports all missing vars at once via `errors.Join`.

## Architecture

Hexagonal layout: domain declares ports, adapters implement them, `cmd/server` wires everything.

- `cmd/server/` — composition root only. `server.go` runs startup as a `startupSteps` struct of funcs (swappable in tests): config → email gateway → DB → broker → consumers → scheduler → router deps → listen → graceful shutdown on SIGINT/SIGTERM. `router.go` (`buildRouterDependencies`), `consumers.go`, `scheduler.go` build the dependency graphs.
- `internal/domains/<domain>/` — use cases + ports. Every client-facing domain (`users`, `profiles`, `uploads`) splits CQRS-style into `write/` (models, `<x>_write_repository.go` port, usecases; also internal lookups such as `FindUserByID`) and `read/` (client-facing read models, `<x>_read_repository.go` port with `Get<X>ReadModelByID`, `queries` such as `GetUserByID`), even while the two models match; internal-only domains (`messaging`) stay flat. Write handlers answer the read model by re-reading through the resource's get-by-id query after the write. Each domain has its own `errors/` package.
- `internal/auth/` — auth primitives as small packages: `challenge` (email/phone verification and password reset tokens, per-purpose lifetimes, `EnsureIssuanceAllowed` rate limits per target and per user), `refreshtoken` (issue/rotate/revoke/cleanup), `jwt` (access tokens), `password` (argon2id), `token` (random + hash), `googleidtoken` (verified Google identity + `Verifier` port, `InMemoryVerifier` fake).
- `internal/events/` — domain events. `Dispatcher.Dispatch` records to `event_log` table and publishes to RabbitMQ; failures are **logged, never returned** (event dispatch never fails a use case). Per-event constructors live in subpackages (`events/user`, `events/auth`, ...) as `NewX` + `DispatchX`. Labels in `event_label_list.go` double as routing keys. Use `RedactedData` when the payload carries a secret (e.g. challenge token) — the log stores the redacted copy, subscribers get the full one.
- `internal/api/http/` — `routes` (one `<resource>_routes.go` per resource exporting `XRoutePath` + `RegisterXRoutes(mux, XRoutesDependencies)`; `BuildRouter` calls each on a single `http.ServeMux` — no nested muxes or trailing-slash prefixes; patterns are full Go 1.22 `"METHOD XRoutePath/sub/{wildcard}"` strings, and `XIDPathValue` consts hold the bare wildcard name (no slash) since handlers read it with `r.PathValue`; unmatched paths/methods keep the mux's plain-text 404/405 on purpose — don't wrap them in `ErrorResponse` or add `route_not_found`/`method_not_allowed` codes), `handlers` (one package per resource, laid out as `<usecase>.go` + `<usecase>_test.go` per handler, `validation_rejection_rules.go` for rule sets shared across its handlers, `errors.go`; each handler file owns its request and response structs — no shared `*_request.go`/`*_response.go`; the client-facing read models carry the json tags (a chosen coupling, HTTP is the only transport), and every handler still declares its response explicitly as a struct embedding the read model — `type createClientResponse struct { readmodels.ClientReadModel }`, built by `buildCreateClientResponse` — adding fields beside it when the answer carries more (`signUpUserResponse` adds `AccessToken`); list envelopes hold `[]listedXResponse` items embedding the read model the same way; only answers with no read model (e.g. Google sign-in config) list their fields; read-model calendar dates (DATE columns) use a type marshalling `"YYYY-MM-DD"`, never a bare `time.Time`; read-model list fields are never nil (producers build them with `make(..., 0, n)`, integration tests pin it) so they encode as `[]`; map domain errors → `{status, code}` rejection via `errors.As` switch; failures answer `handlers.ErrorResponse{code, message, details}` (`details` null when unset) where `code` is a snake_case `handlers.ErrorCode` from the fixed list in `handlers/errorcode.go` — add new codes there, clients branch on them; rejections that carry data use `handlers.RejectWhenWithDetails` (e.g. `verification_email_rate_limited` → `details.retryAfterSeconds` + `Retry-After`); rejections are not logged, 500s log the cause and answer `internal_error` with a generic message; collapse distinct rejections into one code when distinct codes would leak account state, as `access_token_invalid`/`refresh_token_invalid`/`password_reset_link_invalid` do, and hold such answers to a constant duration with `handlers.WaitForResponseFloor` where timing would leak it (password reset email); responses send an explicit `null` for every missing value — never `""`, a zero time or a dropped key: a field that can be missing is a pointer, no `omitempty` anywhere under `handlers/` or in a read-models package (guarded by `handlers/omitempty_guard_test.go` and a per-package `read/models/omitempty_guard_test.go`); copy pointers straight from read models whose mapper knows the column is NULL, and convert domain zero values at the handler with `handlers.NullableString` / `handlers.NullableTime`; handler tests decode into `map[string]any` and assert the key is present with a `nil` value), `middleware` (`RequireAuthenticatedUser`), `validators` (`primitive` = email/password/name rules, `input` = per-request; `input.ValidateXInput(raw) (XInput, error)` trims/normalizes and returns the cleaned input — handlers never trim, except future PATCH `PatchField` readers whose absent/null/blank semantics are request-shape decoding; inputs with many optional fields declare rules as `validate` tags checked by the shared `input.validateTaggedStruct` (go-playground/validator) — custom tags wrap `primitive` funcs, and a `map[fieldRule]func(value string) error` maps each failed `{field, tag}` to a typed `primitive.Err*` so error codes stay stable; an unmapped tag surfaces as `ErrValidationRuleUnmapped` → 500; `ValidatePageRequestInput` is the one validator that returns a parsed type, `pagination.PageRequest`, since parsing turns its raw strings into numbers).
- Refresh tokens never appear in a JSON body: signup/signin/refresh set them as the `refresh_token` cookie (`handlers/tokens/refresh_token_cookie.go`: HttpOnly, Secure, SameSite=Strict, `Path` = `routes.TokenRoutePath`, guarded by `routes_test`); refresh and revoke read it from there, and any refresh rejection or revoke clears it. Bodies carry only `accessToken`. The browser client must reach the API same-site (e.g. dev proxy) and send `credentials: "include"` on token routes — no CORS layer ships.
- `internal/api/messaging/` — inbound side of RabbitMQ. `consumers.BuildRegistry` declares every consumed event (label → queue → handler); handlers in `handlers/messaging` translate payloads into domain use case calls. Consumers with nil handlers are skipped with a warning.
- `internal/adapters/` — `repositories/postgres/<domain>/{repositories,mappers}` wrap generated sqlc queries; `brokers/rabbitmq` (topic exchange, publisher confirms, dead-letter topology); `gateways/brevo` (email); `gateways/firebase` (upload file storage); `gateways/google` (Sign in with Google ID token verification, `google.golang.org/api/idtoken` + explicit issuer check); `imaging/libvips` (image inspection + WebP previews). Stripe gateway is a placeholder.
- `internal/scheduler/` — cron wrapper (robfig/cron); jobs registered in `cmd/server/scheduler.go`.

### Dependency injection pattern

No DI container. Every use case is a plain function `func X(ctx, d XDependencies, params XParams) (XResult, error)` — `XParams` (input) → `XResult` (output, a named struct) → `XDependencies` declared at the top of the file; error-only use cases return plain `error`, no Result. Params/Result carry no `json` tags. Callers depend on a **function type**, not the use case directly — `cmd/server` closes over the dependencies:

```go
signUpUser := func(ctx context.Context, params userusecases.SignUpUserParams) (userusecases.SignUpUserResult, error) {
    return userusecases.SignUpUser(ctx, signUpUserDependencies, params)
}
```

Handlers take a `XDependencies` struct of these func types. Each handler file opens with its unexported json-tagged `xRequest` struct (only where a JSON body exists): the body decodes into `xRequest` — never into use case Params. The `xResponse` struct embeds the resource's read model (re-read after a write) rather than copying its fields, so the response stays visible in the handler while the read model owns the keys. Adding a feature = use case + deps struct → wire closure in `cmd/server` → plug into router/registry deps.

### Adding a DB query

1. Migration in `sql/schema/<YYYYMMDDHHMM>_<name>.sql` with `-- +goose Up` / `-- +goose Down`.
2. Query in `sql/queries/<table>_{read,write}.sql` with `-- name: X :one|:many|:exec`.
3. `make sqlc`.
4. Repository method in `internal/adapters/repositories/postgres/<domain>/repositories` + mapper; `sql.Null*` ↔ Go value conversions go through `postgres/sqlnull` (`StringPointer`, `TimePointer`, `UUIDPointer`, `StringUnlessEmpty`, `Int32UnlessZero`, `UUIDUnlessNil`), not private helpers; translate `sql.ErrNoRows` → domain `ErrXNotFound`, other errors → domain `ErrXFailed{Cause}`. Add `var _ port.Interface = Impl{}` compile-time check.
5. Any rule living in the SQL (ordering, ownership, soft delete, CTE side effects) gets a flow in `internal/test/integration` (write through the use case, read back through the query). If the port has a shipped in-memory fake, add the rule to its contract suite too.

Soft delete: queries filter `deleted_at IS NULL`. Deleting an account also frees what other rows must not keep holding: `users.google_subject` (so the Google account can sign up again) and `profiles.picture_upload_id` (so the orphaned upload cleanup can delete the picture).

### Read vs write user repos

Write repo (`FindUserByID`, `FindUserByEmail`) serves internal checks (existence, verification state) — including auth middleware. Read repo (`GetUserReadModelByID`) serves client-facing responses only (signup, signin, `GET /users/me`).

## Conventions

- Every package has a one-line package doc; every exported name has a one-line doc comment starting with its name.
- Structured params everywhere: `XParams` / `XDependencies` structs instead of positional args (including in `cmd/server` private helpers). `ctx` (and an `http.ResponseWriter`) stay positional; everything else goes in the struct, so helpers take `(ctx, params)` or `(ctx, d, params)`.
- Each package has `errors.go` with `Err<What>` struct types (value receiver `Error()`, exported fields carrying context, `Cause` + `Unwrap()` for wrapped errors). Shared message text in package-level `const ...Text`. Match errors with `errors.As`, never strings.
- Logging via `log/slog` with key-value pairs (`"userId", id`, `"error", err`).
- Tests: no mocks. Use in-memory implementations of ports with failure-injection fields (`SaveFailure error`): shipped ones (`InMemoryUserRepository`, `InMemoryProfileRepository`, `InMemoryUploadRepository` — the last two implement both the write and the read port —, `challenge.InMemoryChallengeRepository`, `refreshtoken.InMemoryRefreshTokenRepository`, …) — a fake that mirrors SQL ships, it is not hand-rolled per test package. Wire real use cases over them rather than stubbing the use case. Event capture through `eventstest.CapturedEvents`. Fixed `time.Date(...)` fixtures (far-future/far-past dates where a fake compares against the wall clock, as `NOW()` would). `email.InMemoryGateway` is a shipped fake for the email port. Assert on returned values and resulting store state, never on which functions were called.
- Shipped fakes that mirror SQL rules are held to it by a contract suite (`<x>repositorytest.Run<X>RepositoryContract` for users, profiles, uploads, challenges and refresh tokens), which runs against both the fake and Postgres (`internal/test/integration/contracts_test.go`). Change a rule → change the contract, and both sides follow.
- One rule, one layer: validators test input rules, use cases test domain rules, handlers test HTTP mapping (one case per error code, plus response shape), integration tests SQL rules. Don't re-test a lower layer's variants from above.
- Frontend API docs live in `docs/api/` (index `docs/README.md`, shared rules + full error code table in `docs/api/conventions.md`). Adding or changing an endpoint, request/response field or error code → update the matching `docs/api/<resource>.md` (and the conventions table for new codes) in the same change.
- Keep test infra light: don't build protocol-level fakes (e.g. fake AMQP server) — prefer live-broker tests with `t.Skip`, or ask first.
