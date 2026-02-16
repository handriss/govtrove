FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /jobs ./cmd/jobs

FROM alpine:3.19

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /jobs /jobs

RUN adduser -D -u 1000 appuser
USER appuser

ENTRYPOINT ["/jobs"]
