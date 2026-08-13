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