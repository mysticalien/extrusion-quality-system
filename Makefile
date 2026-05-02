ifneq (,$(wildcard .env))
include .env
export
endif

APP_NAME := extrusion-quality-system

COMPOSE := docker compose
MIGRATIONS_DIR := migrations

POSTGRES_CONTAINER ?= $(or $(POSTGRES_CONTAINER_NAME),extrusion_postgres)

KAFKA_CONTAINER ?= $(or $(KAFKA_CONTAINER_NAME),extrusion_kafka)
KAFKA_TOPIC ?= $(or $(KAFKA_TELEMETRY_TOPIC),extrusion.telemetry.raw)

# For commands executed inside Docker network / Kafka container.
KAFKA_CONTAINER_BOOTSTRAP ?= kafka:29092

SERVER_PACKAGE := ./cmd/server
SIMULATOR_PACKAGE := ./cmd/simulator

.PHONY: help
help:
	@echo ""
	@echo "Available commands:"
	@echo ""
	@echo "Docker Compose:"
	@echo "  make compose-up              Build and start full system in foreground"
	@echo "  make compose-up-detached     Build and start full system in background"
	@echo "  make compose-down            Stop compose services"
	@echo "  make compose-clean           Stop compose services and remove volumes"
	@echo "  make compose-logs            Show compose logs"
	@echo "  make compose-ps              Show compose services status"
	@echo ""
	@echo "Legacy docker aliases:"
	@echo "  make docker-up               Start docker services"
	@echo "  make docker-down             Stop docker services"
	@echo "  make docker-restart          Restart docker services"
	@echo "  make docker-logs             Show docker logs"
	@echo "  make docker-clean            Stop docker services and remove volumes"
	@echo ""
	@echo "Database:"
	@echo "  make db-psql                 Open psql inside postgres container"
	@echo "  make db-tables               Show database tables"
	@echo "  make db-quality-weights      Show quality weights"
	@echo "  make db-reset                Reset database volumes and apply migrations"
	@echo ""
	@echo "Migrations:"
	@echo "  make goose-install           Install goose"
	@echo "  make migrate-up              Apply all migrations"
	@echo "  make migrate-down            Rollback last migration"
	@echo "  make migrate-status          Show migration status"
	@echo "  make migrate-redo            Rollback and re-apply last migration"
	@echo "  make migrate-reset           Rollback all migrations"
	@echo ""
	@echo "Kafka:"
	@echo "  make kafka-create-topic      Create telemetry Kafka topic"
	@echo "  make kafka-topics            List Kafka topics"
	@echo "  make kafka-describe-topic    Describe telemetry Kafka topic"
	@echo "  make kafka-consume           Consume telemetry topic from beginning"
	@echo "  make kafka-consume-latest    Consume only new telemetry messages"
	@echo ""
	@echo "Go:"
	@echo "  make fmt                     Format Go code"
	@echo "  make test                    Run tests"
	@echo "  make build                   Build server and simulator"
	@echo "  make check                   Format, test and build"
	@echo "  make run-server              Run backend server locally"
	@echo "  make run-simulator           Run simulator locally"
	@echo ""

.PHONY: require-database-url
require-database-url:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "DATABASE_URL is not set. Create .env from .env.example and fill local values." >&2; \
		exit 1; \
	fi

.PHONY: compose-up
compose-up:
	$(COMPOSE) up --build

.PHONY: compose-up-detached
compose-up-detached:
	$(COMPOSE) up --build -d

.PHONY: compose-down
compose-down:
	$(COMPOSE) down

.PHONY: compose-clean
compose-clean:
	$(COMPOSE) down -v

.PHONY: compose-logs
compose-logs:
	$(COMPOSE) logs -f

.PHONY: compose-ps
compose-ps:
	$(COMPOSE) ps

.PHONY: docker-up
docker-up:
	$(COMPOSE) up -d

.PHONY: docker-down
docker-down:
	$(COMPOSE) down

.PHONY: docker-restart
docker-restart:
	$(COMPOSE) down
	$(COMPOSE) up -d

.PHONY: docker-logs
docker-logs:
	$(COMPOSE) logs -f

.PHONY: docker-clean
docker-clean:
	$(COMPOSE) down -v

.PHONY: db-psql
db-psql:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

.PHONY: db-tables
db-tables:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "\dt"'

.PHONY: db-quality-weights
db-quality-weights:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "SELECT id, parameter, weight, updated_by FROM quality_weights ORDER BY parameter;"'

.PHONY: db-reset
db-reset: require-database-url
	$(COMPOSE) down -v
	$(COMPOSE) up -d postgres
	@echo "Waiting for postgres..."
	@sleep 5
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

.PHONY: goose-install
goose-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest

.PHONY: migrate-up
migrate-up: require-database-url
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: require-database-url
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

.PHONY: migrate-status
migrate-status: require-database-url
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

.PHONY: migrate-redo
migrate-redo: require-database-url
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" redo

.PHONY: migrate-reset
migrate-reset: require-database-url
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" reset

.PHONY: kafka-topics
kafka-topics:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--list

.PHONY: kafka-create-topic
kafka-create-topic:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--create \
		--if-not-exists \
		--topic $(KAFKA_TOPIC) \
		--partitions 3 \
		--replication-factor 1

.PHONY: kafka-consume
kafka-consume:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-console-consumer.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--topic $(KAFKA_TOPIC) \
		--from-beginning

.PHONY: kafka-consume-latest
kafka-consume-latest:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-console-consumer.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--topic $(KAFKA_TOPIC)

.PHONY: kafka-describe-topic
kafka-describe-topic:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--describe \
		--topic $(KAFKA_TOPIC)

.PHONY: fmt
fmt:
	gofmt -w cmd internal

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build $(SERVER_PACKAGE)
	go build $(SIMULATOR_PACKAGE)

.PHONY: check
check: fmt test build

.PHONY: run-server
run-server:
	CONFIG_PATH=.env go run $(SERVER_PACKAGE)

.PHONY: run-simulator
run-simulator:
	CONFIG_PATH=.env go run $(SIMULATOR_PACKAGE)