.PHONY: help
help:
	@echo "🚀 Proto Trading Service - Production Ready"
	@echo ""
	@echo "📋 Available commands:"
	@echo "  make setup       - Complete initial setup (docker + build + migrations)"
	@echo "  make dev         - Start development environment"
	@echo "  make prod        - Start production environment"
	@echo "  make build       - Build Go service Docker image"
	@echo "  make test-auth   - Test authentication flow"
	@echo "  make test-api    - Test API endpoints with authentication"
	@echo "  make logs        - Show all service logs"
	@echo "  make clean       - Stop and clean everything"

# Setup commands
.PHONY: setup
setup: clean build docker-up wait-services migrate
	@echo "✅ Setup complete! Run 'make prod' to start all services"

.PHONY: dev
dev: build docker-up wait-services migrate
	@echo "🚀 Development environment started!"
	@echo ""
	@echo "📊 Services:"
	@echo "  Frontend:      http://localhost:8000"
	@echo "  Trading API:   http://localhost:8080"
	@echo "  Kratos UI:     http://localhost:4455"
	@echo "  Kratos API:    http://localhost:4433"
	@echo ""
	@echo "🔐 Authentication Flow:"
	@echo "  1. Open frontend: http://localhost:8000"
	@echo "  2. Click login → redirects to http://localhost:4455/login"
	@echo "  3. Login with Google OAuth"
	@echo "  4. Redirected back to frontend with session"
	@echo "  5. Frontend can now call API with session cookie"

.PHONY: prod
prod: build
	@echo "🚀 Starting production environment..."
	@docker-compose -f docker-compose.yml up -d
	@make wait-services
	@echo "✅ Production environment started!"

# Build commands
.PHONY: build
build:
	@echo "🔨 Building Go service Docker image..."
	@docker build -t proto-trading-service:latest .
	@echo "✅ Build complete"

.PHONY: docker-up
docker-up:
	@echo "🐳 Starting Docker services..."
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
	@echo "Checking PostgreSQL (trading)..."
	@until docker exec trading_postgres pg_isready -U trading 2>/dev/null; do \
		echo "  ⏳ Waiting for trading_postgres..."; \
		sleep 2; \
	done
	@echo "  ✅ Trading PostgreSQL is ready"
	@echo "Checking PostgreSQL (kratos)..."
	@until docker exec kratos_postgres pg_isready -U kratos 2>/dev/null; do \
		echo "  ⏳ Waiting for kratos_postgres..."; \
		sleep 2; \
	done
	@echo "  ✅ Kratos PostgreSQL is ready"
	@echo "Checking Kratos API..."
	@until curl -s http://localhost:4433/health/ready > /dev/null 2>&1; do \
		echo "  ⏳ Waiting for Kratos API..."; \
		sleep 2; \
	done
	@echo "  ✅ Kratos API is ready"
	@echo "Checking Kratos UI..."
	@until curl -s http://localhost:4455/health > /dev/null 2>&1; do \
		echo "  ⏳ Waiting for Kratos UI..."; \
		sleep 2; \
	done
	@echo "  ✅ Kratos UI is ready"
	@echo "Checking Trading Service..."
	@until curl -s http://localhost:8080/health > /dev/null 2>&1; do \
		echo "  ⏳ Waiting for Trading Service..."; \
		sleep 2; \
	done
	@echo "  ✅ Trading Service is ready"
	@echo "🎉 All services are ready!"

# Database commands
.PHONY: migrate
migrate:
	@echo "🔄 Running migrations..."
	@docker exec -i trading_postgres psql -U trading -d trading < migrations/001_initial.sql 2>/dev/null || echo "Migration 1 already applied"
	@docker exec -i trading_postgres psql -U trading -d trading < migrations/002_user_preferences.sql 2>/dev/null || echo "Migration 2 already applied"
	@echo "✅ Migrations complete"

.PHONY: db-shell
db-shell:
	@echo "🐘 Opening trading database shell..."
	docker exec -it trading_postgres psql -U trading -d trading

.PHONY: db-kratos-shell
db-kratos-shell:
	@echo "🐘 Opening kratos database shell..."
	docker exec -it kratos_postgres psql -U kratos -d kratos

.PHONY: db-reset
db-reset:
	@echo "🗑️ Resetting databases..."
	@docker-compose down -v
	@docker-compose up -d postgres postgres-kratos
	@make wait-services
	@make migrate
	@echo "✅ Databases reset complete"

