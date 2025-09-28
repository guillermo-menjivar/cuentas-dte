.PHONY: build-cuentas
build-cuentas:
	$(MAKE) -C cuentas build
	$(MAKE) -C cuentas build-linux

.PHONY: clean
clean:
	$(MAKE) -C cuentas clean
	$(MAKE) -C artifacts clean
