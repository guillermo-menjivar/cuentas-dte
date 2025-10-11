curl -X POST http://localhost:8080/v1/companies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paredes & Paredes, S.A. de C.V",
    "nit": "0614-300506-101-3",
    "dte_ambiente": "02",
    "nombre_comercial": "Paredes & Paredes, S.A. de C.V",
    "ncr": "172631-3",
    "firmador_username": "06143005061013",
    "firmador_password": "MF7HwttFuZ.*3RY",
    "hc_username": "06143005061013",
    "hc_password": "MF7HwttFuZ.*3RY",
    "cod_actividad": "69100",
    "email": "contact@paredes.com",
    "departamento": "06",
    "municipio": "06.21",
    "telefono": "23232323"
  }' | jq .
