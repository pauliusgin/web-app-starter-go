# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**This repo is a starter kit**, not a product. It is the Go backend skeleton copied into every new web app: HTTP API + RabbitMQ event consumers + cron jobs in one binary (`cmd/server`). Postgres via sqlc (queries) and goose (migrations). Brevo sends transactional email. Firebase Storage holds uploaded files; libvips (cgo, via govips) makes image previews — `brew install vips` before building or testing.

What ships out of the box: email/password signup + signin, email verification, email + password change, account deletion, JWT access tokens with rotating refresh tokens, event log + RabbitMQ dispatch with dead-lettering, transactional email log, file uploads with previews, user profiles, cron cleanup jobs.

The app name placeholder is `web_app_starter` (module `github.com/pauliusgin/web_app_starter-go`). Changes made here are meant to be generic — nothing domain-specific to one app belongs in the kit.

## Starting a new project from this kit

1. Copy the directory (without `.git`), then `git init` and `git config core.hooksPath .githooks`.
2. Replace every `web_app_starter` with the new app name: `grep -rl --exclude-dir=.git web_app_starter . | xargs sed -i '' 's/web_app_starter/<new_name>/g'`. This covers the module path in `go.mod` + imports, Makefile container/DB/vhost names, the RabbitMQ exchange (`rabbitmq.EventExchange`, `DeadLetterExchange`) and queue names (`consumers.AuthChallengeEmailQueue`), and test fixtures. Then `go build ./... && go test ./...`.
3. `cp .env.example .env` and fill it in (see Config below).
4. `make db-build`, `make mq-build`, create the RabbitMQ vhost named in `MQ_VHOST`, then `make migrate-all`.
5. Add app-specific upload purposes in `internal/domains/uploads/models/upload_rules.go` (`profile_picture` and `document_attachment` are the generic defaults), and grow domains under `internal/domains/`.
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
make test                         # go test ./...
go test ./internal/auth/refreshtoken -run TestRotateRefreshToken   # single package / test

