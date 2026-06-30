# Multi-stage build for the SkillHub Go backend.
#
# Phase 01: minimal image that builds and runs the health-only server.

FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/skillhub-server ./cmd/skillhub-server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/skillhub-server /usr/local/bin/skillhub-server
EXPOSE 8080
ENTRYPOINT ["skillhub-server"]
