.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make setup       - Complete initial setup (docker + migrations)"
	@echo "  make run         - Run the Go service"
	@echo "  make dev         - Start everything (docker + service)"
	@echo "  make test-auth   - Test authentication flow"
	@echo "  make test-api    - Test API endpoints"
	@echo "  make logs        - Show all service logs"
	@echo "  make clean       - Stop and clean everything"

# Setup commands
.PHONY: setup
setup: clean docker-up wait-services migrate
	@echo "✅ Setup complete! Run 'make dev' to start the service"

.PHONY: docker-up
docker-up:
	@echo "🚀 Starting Docker services..."
	@docker-compose down 2>/dev/null || true
	@docker-compose up -d

.PHONY: docker-down
docker-down:
	@echo "🛑 Stopping Docker services..."
	@docker-compose down

.PHONY: docker-logs
docker-logs:
	docker-compose logs -f

.PHONY: wait-services
wait-services:
	@echo "⏳ Waiting for services to be ready..."
	@sleep 5
	@until docker exec trading_postgres pg_isready -U trading 2>/dev/null; do \
		echo "Waiting for trading_postgres..."; \
		sleep 2; \
	done
	@echo "✅ PostgreSQL is ready"
	@until docker exec kratos_postgres pg_isready -U kratos 2>/dev/null; do \
		echo "Waiting for kratos_postgres..."; \
		sleep 2; \
	done
	@echo "✅ Kratos PostgreSQL is ready"
	@until curl -s http://localhost:4433/health/ready > /dev/null 2>&1; do \
		echo "Waiting for Kratos..."; \
		sleep 2; \
	done
	@echo "✅ Kratos is ready"
	@until curl -s http://localhost:4455/health > /dev/null 2>&1; do \
		echo "Waiting for Kratos UI..."; \
		sleep 2; \
	done
	@echo "✅ Kratos UI is ready"

# Database commands
.PHONY: migrate
migrate:
	@echo "🔄 Running migrations..."
	@docker exec -i trading_postgres psql -U trading -d trading < migrations/001_initial.sql
	@docker exec -i trading_postgres psql -U trading -d trading < migrations/002_user_preferences.sql
	@echo "✅ Migrations complete"

.PHONY: db-shell
db-shell:
	docker exec -it trading_postgres psql -U trading -d trading

.PHONY: db-kratos-shell
db-kratos-shell:
	docker exec -it kratos_postgres psql -U kratos -d kratos

# Development commands
.PHONY: run
run:
	@echo "🚀 Starting Go service..."
	go run cmd/server/main.go

.PHONY: dev
dev:
	@echo "🚀 Starting development environment..."
	@make docker-up
	@make wait-services
	@make migrate
	@echo "✅ Infrastructure ready! Starting service..."
	@make run

.PHONY: build
build:
	go build -o bin/server cmd/server/main.go

# Testing commands
.PHONY: test-health
test-health:
	@echo "🏥 Testing health endpoints..."
	@echo "\n--- Trading Service Health ---"
	@curl -s http://localhost:8080/health | jq || echo "Service not running"
	@echo "\n--- Trading Service Ready ---"
	@curl -s http://localhost:8080/ready | jq || echo "Service not ready"
	@echo "\n--- Kratos Health ---"
	@curl -s http://localhost:4433/health/ready | jq || echo "Kratos not ready"
	@echo "\n--- Kratos UI Health ---"
	@curl -s http://localhost:4455/health | jq || echo "Kratos UI not ready"

.PHONY: test-auth
test-auth:
	@echo "🔐 Testing authentication flow..."
	@echo "\n--- Kratos Health ---"
	@curl -s http://localhost:4433/health/ready | jq
	@echo "\n--- Create Login Flow ---"
	@curl -s http://localhost:4433/self-service/login/browser | jq '.ui.action'
	@echo "\n--- Kratos UI Login Page ---"
	@echo "Open in browser: http://localhost:4455/login"
	@echo "\n--- List Identities (Admin) ---"
	@curl -s http://localhost:4434/admin/identities | jq

.PHONY: test-api
test-api:
	@echo "📊 Testing API endpoints..."
	@echo "\n⚠️  You need to login first via http://localhost:4455/login"
	@echo "\nAfter login, the session cookie will be set automatically"
	@echo "\nThen you can test protected endpoints:"
	@echo "  curl http://localhost:8080/auth/me"

.PHONY: test-session
test-session:
	@echo "🔍 Testing current session..."
	@echo "\n--- Who Am I (needs valid session) ---"
	@curl -s -b cookies.txt -c cookies.txt http://localhost:4433/sessions/whoami | jq || echo "No active session"

# Logs and debugging
.PHONY: logs
logs:
	@echo "📜 Showing all logs (Ctrl+C to exit)..."
	docker-compose logs -f

.PHONY: logs-kratos
logs-kratos:
	docker logs -f kratos

.PHONY: logs-kratos-ui
logs-kratos-ui:
	docker logs -f kratos_ui

.PHONY: logs-postgres
logs-postgres:
	docker logs -f trading_postgres

# Cleanup
.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	rm -rf bin/
	rm -f cookies.txt

.PHONY: reset
reset: clean setup
	@echo "♻️  Full reset complete!"

# Utilities
.PHONY: check-ports
check-ports:
	@echo "🔍 Checking ports..."
	@lsof -i :8080 || echo "✅ Port 8080 is free (Go service)"
	@lsof -i :5433 || echo "✅ Port 5433 is free (Trading PostgreSQL)"
	@lsof -i :5434 || echo "✅ Port 5434 is free (Kratos PostgreSQL)"
	@lsof -i :4433 || echo "✅ Port 4433 is free (Kratos Public)"
	@lsof -i :4434 || echo "✅ Port 4434 is free (Kratos Admin)"
	@lsof -i :4455 || echo "✅ Port 4455 is free (Kratos UI)"
	@lsof -i :8000 || echo "✅ Port 8000 is free (Frontend)"

.PHONY: open-ui
open-ui:
	@echo "🌐 Opening UIs in browser..."
	@open http://localhost:8000 || xdg-open http://localhost:8000 || echo "Frontend: http://localhost:8000"
	@open http://localhost:4455 || xdg-open http://localhost:4455 || echo "Kratos UI: http://localhost:4455"

.PHONY: create-test-data
create-test-data:
	@echo "📝 Creating test CSV file..."
	@echo "Symbol,Date,Open,High,Low,Close,Volume" > test_data.csv
	@echo "BBCA.JK,2025-01-06,8500,8600,8450,8550,12500000" >> test_data.csv
	@echo "BBCA.JK,2025-01-07,8550,8650,8500,8600,13000000" >> test_data.csv
	@echo "BBRI.JK,2025-01-06,4500,4600,4450,4550,25000000" >> test_data.csv
	@echo "BBRI.JK,2025-01-07,4550,4650,4500,4600,28000000" >> test_data.csv
	@echo "✅ Created test_data.csv"

# Generate secrets for Kratos
.PHONY: generate-secrets
generate-secrets:
	@echo "🔐 Generating secrets for Kratos..."
	@echo "SECRETS_COOKIE=$$(openssl rand -hex 16)"
	@echo "SECRETS_CIPHER=$$(openssl rand -hex 16)"
	@echo "COOKIE_SECRET=$$(openssl rand -hex 16)"
	@echo "\n📋 Copy these to your docker-compose.yml or .env file"