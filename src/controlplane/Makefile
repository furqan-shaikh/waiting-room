.PHONY: tidy build run clean install-migrate check-migrate migrate-up migrate-down migrate-version

# override using: make migrate-up PG_DATABASE_URL='postgres://...'
PG_DATABASE_URL ?= postgres://wr:wr@localhost:5432/waitingroomdb?sslmode=disable
# override using: make migrate-up MIGRATIONS_DIR='..'
MIGRATIONS_DIR ?= dao/migrations

# CGO_ENABLED=0 disables Cgo to produce a statically-linked Go binary, making the executable more portable and container-friendly.
# A statically-linked binary doesn't have run-time dependencies on other libraries, not even the C runtime library (libc)
build:
	CGO_ENABLED=0 go build -o ./bin/waitingroom-controlplane .

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
	migrate -database "$(PG_DATABASE_URL)" -path $(MIGRATIONS_DIR) up

migrate-down: check-migrate
	migrate -database "$(PG_DATABASE_URL)" -path $(MIGRATIONS_DIR) down 1

migrate-version: check-migrate
	migrate -database "$(PG_DATABASE_URL)" -path $(MIGRATIONS_DIR) version
