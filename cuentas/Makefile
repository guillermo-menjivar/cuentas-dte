.PHONY: build
build:
	go mod tidy
	go build -o bin/cuentas main.go

.PHONY: build-linux
build-linux:
	go mod tidy
	GOOS=linux GOARCH=amd64 go build -o bin/cuentas-linux main.go
