APP_NAME := extrusion-quality-system

COMPOSE := docker compose
MIGRATIONS_DIR := migrations

POSTGRES_CONTAINER ?= extrusion_postgres

SERVER_PACKAGE := ./cmd/server
SIMULATOR_PACKAGE := ./cmd/simulator

.PHONY: help
help:
	@echo ""
	@echo "Available commands:"
	@echo ""
	@echo "  make docker-up              Start docker services"
	@echo "  make docker-down            Stop docker services"
	@echo "  make docker-restart         Restart docker services"
	@echo "  make docker-logs            Show docker logs"
	@echo "  make db-psql                Open psql inside postgres container"
	@echo "  make db-tables              Show database tables"
	@echo "  make db-quality-weights     Show quality weights"
	@echo ""
	@echo "  make goose-install          Install goose"
	@echo "  make migrate-up             Apply all migrations"
	@echo "  make migrate-down           Rollback last migration"
	@echo "  make migrate-status         Show migration status"
	@echo "  make migrate-redo           Rollback and re-apply last migration"
	@echo "  make migrate-reset          Rollback all migrations"
	@echo ""
	@echo "  make fmt                    Format Go code"
	@echo "  make test                   Run tests"
	@echo "  make build                  Build server and simulator"
	@echo "  make check                  Format, test and build"
	@echo ""
	@echo "  make run-server             Run backend server"
	@echo "  make run-simulator          Run simulator"
	@echo ""

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

.PHONY: db-psql
db-psql:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

.PHONY: db-tables
db-tables:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "\dt"'

.PHONY: db-quality-weights
db-quality-weights:
	docker exec -it $(POSTGRES_CONTAINER) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "SELECT id, parameter, weight, updated_by FROM quality_weights ORDER BY parameter;"'

.PHONY: goose-install
goose-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest

.PHONY: migrate-up
migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

.PHONY: migrate-status
migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

.PHONY: migrate-redo
migrate-redo:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" redo

.PHONY: migrate-reset
migrate-reset:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" reset

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
	go run $(SERVER_PACKAGE)

.PHONY: run-simulator
run-simulator:
	go run $(SIMULATOR_PACKAGE)

.PHONY: docker-clean
docker-clean:
	docker compose down -v

.PHONY: db-reset
db-reset:
	docker compose down -v
	docker compose up -d
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

KAFKA_CONTAINER ?= extrusion_kafka
KAFKA_TOPIC ?= extrusion.telemetry.raw

# Для команд внутри Docker-сети / контейнера Kafka.
KAFKA_CONTAINER_BOOTSTRAP ?= kafka:29092

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

.PHONY: kafka-describe-topic
kafka-describe-topic:
	docker exec -it $(KAFKA_CONTAINER) /opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server $(KAFKA_CONTAINER_BOOTSTRAP) \
		--describe \
		--topic $(KAFKA_TOPIC)