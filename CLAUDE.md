# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**This repo is a starter kit**, not a product. It is the Go backend skeleton copied into every new web app: HTTP API + RabbitMQ event consumers + cron jobs in one binary (`cmd/server`). Postgres via sqlc (queries) and goose (migrations). Brevo sends transactional email. Firebase Storage holds uploaded files; libvips (cgo, via govips) makes image previews — `brew install vips` before building or testing.

What ships out of the box: email/password signup + signin, email verification, email + password change, account deletion, JWT access tokens with rotating refresh tokens (HttpOnly cookie), event log + RabbitMQ dispatch with dead-lettering, transactional email log, file uploads with previews, user profiles, cron cleanup jobs.

The app name placeholder is `web_app_starter` (module `github.com/pauliusgin/web_app_starter-go`). Changes made here are meant to be generic — nothing domain-specific to one app belongs in the kit.

## Starting a new project from this kit

1. Copy the directory (without `.git`), then `git init` and `git config core.hooksPath .githooks`.
2. Replace every `web_app_starter` with the new app name: `grep -rl --exclude-dir=.git web_app_starter . | xargs sed -i '' 's/web_app_starter/<new_name>/g'`. This covers the module path in `go.mod` + imports, Makefile container/DB/vhost names, the RabbitMQ exchange (`rabbitmq.EventExchange`, `DeadLetterExchange`) and queue names (`consumers.AuthChallengeEmailQueue`), and test fixtures. Then `go build ./... && go test ./...`.
3. `cp .env.example .env` and fill it in (see Config below).
4. `make db-build`, `make mq-build`, create the RabbitMQ vhost named in `MQ_VHOST`, then `make migrate-all`.
5. Add app-specific upload purposes in `internal/domains/uploads/write/models/upload_rules.go` (`profile_picture` and `document_attachment` are the generic defaults), and grow domains under `internal/domains/`.
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

Git hooks live in `.githooks/`:

- pre-commit: `gofmt -w` on staged `.go` files.
- pre-push: `sqlc diff` (generated code must match `sql/`) + `go test ./...`.

`internal/test/integration` starts its own Postgres via testcontainers-go (`internal/test/support/postgrestest`): one container per run, migrations from the embedded `sql/schema` (`schema.Migrations`, goose library), and each test gets a fresh database cloned from the migrated template (`testPostgres.NewTestDatabase(t)`). Without Docker those tests skip; `INTEGRATION_REQUIRED=1` turns the skip into a failure (for CI).

## Config

Env-only (`internal/config`), read from `.env` if present; `.env.example` lists every var. Required: `DATABASE_URL`, `RABBITMQ_URL`, `JWT_SECRET`, `FRONTEND_BASE_URL`, `APP_ENV`, `BREVO_API_KEY`, `BREVO_SENDER_EMAIL`, `BREVO_EMAIL_VERIFICATION_TEMPLATE_ID`, `GOOGLE_FIREBASE_CREDENTIALS` (base64 service account JSON), `GOOGLE_FIREBASE_STORAGE_BUCKET`. `SERVER_HOST` defaults to 0.0.0.0, `SERVER_PORT` to 3000. `config.Load` reports all missing vars at once via `errors.Join`.

## Architecture

Hexagonal layout: domain declares ports, adapters implement them, `cmd/server` wires everything.

