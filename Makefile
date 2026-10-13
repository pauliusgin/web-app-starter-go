DB_CONTAINER_NAME=web_app_starter-app
DB_IMAGE=postgres:18
DB_NAME=web_app_starter
DB_USER=user
DB_PASSWORD=password
DB_PORT=5432
DB_HOST=localhost

DB_URL=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslrequire=true

MQ_CONTAINER_NAME=web_app_starter-app-mq
MQ_IMAGE=rabbitmq:4-management
MQ_USER=user
MQ_PASSWORD=password
MQ_PORT=5672
MQ_MANAGEMENT_PORT=15672
MQ_HOST=localhost
MQ_VHOST=web_app_starter-dev

MQ_URL=amqp://$(MQ_USER):$(MQ_PASSWORD)@$(MQ_HOST):$(MQ_PORT)/$(MQ_VHOST)


# build local database
db-build:
	@echo "Building PostgreSQL Docker container..."
	@docker run --name $(DB_CONTAINER_NAME) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-p $(DB_PORT):5432 \
		-d $(DB_IMAGE)
	@echo "PostgreSQL container $(DB_CONTAINER_NAME) running on port $(DB_PORT)."
	@echo "POSTGRES_USER=$(DB_USER)"
	@echo "POSTGRES_PASSWORD=$(DB_PASSWORD)"
	@echo "POSTGRES_HOST=$(DB_HOST)"
	@echo "DB_PORT $(DB_PORT):5432"
	@echo "DB_NAME=$(DB_NAME)"

db-start:
	@echo "Starting the container..."
	@docker start $(DB_CONTAINER_NAME) || true
	@echo "PostgreSQL container $(DB_CONTAINER_NAME) started."

db-stop:
	@echo "Stopping the container..."
	@docker stop $(DB_CONTAINER_NAME) || true
	@echo "PostgreSQL container $(DB_CONTAINER_NAME) stopped."

db-remove:
	@echo "Removing the container..."
	@docker rm $(DB_CONTAINER_NAME) || true
	@echo "PostgreSQL container $(DB_CONTAINER_NAME) removed."

# build local message broker
mq-build:
	@echo "Building RabbitMQ Docker container..."
	@docker run --name $(MQ_CONTAINER_NAME) \
		-e RABBITMQ_DEFAULT_USER=$(MQ_USER) \
		-e RABBITMQ_DEFAULT_PASS=$(MQ_PASSWORD) \
		-e RABBITMQ_DEFAULT_VHOST=$(MQ_VHOST) \
		-p $(MQ_PORT):5672 \
		-p $(MQ_MANAGEMENT_PORT):15672 \
		-d $(MQ_IMAGE)
	@echo "RabbitMQ container $(MQ_CONTAINER_NAME) running on port $(MQ_PORT)."
	@echo "RABBITMQ_DEFAULT_USER=$(MQ_USER)"
	@echo "RABBITMQ_DEFAULT_PASS=$(MQ_PASSWORD)"
	@echo "RABBITMQ_HOST=$(MQ_HOST)"
	@echo "MQ_PORT $(MQ_PORT):5672"
	@echo "MQ_MANAGEMENT_PORT $(MQ_MANAGEMENT_PORT):15672"
	@echo "MQ_VHOST=$(MQ_VHOST)"
	@echo "MQ_URL=$(MQ_URL)"

# create the vhost on an already running broker
mq-vhost:
	@echo "Creating vhost $(MQ_VHOST)..."
	@docker exec $(MQ_CONTAINER_NAME) rabbitmqctl add_vhost $(MQ_VHOST) || true
	@docker exec $(MQ_CONTAINER_NAME) rabbitmqctl set_permissions -p $(MQ_VHOST) $(MQ_USER) ".*" ".*" ".*"
	@echo "Vhost $(MQ_VHOST) ready for user $(MQ_USER)."

mq-start:
	@echo "Starting the container..."
	@docker start $(MQ_CONTAINER_NAME) || true
	@echo "RabbitMQ container $(MQ_CONTAINER_NAME) started."

mq-stop:
	@echo "Stopping the container..."
	@docker stop $(MQ_CONTAINER_NAME) || true
	@echo "RabbitMQ container $(MQ_CONTAINER_NAME) stopped."

mq-remove:
	@echo "Removing the container..."
	@docker rm $(MQ_CONTAINER_NAME) || true
	@echo "RabbitMQ container $(MQ_CONTAINER_NAME) removed."

mq-reset-queues:
	@echo "Deleting consumer queues..."
	@docker exec $(MQ_CONTAINER_NAME) rabbitmqctl delete_queue -p $(MQ_VHOST) web_app_starter.auth_challenge.email || true
	@echo "Consumer queues deleted; the server redeclares them on start."

# generate sqlc queries
sqlc:
	sqlc generate

# Before running goose commands, set:
# export GOOSE_DBSTRING=<db_connection_string>

# run all migrations
migrate-all:
	goose -dir sql/schema postgres up

# run one migration up
migrate-one:
	goose -dir sql/schema postgres up-by-one

# roll back one migration
migrate-down:
	goose -dir sql/schema postgres down
	
# build an executable
build:
	go build -o ./bin/server ./cmd/server

# run the project for development
run:
	go run ./cmd/server

# run all the tests
test:
	go test ./...