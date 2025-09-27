
.PHONY: clean
clean:
	rm -rf dte-firmador

.PHONY: stage_firmador
stage_firmador:
	unzip dte-firmador.zip
	rm -rf dte-firmador/servicioFirmadoWindows
	rm -rf dte-firmador/dockerConSSL
	cp certs/Certificado_06143005061013.crt dte-firmador/dockerSinSSL/docker/temp/

.PHONY: run_firmador
run_firmador: stage_firmador
	docker-compose -f dte-firmador/dockerSinSSL/docker/docker-compose.yml up -d
