.PHONY: build
build:
	go mod tidy
	go build -o bin/cuentas main.go

.PHONY: build-linux
build-linux:
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o bin/cuentas-linux main.go

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: migrate-up
migrate-up:
	docker-compose exec cuentas ./cuentas migrate up

.PHONY: migrate-down
migrate-down:
	docker-compose exec cuentas ./cuentas migrate down

.PHONY: migrate-status
migrate-status:
	docker-compose exec cuentas ./cuentas migrate status
