.PHONY: configure_server
configure_server:
	bash build/install_docker.sh


.PHONY: clean
clean:
	docker compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml down
	rm -rf dte-firmador

.PHONY: stage_firmador
stage_firmador:
	unzip dte-firmador.zip
	rm -rf dte-firmador/servicioFirmadoWindows
	rm -rf dte-firmador/dockerConSSL
	cp certs/Certificado_06143005061013.crt dte-firmador/dockerSinSSL/docker/temp/
	sed -i '/volumes:/a\  environment:\   - JAVA_OPTS=-Xmx512m -Xms256m\  ulimits:\    nofile:\      soft: 65536\      hard: 65536' dte-firmador/dockerSinSSL/docker/docker-compose.yml

.PHONY: run_firmador
run_firmador: stage_firmador
	docker-compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml up
