curl -X POST http://localhost:8080/v1/companies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Paredes & Paredes, S.A. de C.V",
    "nit": "0614-300506-101-3",
    "ncr": "172631-3",
    "hc_username": "06143005061013",
    "hc_password": "MF7HwttFuZ.*3RY",
    "email": "contact@paredes.com"
  }' | jq .
