.PHONY: configure_server
configure_server:
	bash build/install_docker.sh

.PHONY: clean
clean:
	-docker compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml down 2>/dev/null || true
	rm -rf dte-firmador

.PHONY: run_firmador
run_firmador: stage_firmador
	docker-compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml up

.PHONY: stage_firmador
stage_firmador:
	unzip dte-firmador.zip
	rm -rf dte-firmador/servicioFirmadoWindows
	rm -rf dte-firmador/dockerConSSL
	cp certs/Certificado_06143005061013.crt dte-firmador/dockerSinSSL/docker/temp/
	cp build/custom_docker-compose.yml dte-firmador/dockerSinSSL/docker/docker-compose.yml