make sqlc                         # regenerate internal/adapters/repositories/postgres/sqlc from sql/
export GOOSE_DBSTRING=<db url>    # required before migrate-* targets
make migrate-all | migrate-one | migrate-down
```

Git hooks live in `.githooks/`:

- pre-commit: `gofmt -w` on staged `.go` files.
- pre-push: `sqlc diff` (generated code must match `sql/`) + `go test ./...`.

## Config

Env-only (`internal/config`), read from `.env` if present; `.env.example` lists every var. Required: `DATABASE_URL`, `RABBITMQ_URL`, `JWT_SECRET`, `FRONTEND_BASE_URL`, `APP_ENV`, `BREVO_API_KEY`, `BREVO_SENDER_EMAIL`, `BREVO_EMAIL_VERIFICATION_TEMPLATE_ID`, `GOOGLE_FIREBASE_CREDENTIALS` (base64 service account JSON), `GOOGLE_FIREBASE_STORAGE_BUCKET`. `SERVER_HOST` defaults to 0.0.0.0, `SERVER_PORT` to 3000. `config.Load` reports all missing vars at once via `errors.Join`.

## Architecture

Hexagonal layout: domain declares ports, adapters implement them, `cmd/server` wires everything.

- `cmd/server/` — composition root only. `server.go` runs startup as a `startupSteps` struct of funcs (swappable in tests): config → email gateway → DB → broker → consumers → scheduler → router deps → listen → graceful shutdown on SIGINT/SIGTERM. `router.go` (`buildRouterDependencies`), `consumers.go`, `scheduler.go` build the dependency graphs.
- `internal/domains/<domain>/` — use cases + ports. Shipped domains: `users` (split CQRS-style into `write/` and `read/`), `profiles`, `uploads`, `messaging`. Each domain has its own `errors/` package.
- `internal/auth/` — auth primitives as small packages: `challenge` (email/phone verification tokens), `refreshtoken` (issue/rotate/revoke/cleanup), `jwt` (access tokens), `password` (argon2id), `token` (random + hash).
- `internal/events/` — domain events. `Dispatcher.Dispatch` records to `event_log` table and publishes to RabbitMQ; failures are **logged, never returned** (event dispatch never fails a use case). Per-event constructors live in subpackages (`events/user`, `events/auth`, ...) as `NewX` + `DispatchX`. Labels in `event_label_list.go` double as routing keys. Use `RedactedData` when the payload carries a secret (e.g. challenge token) — the log stores the redacted copy, subscribers get the full one.
- `internal/api/http/` — `routes` (one sub-router per prefix, mounted in `BuildRouter`; Go 1.22 `"METHOD /path"` patterns on `http.ServeMux`; unmatched paths/methods keep the mux's plain-text 404/405 on purpose), `handlers` (one package per resource; map domain errors → `{status, code}` rejection via `errors.As` switch; failures answer `handlers.ErrorResponse{code, message, details}` (`details` null when unset) where `code` is a snake_case `handlers.ErrorCode` from the fixed list in `handlers/errorcode.go`; 500s log the cause and answer `internal_error`; collapse distinct rejections into one code when distinct codes would leak account state; absent values answer an explicit `null`, never `""`, zero time or a dropped key — nullable response fields are pointers without `omitempty` (`handlers.NullableString` / `NullableTime` convert domain zero values; `TestResponseShapesNeverOmitEmptyFields` bans `omitempty` under `handlers/`)), `middleware` (`RequireAuthenticatedUser`), `validators` (`primitive` = email/password/string rules, `input` = per-request).
- `internal/api/messaging/` — inbound side of RabbitMQ. `consumers.BuildRegistry` declares every consumed event (label → queue → handler); handlers in `handlers/messaging` translate payloads into domain use case calls. Consumers with nil handlers are skipped with a warning.
- `internal/adapters/` — `repositories/postgres/<domain>/{repositories,mappers}` wrap generated sqlc queries; `brokers/rabbitmq` (topic exchange, publisher confirms, dead-letter topology); `gateways/brevo` (email); `gateways/firebase` (upload file storage); `imaging/libvips` (image inspection + WebP previews). Stripe gateway is a placeholder.
- `internal/scheduler/` — cron wrapper (robfig/cron); jobs registered in `cmd/server/scheduler.go`.

### Dependency injection pattern

No DI container. Every use case is a plain function `func X(ctx, d XDependencies, params XParams) (XResult, error)` — `XParams` (input) → `XResult` (output, a named struct) → `XDependencies` declared at the top of the file; error-only use cases return plain `error`, no Result. Params/Result carry no `json` tags. Callers depend on a **function type**, not the use case directly — `cmd/server` closes over the dependencies:

```go
signupUser := func(ctx context.Context, params usecases.SignupUserParams) (usecases.SignupUserResult, error) {
    return usecases.SignupUser(ctx, signupUserDependencies, params)
}
```

Handlers take a `XDependencies` struct of these func types. Each handler file opens with its unexported json-tagged `xRequest` / `xResponse` structs (only where a JSON body exists): the body decodes into `xRequest` — never into use case Params — and the response is built field by field from the Result. Adding a feature = use case + deps struct → wire closure in `cmd/server` → plug into router/registry deps.

### Adding a DB query

1. Migration in `sql/schema/<YYYYMMDDHHMM>_<name>.sql` with `-- +goose Up` / `-- +goose Down`.
2. Query in `sql/queries/<table>_{read,write}.sql` with `-- name: X :one|:many|:exec`.
3. `make sqlc`.
4. Repository method in `internal/adapters/repositories/postgres/<domain>/repositories` + mapper (`sql.Null*` ↔ domain conversions go through `postgres/sqlnull`); translate `sql.ErrNoRows` → domain `ErrXNotFound`, other errors → domain `ErrXFailed{Cause}`. Add `var _ port.Interface = Impl{}` compile-time check.

Soft delete: queries filter `deleted_at IS NULL`.

### Read vs write user repos

Write repo (`FindUserByID`, `FindUserByEmail`) serves internal checks (existence, verification state) — including auth middleware. Read repo (`FindUserReadModelByID`) is reserved for client-facing responses and is intentionally unwired for now; don't delete it as dead code.

## Conventions

- Every package has a one-line package doc; every exported name has a one-line doc comment starting with its name.
- Structured params everywhere: `XParams` / `XDependencies` structs instead of positional args (including in `cmd/server` private helpers).
- Each package has `errors.go` with `Err<What>` struct types (value receiver `Error()`, exported fields carrying context, `Cause` + `Unwrap()` for wrapped errors). Shared message text in package-level `const ...Text`. Match errors with `errors.As`, never strings.
- Logging via `log/slog` with key-value pairs (`"userId", id`, `"error", err`).
- Tests: no mocks. Use in-memory implementations of ports (e.g. `inMemoryChallengeStore` in `internal/auth/challenge/in_memory_repository_test.go`, shipped `InMemory*` repositories/gateways in domain packages) with failure-injection fields (`saveFailure error`). Fixed `time.Date(...)` fixtures. `email.InMemoryGateway` is a shipped fake for the email port. Assert on returned values and resulting store state.
- Keep test infra light: don't build protocol-level fakes (e.g. fake AMQP server) — prefer live-broker tests with `t.Skip`, or ask first.
