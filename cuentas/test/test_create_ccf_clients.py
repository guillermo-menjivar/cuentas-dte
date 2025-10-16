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
        "nit": "0614-100200-001-3",
        "ncr": "100200-1",
        "business_name": "Distribuidora El Salvador S.A. de C.V.",
        "legal_business_name": "Distribuidora El Salvador Sociedad Anonima de Capital Variable",
        "giro": "Distribucion de productos alimenticios",
        "tipo_contribuyente": "Gran Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "46390",
        "desc_actividad": "Comercio al por mayor de productos alimenticios, bebidas y tabaco",
        "nombre_comercial": "Distribuidora El Salvador",
        "telefono": "22501111",
        "correo": "ventas@distelsalvador.com",
        "full_address": "Boulevard Constitucion, Edificio Comercial, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0614-200300-102-4",
        "ncr": "200300-2",
        "business_name": "Constructora Moderna S.A.",
        "legal_business_name": "Constructora Moderna Sociedad Anonima",
        "giro": "Construccion de edificios residenciales y no residenciales",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "41000",
        "desc_actividad": "Construccion de edificios",
        "nombre_comercial": "Constructora Moderna",
        "telefono": "22502222",
        "correo": "info@constmoderna.com",
        "full_address": "Avenida Revolucion, Torre Empresarial, Piso 3, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0614-300400-001-5",
        "ncr": "300400-3",
        "business_name": "Tecnologia e Innovacion S.A. de C.V.",
        "legal_business_name": "Tecnologia e Innovacion Sociedad Anonima de Capital Variable",
        "giro": "Servicios de tecnologia de la informacion",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "62010",
        "desc_actividad": "Actividades de programacion informatica",
        "nombre_comercial": "TechInova",
        "telefono": "22503333",
        "correo": "contacto@techinova.sv",
        "full_address": "Centro Comercial Galerias, Torre B, Oficina 501, Antiguo Cuscatlan",
        "country_code": "SV",
        "department_code": "05",
        "municipality_code": "23",
    },
    {
        "nit": "0614-400500-103-6",
        "ncr": "400500-4",
        "business_name": "Servicios Profesionales Consultores S.A.",
        "legal_business_name": "Servicios Profesionales Consultores Sociedad Anonima",
        "giro": "Actividades de consultoria de gestion empresarial",
        "tipo_contribuyente": "Pequeño Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "70200",
        "desc_actividad": "Actividades de consultoria de gestion empresarial",
        "nombre_comercial": "SPConsultores",
        "telefono": "22504444",
        "correo": "info@spconsultores.com",
        "full_address": "Colonia Escalon, Calle Reforma, Edificio Profesional, San Salvador",
        "country_code": "SV",
        "department_code": "06",
        "municipality_code": "23",
    },
    {
        "nit": "0214-500600-001-7",
        "ncr": "500600-5",
        "business_name": "Comercial Santa Ana S.A. de C.V.",
        "legal_business_name": "Comercial Santa Ana Sociedad Anonima de Capital Variable",
        "giro": "Venta al por mayor de materiales de construccion",
        "tipo_contribuyente": "Mediano Contribuyente",
        "tipo_persona": "2",
        "cod_actividad": "46740",
        "desc_actividad": "Comercio al por mayor de ferreteria, fontaneria y calefaccion",
        "nombre_comercial": "Comercial Santa Ana",
        "telefono": "24405555",
        "correo": "ventas@comsantaana.com",
        "full_address": "Carretera Panamericana KM 65, Santa Ana",
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
