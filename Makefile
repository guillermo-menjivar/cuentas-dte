.PHONY: build-cuentas
build-cuentas:
	$(MAKE) -C cuentas build
	$(MAKE) -C cuentas build-linux
