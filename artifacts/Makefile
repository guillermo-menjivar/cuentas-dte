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

.phony: stage_firmador
stage_firmador:
	unzip dte-firmador.zip
	rm -rf dte-firmador/serviciofirmadowindows
	rm -rf dte-firmador/dockerconssl
	cp certs/06143005061013.crt dte-firmador/dockersinssl/docker/temp/06143005061013.crt
	cp certs/PublicteKey_06143005061013.key dte-firmador/dockerSinSSL/docker/temp/PublicteKey_06143005061013.key
	cp build/custom_docker-compose.yml dte-firmador/dockerSinSSL/docker/docker-compose.yml
