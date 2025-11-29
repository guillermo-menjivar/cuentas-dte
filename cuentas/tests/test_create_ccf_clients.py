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


def get_existing_clients(company_id: str) -> List[Dict]:
    """Get list of existing clients"""
    headers = {"X-Company-ID": company_id}
    try:
        response = requests.get(f"{BASE_URL}/clients", headers=headers)
        response.raise_for_status()
        clients = response.json().get("clients", [])
        # Ensure we always return a list, never None
        return clients if clients is not None else []
    except Exception as e:
        print(f"⚠️  Warning: Could not fetch existing clients: {e}")
        return []  # Always return empty list on error


def client_exists(nit: str, existing_clients: List[Dict]) -> bool:
    """Check if client with given NIT already exists"""
    # Safety check: ensure existing_clients is not None
    if not existing_clients:
        return False

    for client in existing_clients:
        if client.get("nit") == nit:
            return True
    return False


def create_ccf_clients(company_id: str):
    """Create CCF test clients"""
    headers = {"Content-Type": "application/json", "X-Company-ID": company_id}

    print("=" * 60)
    print("🏢 CREATING CCF (CRÉDITO FISCAL) TEST CLIENTS")
    print("=" * 60)
    print(f"Company ID: {company_id}")
    print(f"Total clients to create: {len(CCF_CLIENTS)}\n")

    # Get existing clients to check for duplicates
    print("🔍 Checking for existing clients...")
    existing_clients = get_existing_clients(company_id)

    if existing_clients:
        print(f"✅ Found {len(existing_clients)} existing client(s)\n")
    else:
        print(f"✅ No existing clients found\n")

    created = 0
    skipped = 0
    failed = 0

    for i, client_data in enumerate(CCF_CLIENTS, 1):
        print(f"[{i}/{len(CCF_CLIENTS)}] Creating: {client_data['business_name']}")

        # Check if client already exists
        if client_exists(client_data["nit"], existing_clients):
            print(f"  ⚠️  Already exists (NIT: {client_data['nit']})")
            print(f"  → Skipping...")
            skipped += 1
            print()
            continue

        try:
            response = requests.post(
                f"{BASE_URL}/clients", headers=headers, json=client_data
            )
            response.raise_for_status()
            result = response.json()
            print(f"  ✅ Created: {result.get('id')}")
            print(f"     NIT: {client_data['nit']}")
            print(f"     NCR: {client_data['ncr']}")
            print(f"     Email: {client_data['correo']}")
            created += 1
        except requests.exceptions.HTTPError as e:
            # Check if it's a duplicate error (in case race condition)
            if e.response.status_code == 409:
                print(f"  ⚠️  Already exists (detected during creation)")
                print(f"  → Skipping...")
                skipped += 1
            else:
                print(f"  ❌ Failed: {e.response.status_code}")
                try:
                    error_detail = e.response.json()
                    print(f"     Error: {error_detail.get('error', e.response.text)}")
                except:
                    print(f"     Error: {e.response.text}")
                failed += 1
        except Exception as e:
            print(f"  ❌ Failed: {e}")
            failed += 1

        print()

    print("=" * 60)
    print("📊 SUMMARY")
    print("=" * 60)
    print(f"✅ Successfully created: {created}")
    print(f"⚠️  Already existed (skipped): {skipped}")
    print(f"❌ Failed: {failed}")
    print("=" * 60)

    # Exit with success if no hard failures
    if failed > 0:
        sys.exit(1)
    else:
        sys.exit(0)


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
