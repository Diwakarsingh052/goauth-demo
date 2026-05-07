# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/web ./cmd/web

# API
FROM alpine:3.19 AS api

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /bin/api .
COPY .env* ./

EXPOSE 8080

CMD ["./api"]

# Web
FROM alpine:3.19 AS web

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /bin/web .
COPY templates/ ./templates/
COPY static/ ./static/
COPY .env* ./

EXPOSE 8081

CMD ["./web"]