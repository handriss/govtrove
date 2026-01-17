FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /ingest ./cmd/ingest

FROM alpine:3.19

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /ingest /ingest

# Security: run as non-root
RUN adduser -D -u 1000 appuser
USER appuser

ENTRYPOINT ["/ingest"]
