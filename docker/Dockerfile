# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bot ./cmd/bot

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/bot /app/bot

ENTRYPOINT ["/app/bot"]
