.PHONY: generate clean help

# Generate sqlc code from queries and migrations
generate:
	@echo "🔄 Generating sqlc code..."
	@go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
	@echo "✅ Generation complete!"

# Clean generated files
clean:
	@echo "🧹 Cleaning sqlc generated files..."
	@rm -rf ./entities/ 2>/dev/null || true
	@echo "✅ Cleanup complete! (removed entities/ directory)"

# Clean and regenerate
reset: clean generate

# Start local development infrastructure
dev-up:
	@echo "🚀 Starting local development infrastructure..."
	@docker compose -f ./.dev/docker-compose_local.yaml --env-file ./.dev/.env -p openteam up -d
	@echo "✅ Docker containers started in detached mode"


# Stop and remove local development infrastructure
dev-down:
	@echo "🛑 Stopping local development infrastructure..."
	@docker compose -f ./.dev/docker-compose_local.yaml --env-file ./.dev/.env -p openteam down
	@echo "✅ Docker containers stopped and removed"

# Show help
help:
	@echo "Available targets:"
	@echo "  generate    - Generate sqlc code from schemas"
	@echo "  clean       - Remove all generated files"
	@echo "  reset       - Clean and regenerate (fresh start)"
	@echo "  dev-up      - Start local development infrastructure"
	@echo "  dev-down    - Stop and remove local development infrastructure"
	@echo "  help        - Show this help message"

