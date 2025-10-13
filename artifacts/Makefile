.phony: configure_server
configure_server:
	bash build/install_docker.sh

.phony: clean
clean:
	docker-compose -f dte-firmador/dockersinssl/docker/docker-compose.yml down 2>/dev/null || true
	rm -rf dte-firmador

.phony: run_firmador
run_firmador: stage_firmador
	docker-compose -f dte-firmador/dockersinssl/docker/docker-compose.yml up

.PHONY: stage_firmador
stage_firmador:
	unzip -o dte-firmador.zip
	rm -rf dte-firmador/serviciofirmadowindows
	rm -rf dte-firmador/dockerconssl
	cp certs/firmador_certs/06143005061013.crt dte-firmador/dockerSinSSL/docker/temp/06143005061013.crt
	cp build/custom_docker-compose.yml dte-firmador/dockerSinSSL/docker/docker-compose.yml