- `cmd/server/` — composition root only. `server.go` runs startup as a `startupSteps` struct of funcs (swappable in tests): config → email gateway → DB → broker → consumers → scheduler → router deps → listen → graceful shutdown on SIGINT/SIGTERM. `router.go` (`buildRouterDependencies`), `consumers.go`, `scheduler.go` build the dependency graphs.
- `internal/domains/<domain>/` — use cases + ports. Every client-facing domain (`users`, `profiles`, `uploads`) splits CQRS-style into `write/` (models, `<x>_write_repository.go` port, usecases; also internal lookups such as `FindUserByID`) and `read/` (client-facing read models, `<x>_read_repository.go` port with `Get<X>ReadModelByID`, `queries` such as `GetUserByID`), even while the two models match; internal-only domains (`messaging`) stay flat. Write handlers answer the read model by re-reading through the resource's get-by-id query after the write. Each domain has its own `errors/` package.
- `internal/auth/` — auth primitives as small packages: `challenge` (email/phone verification tokens), `refreshtoken` (issue/rotate/revoke/cleanup), `jwt` (access tokens), `password` (argon2id), `token` (random + hash).
- `internal/events/` — domain events. `Dispatcher.Dispatch` records to `event_log` table and publishes to RabbitMQ; failures are **logged, never returned** (event dispatch never fails a use case). Per-event constructors live in subpackages (`events/user`, `events/auth`, ...) as `NewX` + `DispatchX`. Labels in `event_label_list.go` double as routing keys. Use `RedactedData` when the payload carries a secret (e.g. challenge token) — the log stores the redacted copy, subscribers get the full one.
- `internal/api/http/` — `routes` (one `<resource>_routes.go` per resource exporting `XRoutePath` + `RegisterXRoutes(mux, XRoutesDependencies)`; `BuildRouter` calls each on a single `http.ServeMux` — no nested muxes or trailing-slash prefixes; patterns are full Go 1.22 `"METHOD XRoutePath/sub"` strings; unmatched paths/methods keep the mux's plain-text 404/405 on purpose), `handlers` (one package per resource, laid out as `<usecase>.go` + `<usecase>_test.go` per handler, `validation_rejection_rules.go` for rule sets shared across its handlers, `errors.go`; map domain errors → `{status, code}` rejection via `errors.As` switch; failures answer `handlers.ErrorResponse{code, message, details}` (`details` null when unset) where `code` is a snake_case `handlers.ErrorCode` from the fixed list in `handlers/errorcode.go`; 500s log the cause and answer `internal_error`; collapse distinct rejections into one code when distinct codes would leak account state; absent values answer an explicit `null`, never `""`, zero time or a dropped key — nullable response fields are pointers without `omitempty` (`handlers.NullableString` / `NullableTime` convert domain zero values; `TestResponseShapesNeverOmitEmptyFields` bans `omitempty` under `handlers/`)), `middleware` (`RequireAuthenticatedUser`), `validators` (`primitive` = email/password/string rules, `input` = per-request; `input.ValidateXInput(raw) (XInput, error)` trims/normalizes and returns the cleaned input — handlers never trim or normalize themselves).
- Refresh tokens never appear in a JSON body: signup/signin/refresh set them as the `refresh_token` cookie (`handlers/tokens/refresh_token_cookie.go`: HttpOnly, Secure, SameSite=Strict, `Path` = `routes.TokenRoutePath`, guarded by `routes_test`); refresh and revoke read it from there, and any refresh rejection or revoke clears it. Bodies carry only `accessToken`. The browser client must reach the API same-site (e.g. dev proxy) and send `credentials: "include"` on token routes — no CORS layer ships.
- `internal/api/messaging/` — inbound side of RabbitMQ. `consumers.BuildRegistry` declares every consumed event (label → queue → handler); handlers in `handlers/messaging` translate payloads into domain use case calls. Consumers with nil handlers are skipped with a warning.
- `internal/adapters/` — `repositories/postgres/<domain>/{repositories,mappers}` wrap generated sqlc queries; `brokers/rabbitmq` (topic exchange, publisher confirms, dead-letter topology); `gateways/brevo` (email); `gateways/firebase` (upload file storage); `imaging/libvips` (image inspection + WebP previews). Stripe gateway is a placeholder.
- `internal/scheduler/` — cron wrapper (robfig/cron); jobs registered in `cmd/server/scheduler.go`.

### Dependency injection pattern

No DI container. Every use case is a plain function `func X(ctx, d XDependencies, params XParams) (XResult, error)` — `XParams` (input) → `XResult` (output, a named struct) → `XDependencies` declared at the top of the file; error-only use cases return plain `error`, no Result. Params/Result carry no `json` tags. Callers depend on a **function type**, not the use case directly — `cmd/server` closes over the dependencies:

```go
signUpUser := func(ctx context.Context, params userusecases.SignUpUserParams) (userusecases.SignUpUserResult, error) {
    return userusecases.SignUpUser(ctx, signUpUserDependencies, params)
}
```

Handlers take a `XDependencies` struct of these func types. Each handler file opens with its unexported json-tagged `xRequest` / `xResponse` structs (only where a JSON body exists): the body decodes into `xRequest` — never into use case Params — and the response is built field by field from the Result. Adding a feature = use case + deps struct → wire closure in `cmd/server` → plug into router/registry deps.

### Adding a DB query

1. Migration in `sql/schema/<YYYYMMDDHHMM>_<name>.sql` with `-- +goose Up` / `-- +goose Down`.
2. Query in `sql/queries/<table>_{read,write}.sql` with `-- name: X :one|:many|:exec`.
3. `make sqlc`.
4. Repository method in `internal/adapters/repositories/postgres/<domain>/repositories` + mapper (`sql.Null*` ↔ domain conversions go through `postgres/sqlnull`); translate `sql.ErrNoRows` → domain `ErrXNotFound`, other errors → domain `ErrXFailed{Cause}`. Add `var _ port.Interface = Impl{}` compile-time check.
5. Any rule living in the SQL (ordering, ownership, soft delete, CTE side effects) gets a flow in `internal/test/integration` (write through the use case, read back through the query). If the port has a shipped in-memory fake, add the rule to its contract suite too.

Soft delete: queries filter `deleted_at IS NULL`.

### Read vs write user repos

Write repo (`FindUserByID`, `FindUserByEmail`) serves internal checks (existence, verification state) — including auth middleware. Read repo (`GetUserReadModelByID`) serves client-facing responses only (signup, signin, `GET /users/me`).

## Conventions

- Every package has a one-line package doc; every exported name has a one-line doc comment starting with its name.
- Structured params everywhere: `XParams` / `XDependencies` structs instead of positional args (including in `cmd/server` private helpers).
- Each package has `errors.go` with `Err<What>` struct types (value receiver `Error()`, exported fields carrying context, `Cause` + `Unwrap()` for wrapped errors). Shared message text in package-level `const ...Text`. Match errors with `errors.As`, never strings.
- Logging via `log/slog` with key-value pairs (`"userId", id`, `"error", err`).
- Tests: no mocks. Use in-memory implementations of ports with failure-injection fields (`SaveFailure error`): shipped ones (`InMemoryUserRepository`, `InMemoryProfileRepository`, `InMemoryUploadRepository` — the last two implement both the write and the read port —, `challenge.InMemoryChallengeRepository`, `refreshtoken.InMemoryRefreshTokenRepository`, …) — a fake that mirrors SQL ships, it is not hand-rolled per test package. Wire real use cases over them rather than stubbing the use case. Event capture through `eventstest.CapturedEvents`. Fixed `time.Date(...)` fixtures. `email.InMemoryGateway` is a shipped fake for the email port. Assert on returned values and resulting store state.
- Shipped fakes that mirror SQL rules are held to it by a contract suite (`<x>repositorytest.Run<X>RepositoryContract` for users, profiles, uploads, challenges and refresh tokens), which runs against both the fake and Postgres (`internal/test/integration/contracts_test.go`). Change a rule → change the contract, and both sides follow.
- One rule, one layer: validators test input rules, use cases test domain rules, handlers test HTTP mapping (one case per error code, plus response shape), integration tests SQL rules. Don't re-test a lower layer's variants from above.
- Keep test infra light: don't build protocol-level fakes (e.g. fake AMQP server) — prefer live-broker tests with `t.Skip`, or ask first.
