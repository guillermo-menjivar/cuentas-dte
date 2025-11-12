.PHONY: build
build:
	go mod tidy
	go build -o bin/cuentas main.go

.PHONY: build-linux
build-linux:
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o bin/cuentas-linux main.go

.PHONY: install
install: build
	sudo cp bin/cuentas /usr/local/bin/

.PHONY: run
run:
	go run main.go

.PHONY: test
test:
	go test ./...

.PHONY: docker-build
docker-build:
	docker build -t cuentas:latest .

.PHONY: docker-up
docker-up:
	docker-compose build --no-cache
	docker-compose up

.PHONY: docker-down
docker-down:
	docker-compose down -v

.PHONY: docker-logs
docker-logs:
	docker-compose logs -f

.PHONY: docker-rebuild
docker-rebuild:
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d

.PHONY: migrate-up
migrate-up:
	docker-compose exec cuentas ./cuentas migrate up

.PHONY: migrate-down
migrate-down:
	docker-compose exec cuentas ./cuentas migrate down

.PHONY: migrate-status
migrate-status:
	docker-compose exec cuentas ./cuentas migrate status

.PHONY: vault-init
vault-init:
	docker-compose exec cuentas ./cuentas vault-init

.PHONY: vault-status
vault-status:
	docker-compose exec vault vault status

.PHONY: redis-cli
redis-cli:
	docker-compose exec redis redis-cli -a redis_password

.PHONY: vault-cli
vault-cli:
	docker-compose exec vault vault auth -method=token token=vault-root-token

.PHONY: setup-all
setup-all: docker-up vault-init migrate-up
	@echo "Full setup completed: Vault initialized, database migrated"

.PHONY: clean
clean:
	rm -rf bin/
	docker-compose down -v --rmi all --remove-orphans
	docker system prune -f

.PHONY: db-connect
db-connect:
	docker exec -it cuentas-postgres psql -U cuentas_user -d cuentas

.PHONY: refresh
refresh:
	@echo "🔄 Refreshing cuentas application..."
	@echo "1. Rebuilding cuentas container..."
	docker-compose build cuentas
	@echo "2. Stopping cuentas container..."
	docker-compose stop cuentas
	@echo "3. Reinitializing Vault secrets..."
	docker-compose up -d vault-init
	@echo "4. Waiting for vault-init to complete..."
	@sleep 5
	@echo "5. Starting cuentas container..."
	docker-compose up -d cuentas
	@echo "6. Showing logs..."
	docker-compose logs -f cuentas

.PHONY: refresh-full
refresh-full:
	@echo "🔄 Full refresh (rebuilding all services)..."
	docker-compose down
	docker-compose up --build -d
	@echo "✅ All services restarted"
	docker-compose logs -f cuentas

.PHONY: vault-reinit
vault-reinit:
	@echo "🔐 Reinitializing Vault secrets..."
	docker-compose up -d vault-init
	@echo "✅ Vault reinitialized"

.PHONY: logs
logs:
	docker-compose logs -f cuentas
