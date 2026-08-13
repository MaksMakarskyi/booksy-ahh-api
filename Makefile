.PHONY: run test format tidy compose migrate migrate-down migrate-status migrate-reset

GOOSE = set -a && . ./.env && set +a && mkdir -p data && goose

run:
	@go run cmd/api/main.go

test:
	@go test ./...

format:
	@go fmt ./...

tidy:
	@go mod tidy

compose:
	@docker compose up --build

migrate:
	@$(GOOSE) up

migrate-down:
	@$(GOOSE) down

migrate-status:
	@$(GOOSE) status

# Drops everything and re-applies. Destroys data — local use only.
migrate-reset:
	@$(GOOSE) reset && $(GOOSE) up
