.PHONY: tidy build run clean install-migrate check-migrate migrate-up migrate-down migrate-version

# override using: make migrate-up POSTGRESQL_URL='postgres://...'
POSTGRESQL_URL ?= postgres://wr:wr@localhost:5432/waitingroomdb?sslmode=disable
# override using: make migrate-up MIGRATIONS_DIR='..'
MIGRATIONS_DIR ?= dao/migrations

build:
	go build -o ./bin/waitingroom-controlplane .

tidy:
	go mod tidy

run:
	go run .

clean:
	rm -f ./bin/waitingroom-controlplane

install-migrate:
	brew install golang-migrate

check-migrate:
	@brew list golang-migrate > /dev/null 2>&1 || (echo "golang-migrate CLI not found. Install with: brew install golang-migrate" && exit 1)

migrate-up: check-migrate
	migrate -database "$(POSTGRESQL_URL)" -path $(MIGRATIONS_DIR) up

migrate-down: check-migrate
	migrate -database "$(POSTGRESQL_URL)" -path $(MIGRATIONS_DIR) down 1

migrate-version: check-migrate
	migrate -database "$(POSTGRESQL_URL)" -path $(MIGRATIONS_DIR) version
