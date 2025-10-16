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
        "nit": "0614-710820-001-9",
        "ncr": "710820-7",
        "business_name": "Importadora Pacific Trade S.A. de C.V.",
        "legal_business_name": "Importadora Pacific Trade Sociedad Anonima de Capital Variable",
        "giro": "Importacion y distribucion de equipos electronicos",
        "tipo_contribuyente": "Gran Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "46510",
        "desc_actividad": "Comercio al por mayor de computadoras y equipo periferico",
        "nombre_comercial": "Pacific Trade",
        "telefono": "22506666",
        "correo": "ventas@pacifictrade.com.sv",
        "full_address": "Zona Industrial, Bodega 15, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0614-820930-102-1",
        "ncr": "820930-8",
        "business_name": "Logistica Express S.A.",
        "legal_business_name": "Logistica Express Sociedad Anonima",
        "giro": "Servicios de transporte y almacenamiento",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "49410",
        "desc_actividad": "Transporte de carga por carretera",
        "nombre_comercial": "Logistica Express",
        "telefono": "22507777",
        "correo": "operaciones@logexpress.sv",
        "full_address": "Carretera al Puerto, KM 12, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0514-930140-001-2",
        "ncr": "930140-9",
        "business_name": "Soluciones Digitales La Libertad S.A. de C.V.",
        "legal_business_name": "Soluciones Digitales La Libertad Sociedad Anonima de Capital Variable",
        "giro": "Desarrollo de software empresarial",
        "tipo_contribuyente": "Pequeño Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "62020",
        "desc_actividad": "Actividades de consultoria de informatica",
        "nombre_comercial": "SolDigital",
        "telefono": "22508888",
        "correo": "info@soldigital.com.sv",
        "full_address": "Plaza Merliot, Edificio 3, Oficina 205, Santa Tecla",
        "country_code": "SV",
        "department_code": "05",
        "municipality_code": "23",
    },
    {
        "nit": "0614-140250-103-3",
        "ncr": "140250-1",
        "business_name": "Farmaceutica Central S.A.",
        "legal_business_name": "Farmaceutica Central Sociedad Anonima",
        "giro": "Distribucion de productos farmaceuticos",
        "tipo_contribuyente": "Gran Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "46460",
        "desc_actividad": "Comercio al por mayor de productos farmaceuticos",
        "nombre_comercial": "FarmaCentral",
        "telefono": "22509999",
        "correo": "ventas@farmacentral.com",
        "full_address": "Boulevard Los Heroes, Centro Medico, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0214-250360-001-4",
        "ncr": "250360-2",
        "business_name": "Agroindustrial Santa Ana S.A. de C.V.",
        "legal_business_name": "Agroindustrial Santa Ana Sociedad Anonima de Capital Variable",
        "giro": "Procesamiento y distribucion de productos agricolas",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "46210",
        "desc_actividad": "Comercio al por mayor de granos y semillas",
        "nombre_comercial": "AgroSantaAna",
        "telefono": "24401111",
        "correo": "ventas@agrosantaana.sv",
        "full_address": "Zona Agricola, Bodega Central, Santa Ana",
        "country_code": "SV",
        "department_code": "02",
        "municipality_code": "16",
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
