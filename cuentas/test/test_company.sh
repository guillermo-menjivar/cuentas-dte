curl -X POST http://localhost:8080/v1/companies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "nit": 12345678901234,
    "ncr": 98765432109876,
    "hc_username": "acme_user",
    "hc_password": "super_secret_password_123",
    "email": "contact@acme.com"
  }' 
