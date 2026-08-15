FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o main ./cmd/api

FROM alpine:latest AS prod

RUN adduser -D -u 10001 app && mkdir -p /app/data && chown -R app:app /app

WORKDIR /app

COPY --from=build --chown=app:app /app/main /app/main

USER app

ENV PORT=8080
EXPOSE 8080

CMD ["./main"]
