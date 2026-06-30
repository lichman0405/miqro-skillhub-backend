# SkillHub Go Backend Makefile
#
# Targets:
#   test            Run all Go tests
#   test-server     Vet and build the server binary
#   run-server      Build and run the server locally
#   compose-config  Validate docker-compose.yml
#   help            Show this help

.PHONY: test test-server run-server compose-config help

# Run all Go tests across the server module.
test:
	cd server && go test ./...

# Vet and build the skillhub-server binary.
test-server:
	cd server && go vet ./...
	cd server && go build ./cmd/skillhub-server

# Build and run the server locally.
run-server:
	cd server && go run ./cmd/skillhub-server

# Validate docker-compose.yml syntax.
compose-config:
	docker compose config

# Show help text.
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
