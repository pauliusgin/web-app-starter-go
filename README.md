🚀🚀🚀

## Commands

```sh
make db-build / db-start          # Postgres 18 in Docker
make mq-build / mq-start          # RabbitMQ 4 + management UI on :15672

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
