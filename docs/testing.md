## Testing

`make test` (`go test ./...`) runs everything. The pre-push hook runs it too, after `sqlc diff`.

### Layers

Each rule is tested once, at the layer that owns it:

| Layer         | Where                                     | Tests                                                                |
| ------------- | ----------------------------------------- | -------------------------------------------------------------------- |
| Validators    | `internal/api/http/validators`            | input rules (trimming, formats, required fields)                     |
| Use cases     | `internal/domains/**`, `internal/auth/**` | domain rules, against in-memory repositories                         |
| HTTP handlers | `internal/api/http/handlers`              | status + error code per rejection, response shape (explicit `null`s) |
| Integration   | `internal/test/integration`               | rules living in SQL, use cases over real Postgres                    |

No mocks: tests use in-memory port implementations (with failure-injection fields such as `SaveFailure`) and assert on returned values and stored state.

### Integration suite

- `internal/test/integration` starts one `postgres:18` container per run via testcontainers, migrates a template database from `sql/schema`, and clones a fresh database per test.
- `internal/test/support/postgrestest` holds that container setup; any package's tests can import it.
- Contract suites (`RunUserRepositoryContract`, `RunProfileRepositoryContract`, `RunUploadRepositoryContract`, …) run against both the in-memory fake and Postgres, so the fake can't drift from the SQL.

### Tests that need external services

These skip when the service is missing:

| Tests                                                         | Needs                                                                     |
| ------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `internal/test/integration`                                   | Docker running (`INTEGRATION_REQUIRED=1` turns the skip into a failure)   |
| `cmd/server/server_test.go`                                   | `make db-start`, `make mq-start`                                          |
| `internal/adapters/brokers/rabbitmq/connection_test.go`       | `make mq-start`                                                           |
| `internal/adapters/gateways/firebase/storage_gateway_test.go` | `GOOGLE_FIREBASE_CREDENTIALS` + `GOOGLE_FIREBASE_STORAGE_BUCKET` exported |

```sh
go test ./internal/test/integration/                                  # integration suite only
go test ./internal/auth/refreshtoken -run TestRotateRefreshToken       # single test
```