# Testing commands
.PHONY: test-health
test-health:
	@echo "🏥 Testing service health..."
	@echo "\n--- Trading Service Health ---"
	@curl -s http://localhost:8080/health | jq || echo "❌ Service not running"
	@echo "\n--- Trading Service Ready ---"
	@curl -s http://localhost:8080/ready | jq || echo "❌ Service not ready"
	@echo "\n--- Kratos Health ---"
	@curl -s http://localhost:4433/health/ready | jq || echo "❌ Kratos not ready"
	@echo "\n--- Kratos UI Health ---"
	@curl -s http://localhost:4455/health | jq || echo "❌ Kratos UI not ready"

.PHONY: test-auth
test-auth:
	@echo "🔐 Testing authentication flow..."
	@echo "\n🔍 Step 1: Check auth status (should be unauthenticated)"
	@curl -s http://localhost:8080/auth/status | jq
	@echo "\n🔗 Step 2: Get login URL"
	@curl -s http://localhost:8080/auth/login-url | jq
	@echo "\n📋 Step 3: Manual login required"
	@echo "  1. Open: http://localhost:4455/login"
	@echo "  2. Login with Google OAuth"
	@echo "  3. Copy session cookie from browser"
	@echo "  4. Run: make test-session COOKIE='your-session-cookie'"

.PHONY: test-session
test-session:
	@echo "🍪 Testing with session cookie..."
	@echo "Usage: make test-session COOKIE='your-ory-kratos-session-value'"
	@if [ -z "$(COOKIE)" ]; then \
		echo "❌ No COOKIE provided"; \
		echo "Get session cookie from browser after login"; \
		exit 1; \
	fi
	@echo "\n--- Testing /auth/me ---"
	@curl -s -H "Cookie: ory_kratos_session=$(COOKIE)" http://localhost:8080/auth/me | jq
	@echo "\n--- Testing protected API ---"
	@curl -s -H "Cookie: ory_kratos_session=$(COOKIE)" http://localhost:8080/api/v1/market-data?symbol=BBCA.JK | jq

.PHONY: test-api
test-api:
	@echo "📊 Testing API endpoints..."
	@echo "\n⚠️  You need to login first via http://localhost:4455/login"
	@echo "\nAfter login, test with your session cookie:"
	@echo "  make test-session COOKIE='your-session-cookie-value'"
	@echo "\nOr test with browser (session cookie automatic):"
	@echo "  GET  http://localhost:8080/auth/status"
	@echo "  GET  http://localhost:8080/auth/me"
	@echo "  GET  http://localhost:8080/api/v1/market-data?symbol=BBCA.JK"

.PHONY: test-cors
test-cors:
	@echo "🌐 Testing CORS configuration..."
	@echo "\n--- Testing from localhost:8000 origin ---"
	@curl -s -H "Origin: http://localhost:8000" \
		-H "Access-Control-Request-Method: GET" \
		-H "Access-Control-Request-Headers: Content-Type,Authorization" \
		-X OPTIONS http://localhost:8080/api/v1/market-data | grep -i "access-control" || echo "No CORS headers"
	@echo "\n--- Testing from localhost:4455 origin ---"
	@curl -s -H "Origin: http://localhost:4455" \
		-H "Access-Control-Request-Method: POST" \
		-H "Access-Control-Request-Headers: Content-Type,Cookie" \
		-X OPTIONS http://localhost:8080/auth/me | grep -i "access-control" || echo "No CORS headers"

# Development commands
.PHONY: logs
logs:
	@echo "📜 Showing all logs (Ctrl+C to exit)..."
	docker-compose logs -f

.PHONY: logs-trading
logs-trading:
	@echo "📜 Trading service logs..."
	docker logs -f trading_service

.PHONY: logs-kratos
logs-kratos:
	@echo "📜 Kratos logs..."
	docker logs -f kratos

.PHONY: logs-kratos-ui
logs-kratos-ui:
	@echo "📜 Kratos UI logs..."
	docker logs -f kratos_ui

.PHONY: restart-trading
restart-trading:
	@echo "🔄 Restarting trading service..."
	@docker-compose restart trading-service
	@echo "✅ Trading service restarted"

