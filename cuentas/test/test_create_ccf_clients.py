#!/usr/bin/env python3
"""
Create test clients for Crédito Fiscal (CCF) testing
Usage: python create_ccf_clients.py <company_id>
"""

import sys
import requests
from typing import Dict, List

BASE_URL = "http://localhost:8080/v1"

# Sample CCF clients (Persona Jurídica - tipo_persona = "2")
CCF_CLIENTS = [
    {
        "nit": "0614-230591-130-6",
        "ncr": "322800-4",
        "business_name": "Rustico Bolengo",
        "legal_business_name": "Comercial Santa Ana Sociedad Anonima de Capital Variable",
        "giro": "Venta al por mayor de materiales de construccion",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "56101",
        "nombre_comercial": "Comercial Santa Ana",
        "telefono": "24405555",
        "correo": "ventas@comsantaana.com",
        "full_address": "Carretera Panamericana KM 65, Santa Ana",
        "country_code": "SV",
        "department_code": "02",
        "municipality_code": "14",
    },
]


def create_ccf_clients(company_id: str):
    """Create CCF test clients"""
    headers = {"Content-Type": "application/json", "X-Company-ID": company_id}

    print("=" * 60)
    print("🏢 CREATING CCF (CRÉDITO FISCAL) TEST CLIENTS")
    print("=" * 60)
    print(f"Company ID: {company_id}")
    print(f"Total clients to create: {len(CCF_CLIENTS)}\n")

    created = 0
    failed = 0

    for i, client_data in enumerate(CCF_CLIENTS, 1):
        print(f"[{i}/{len(CCF_CLIENTS)}] Creating: {client_data['business_name']}")

        try:
            response = requests.post(
                f"{BASE_URL}/clients", headers=headers, json=client_data
            )
            response.raise_for_status()
            result = response.json()

            print(f"  ✅ Created: {result.get('id')}")
            print(f"  NIT: {client_data['nit']}")
            print(f"  NCR: {client_data['ncr']}")
            print(f"  Email: {client_data['correo']}")
            created += 1

        except requests.exceptions.HTTPError as e:
            print(f"  ❌ Failed: {e.response.status_code}")
            print(f"  Error: {e.response.text}")
            failed += 1
        except Exception as e:
            print(f"  ❌ Failed: {e}")
            failed += 1

        print()

    print("=" * 60)
    print("📊 SUMMARY")
    print("=" * 60)
    print(f"Successfully created: {created}")
    print(f"Failed: {failed}")
    print("=" * 60)


def main():
    if len(sys.argv) < 2:
        print("Usage: python create_ccf_clients.py <company_id>")
        print(
            "Example: python create_ccf_clients.py bda93a7d-45dd-4d62-823f-4213806ff68f"
        )
        sys.exit(1)

    company_id = sys.argv[1]
    create_ccf_clients(company_id)


if __name__ == "__main__":
    main()
