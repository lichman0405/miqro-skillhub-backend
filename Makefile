# SkillHub Go Backend Makefile
#
# Targets:
#   test            Run all Go tests
#   test-server     Vet and build all server binaries
#   openapi         Validate server/openapi/openapi.yaml structure
#   coverage        Run tests with cross-package coverage profile
#   run-server      Build and run the server locally
#   compose-config  Validate docker-compose.yml
#   db-reset        Reset and re-apply database migrations (requires PostgreSQL)
#   help            Show this help

.PHONY: test test-server openapi coverage run-server compose-config db-reset help

# Run all Go tests across the server module.
test:
	cd server && go test ./...

# Vet and build the skillhub-server binary.
test-server:
	cd server && go vet ./...
	cd server && go build ./cmd/skillhub-server

# Validate that openapi.yaml is valid YAML and contains all required
# OpenAPI 3.0 structural sections plus the frontend read-model routes.
openapi:
	cd server && go test ./openapi/ -v -run TestOpenAPISpec

# Run tests with cross-package coverage (more realistic than per-package default).
# This is a thin wrapper around direct Go commands; Windows users can run the
# equivalent commands without make.
coverage:
	cd server && go test -coverpkg=./... -coverprofile=coverage.out ./...
	cd server && go tool cover -func=coverage.out

# Build and run the server locally.
run-server:
	cd server && go run ./cmd/skillhub-server

# Validate docker-compose.yml syntax.
compose-config:
	docker compose config

# Reset and re-apply database migrations (requires running PostgreSQL).
db-reset:
	cd server && go run ./cmd/skillhub-migrate reset

# Show help text.
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