.PHONY: rebuild-trading
rebuild-trading:
	@echo "🔨 Rebuilding and restarting trading service..."
	@make build
	@docker-compose up -d --no-deps trading-service
	@echo "✅ Trading service rebuilt and restarted"

# Data commands
.PHONY: create-test-data
create-test-data:
	@echo "📝 Creating test data files..."
	@echo "Symbol,Date,Open,High,Low,Close,Volume" > test_data.csv
	@echo "BBCA.JK,2025-01-06,8500,8600,8450,8550,12500000" >> test_data.csv
	@echo "BBCA.JK,2025-01-07,8550,8650,8500,8600,13000000" >> test_data.csv
	@echo "BBRI.JK,2025-01-06,4500,4600,4450,4550,25000000" >> test_data.csv
	@echo "BBRI.JK,2025-01-07,4550,4650,4500,4600,28000000" >> test_data.csv
	@echo "TLKM.JK,2025-01-06,3200,3250,3180,3220,18000000" >> test_data.csv
	@echo "TLKM.JK,2025-01-07,3220,3280,3200,3260,19000000" >> test_data.csv
	@echo "✅ Created test_data.csv"

.PHONY: upload-test-data
upload-test-data:
	@echo "📤 Uploading test data (requires authentication)..."
	@if [ ! -f test_data.csv ]; then make create-test-data; fi
	@echo "You need to upload via browser or with session cookie:"
	@echo "  curl -X POST -F 'file=@test_data.csv' \\"
	@echo "    -H 'Cookie: ory_kratos_session=YOUR_SESSION' \\"
	@echo "    http://localhost:8080/api/v1/upload/csv"

# Cleanup commands
.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	@docker-compose down -v
	@docker rmi proto-trading-service:latest 2>/dev/null || true
	@rm -rf bin/
	@rm -f cookies.txt test_data.csv
	@echo "✅ Cleanup complete"

.PHONY: reset
reset: clean setup
	@echo "♻️  Full reset complete!"

# Utility commands
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
	@open http://localhost:8080/health || xdg-open http://localhost:8080/health || echo "API Health: http://localhost:8080/health"

.PHONY: status
status:
	@echo "📊 Service Status:"
	@echo "===================="
	@docker-compose ps
	@echo ""
	@echo "🔗 Service URLs:"
	@echo "  Frontend:     http://localhost:8000"
	@echo "  Trading API:  http://localhost:8080"
	@echo "  Kratos UI:    http://localhost:4455"
	@echo "  Kratos API:   http://localhost:4433"
	@echo ""
	@echo "🔐 Authentication:"
	@echo "  Login:        http://localhost:4455/login"
	@echo "  Registration: http://localhost:4455/registration"

# Generate secrets for production
.PHONY: generate-secrets
generate-secrets:
	@echo "🔐 Generating secrets for production..."
	@echo "SECRETS_COOKIE=$(openssl rand -hex 16)"
	@echo "SECRETS_CIPHER=$(openssl rand -hex 16)"
	@echo "COOKIE_SECRET=$(openssl rand -hex 16)"
	@echo ""
	@echo "📋 Copy these to your docker-compose.yml environment variables"

# Documentation
.PHONY: docs
docs:
	@echo "📚 Proto Trading Service Documentation"
	@echo "====================================="
	@echo ""
	@echo "🏗️  Architecture:"
	@echo "  Browser → Frontend (localhost:8000)"
	@echo "     ↓"
	@echo "  Kratos UI (localhost:4455) ← Login/Register"
	@echo "     ↓"
	@echo "  Kratos API (localhost:4433) ← Session management"
	@echo "     ↓"
	@echo "  Trading API (localhost:8080) ← Market data & user preferences"
	@echo ""
	@echo "🔄 Authentication Flow:"
	@echo "  1. User visits frontend"
	@echo "  2. Frontend detects no session → redirects to Kratos UI"
	@echo "  3. User logs in via Google OAuth"
	@echo "  4. Kratos sets session cookie → redirects to frontend"
	@echo "  5. Frontend calls Trading API with session cookie"
	@echo "  6. Trading API validates session with Kratos"
	@echo ""
	@echo "🔧 Development Commands:"
	@echo "  make setup    - First time setup"
	@echo "  make dev      - Start development"
	@echo "  make prod     - Start production"
	@echo "  make test-auth - Test authentication"
	@echo "  make logs     - View logs"
	@echo "  make clean    - Clean everything"