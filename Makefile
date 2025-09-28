.PHONY: build-cuentas
build:
	$(MAKE) -C cuentas build
	$(MAKE) -C cuentas build-linux
