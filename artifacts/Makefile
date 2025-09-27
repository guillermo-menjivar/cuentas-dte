.PHONY: configure_server
configure_server:
	bash build/install_docker.sh

.PHONY: clean
clean:
	-docker compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml down 2>/dev/null || true
	rm -rf dte-firmador

.PHONY: stage_firmador
stage_firmador:
	unzip dte-firmador.zip
	rm -rf dte-firmador/servicioFirmadoWindows
	rm -rf dte-firmador/dockerConSSL
	cp certs/Certificado_06143005061013.crt dte-firmador/dockerSinSSL/docker/temp/
	sed -i 's/\t/  /g' dte-firmador/dockerSinSSL/docker/docker-compose.yml
	sed -i 's|"./temp:/uploads|"./temp:/uploads"|' dte-firmador/dockerSinSSL/docker/docker-compose.yml
	sed -i '/volumes:/a\  environment:\n   - JAVA_OPTS=-Xmx512m -Xms256m\n  ulimits:\n    nofile:\n      soft: 65536\n      hard: 65536' dte-firmador/dockerSinSSL/docker/docker-compose.yml

.PHONY: run_firmador
run_firmador: stage_firmador
	docker-compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml up
