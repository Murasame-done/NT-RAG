# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/go-service ./cmd/server

FROM alpine:3.22

WORKDIR /app
RUN apk add --no-cache ca-certificates

RUN adduser -D -H -u 10001 appuser

COPY --from=builder /out/go-service /app/go-service

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/go-service"]
