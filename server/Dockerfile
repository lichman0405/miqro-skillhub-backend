# Multi-stage build for the SkillHub Go backend.
#
# Stage 1: build the binary.
# Stage 2: minimal runtime image.

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/skillhub-server ./cmd/skillhub-server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/skillhub-server /usr/local/bin/skillhub-server
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["skillhub-server"]
