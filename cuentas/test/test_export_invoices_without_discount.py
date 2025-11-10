#!/usr/bin/env python3
"""
Automated Export Invoice (Type 11) Creation and Finalization - NO DISCOUNTS VERSION
Creates realistic export invoices WITHOUT discounts to test calculation issues
"""

import requests
import random
import sys
from datetime import datetime, timedelta
from typing import List, Dict, Optional


class ExportInvoiceSeeder:
    """Creates and finalizes export invoices (Type 11) to test export functionality"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}
        self.invoices_created = 0
        self.invoices_finalized = 0
        self.errors = 0

    # Country codes for common export destinations (ONLY VALID CODES FROM SCHEMA)
    EXPORT_COUNTRIES = [
        ("9450", "Estados Unidos", "37", "123 Main Street, Miami, FL 33101"),
        ("9450", "Estados Unidos", "37", "456 Broadway, New York, NY 10013"),
        ("9450", "Estados Unidos", "37", "789 Market St, San Francisco, CA 94102"),
        ("9411", "Costa Rica", "37", "San José Centro, Costa Rica"),
        ("9447", "España", "37", "Calle Gran Vía 123, Madrid, España"),
    ]

    # Recinto fiscal options (Tax Enclosures - most common)
    RECINTO_FISCAL = [
        "01",  # Terrestre San Bartolo
        "02",  # Marítima de Acajutla
        "03",  # Aérea De Comalapa
    ]

    # ✅ FIXED: Regimen options (CAT-028) - Most common export regimes
    REGIMEN = [
        "EX-1.1000.000",  # Exportación Definitiva, Régimen Común
        "EX-1.1040.000",  # Exportación Definitiva Sustitución de Mercancías
        "EX-1.1400.000",  # Exportación Definitiva Courier
    ]

    # ✅ FIXED: INCOTERMS with proper codes (01-11)
    INCOTERMS = [
        ("09", "FOB-Libre a bordo"),  # Most common
        ("11", "CIF- Costo seguro y flete"),  # Most common
        ("10", "CFR-Costo y flete"),
        ("01", "EXW-En fabrica"),
        ("02", "FCA-Libre transportista"),
        ("07", "DDP-Entrega con impuestos pagados"),
    ]

    # ✅ Export document types - Type 2 (Receptor/Customs) is most common
    EXPORT_DOCUMENTS = [
        {
            "cod_doc_asociado": 2,  # Receptor (customs documents)
            "desc_documento": "DUA-2025-{num:06d}",
            "detalle_documento": "Declaración Única Aduanera - Aduana La Hachadura",
        },
    ]

    def get_clients(self) -> List[Dict]:
        """Get all clients"""
        print("🔍 Fetching clients...")
        url = f"{self.base_url}/v1/clients"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            data = response.json()
            clients = data.get("clients", []) if data else []
            if clients is None:
                clients = []
            print(f"✅ Found {len(clients)} clients\n")
            return clients
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get clients: {e}")
            return []

    def get_establishments(self) -> List[Dict]:
        """Get all establishments"""
        print("🔍 Fetching establishments...")
        url = f"{self.base_url}/v1/establishments"
        params = {"active_only": "true"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            data = response.json()
            establishments = data.get("establishments", []) if data else []
            if establishments is None:
                establishments = []
            print(f"✅ Found {len(establishments)} establishments\n")
            return establishments
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get establishments: {e}")
            return []

    def get_points_of_sale(self, establishment_id: str) -> List[Dict]:
        """Get points of sale for an establishment"""
        url = f"{self.base_url}/v1/establishments/{establishment_id}/pos"
        params = {"active_only": "true"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            data = response.json()
            pos = data.get("points_of_sale", []) if data else []
            return pos if pos is not None else []
        except requests.exceptions.RequestException:
            return []

    def get_inventory_items(self) -> List[Dict]:
        """Get all inventory items marked for export (0% IVA / tributo C3)"""
        print("🔍 Fetching export inventory items...")
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            data = response.json()
            items = data.get("items", []) if data else []
            if items is None:
                items = []

            # Filter for items that have tributo C3 (0% export IVA)
            export_items = []
            for item in items:
                # Check if item has export tax code
                # You may need to fetch item details or taxes separately
                export_items.append(item)  # For now, assume all can be exported

            print(f"✅ Found {len(export_items)} export-eligible items\n")
            return export_items
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get inventory: {e}")
            return []

    def get_inventory_state(self, item_id: str) -> Optional[Dict]:
        """Get current inventory state"""
        url = f"{self.base_url}/v1/inventory/items/{item_id}/state"
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException:
            return None

    def create_export_client(self, invoice_num: int) -> Dict:
        """Generate export client data (international)"""
        country = random.choice(self.EXPORT_COUNTRIES)
        cod_pais, nombre_pais, tipo_doc, address = country

        # Generate random document number (9 or 14 digits only for tipo 36)
        doc_number = f"{random.randint(100000000, 999999999)}"  # 9 digits

        # Generate company name based on country
        if "Estados Unidos" in nombre_pais:
            company_name = random.choice(
                [
                    "ABC International Trading Corp",
                    "Global Imports LLC",
                    "Pacific Trade Company",
                    "American Export Partners",
                ]
            )
        elif "China" in nombre_pais:
            company_name = random.choice(
                [
                    "Shanghai Trading Co Ltd",
                    "Guangzhou Import Export",
                    "Beijing Commercial Group",
                ]
            )
        elif nombre_pais in ["Guatemala", "Honduras", "Nicaragua", "Costa Rica"]:
            company_name = random.choice(
                [
                    f"Distribuidora {nombre_pais} SA",
                    f"Importaciones {nombre_pais} Ltda",
                    f"Comercial {nombre_pais} Corp",
                ]
            )
        else:
            company_name = f"International Trading #{invoice_num}"

        return {
            "company_name": company_name,
            "cod_pais": cod_pais,
            "nombre_pais": nombre_pais,
            "tipo_documento": tipo_doc,
            "num_documento": doc_number,
            "address": address,
        }

    def generate_export_documents(self, invoice_num: int) -> List[Dict]:
        """Generate export documents - always returns 1 customs document (type 2)"""
        documents = []

        # Always use customs document (type 2)
        doc = {
            "cod_doc_asociado": 2,
            "desc_documento": f"DUA-2025-{invoice_num:06d}",
            "detalle_documento": "Declaración Única Aduanera - Aduana La Hachadura",
        }
        documents.append(doc)

        return documents

    def create_export_invoice(
        self,
        client: Dict,
        client_data: Dict,
        establishment: Dict,
        pos: Dict,
        items: List[Dict],
        invoice_num: int,
    ) -> Optional[Dict]:
        """Create a draft export invoice"""
        url = f"{self.base_url}/v1/invoices"

        # Build line items - ⭐ NO DISCOUNTS
        line_items = []
        for item in items:
            # Random quantity (10-100 for exports - larger quantities)
            quantity = random.randint(10, 100)

            # ⭐ NO DISCOUNT!
            line_items.append(
                {
                    "item_id": item["id"],
                    "quantity": quantity,
                    "discount_percentage": 0,  # ⭐ Always 0!
                }
            )

        # Calculate approximate totals for seguro/flete
        estimated_total = sum(
            item.get("unit_price", 0) * li["quantity"]
            for item, li in zip(items, line_items)
        )
        seguro = round(estimated_total * 0.01, 2)  # 1% insurance
        flete = round(estimated_total * 0.02, 2)  # 2% freight

        # ✅ Select INCOTERMS (now with proper codes)
        incoterms_code, incoterms_desc = random.choice(self.INCOTERMS)

        # Generate export documents
        export_documents = self.generate_export_documents(invoice_num)

        payload = {
            "client_id": client["id"],  # ✅ Use real client ID
            "establishment_id": establishment["id"],
            "point_of_sale_id": pos["id"],
            "payment_terms": "cash",
            "correo": client["correo"],
            "contact_email": client["correo"],
            "payment_method": "01",
            "notes": f"Factura de exportación #{invoice_num} - {client_data['nombre_pais']} - NO DISCOUNTS TEST",
            "line_items": line_items,
            "export_fields": {
                "tipo_item_expor": 1,  # Bienes
                "recinto_fiscal": random.choice(
                    self.RECINTO_FISCAL
                ),  # ✅ "01", "02", or "03"
                "regimen": random.choice(self.REGIMEN),  # ✅ "EX-1.1000.000", etc.
                "incoterms_code": incoterms_code,  # ✅ "09", "11", etc.
                "incoterms_desc": incoterms_desc,  # ✅ "FOB-Libre a bordo", etc.
                "seguro": seguro,
                "flete": flete,
                "observaciones": f"Exportación a {client_data['nombre_pais']} - {client_data['company_name']} [NO DISCOUNT TEST]",
                "receptor_cod_pais": client_data["cod_pais"],
                "receptor_nombre_pais": client_data["nombre_pais"],
                "receptor_tipo_documento": client_data["tipo_documento"],
                "receptor_num_documento": client_data["num_documento"],
                "receptor_complemento": client_data["address"],
            },
            "export_documents": export_documents,
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create export invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def finalize_invoice(self, invoice: Dict) -> bool:
        """Finalize an export invoice"""
        url = f"{self.base_url}/v1/invoices/{invoice['id']}/finalize"

        total = invoice.get("total", 0)

        payload = {
            "payment": {
                "amount": total,
                "payment_method": "01",
                "reference_number": f"EXP-PAY-{invoice['id'][:8]}",
            }
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            self.invoices_finalized += 1
            return True
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to finalize export invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return False

    def seed_export_invoices(
        self,
        start_date: str,
        end_date: str,
        num_invoices: int = 10,
    ):
        """Create and finalize export invoices"""
        print("=" * 70)
        print(" " * 5 + "EXPORT INVOICE (TYPE 11) SEEDER - NO DISCOUNTS TEST")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Date Range: {start_date} to {end_date}")
        print(f"Export Invoices to Create: {num_invoices}")
        print(f"⭐ Discounts: DISABLED (all items at 0% discount)\n")

        # Load data
        clients = self.get_clients()
        establishments = self.get_establishments()
        inventory_items = self.get_inventory_items()

        # Validate
        if not clients:
            print("❌ No clients found. Please create clients first.")
            sys.exit(1)

        if not establishments:
            print("❌ No establishments found.")
            sys.exit(1)

        if not inventory_items:
            print("❌ No inventory items found.")
            sys.exit(1)

        # Get points of sale
        establishments_with_pos = []
        for est in establishments:
            pos_list = self.get_points_of_sale(est["id"])
            if pos_list:
                establishments_with_pos.append(
                    {
                        "establishment": est,
                        "points_of_sale": pos_list,
                    }
                )

        if not establishments_with_pos:
            print("❌ No points of sale found.")
            sys.exit(1)

        print(f"📊 Data Summary:")
        print(f"   Clients: {len(clients)}")
        print(f"   Establishments: {len(establishments_with_pos)}")
        print(f"   Inventory Items: {len(inventory_items)}")
        print(f"   Export Destinations: {len(self.EXPORT_COUNTRIES)} countries")
        print(f"   Regimen Options: {len(self.REGIMEN)}")
        print(f"   INCOTERMS Options: {len(self.INCOTERMS)}")
        print()

        # Parse date range
        start = datetime.strptime(start_date, "%Y-%m-%d")
        end = datetime.strptime(end_date, "%Y-%m-%d")
        date_range_days = (end - start).days

        # Create export invoices
        for i in range(num_invoices):
            print(f"[{i+1}/{num_invoices}] Creating export invoice (NO DISCOUNTS)...")

            # Pick a random real client
            client = random.choice(clients)

            # Generate international client data (for export fields)
            client_data = self.create_export_client(i + 1)

            # Select establishment and POS
            est_with_pos = random.choice(establishments_with_pos)
            establishment = est_with_pos["establishment"]
            pos = random.choice(est_with_pos["points_of_sale"])

            # Select 2-5 items with stock
            items_with_stock = []
            for item in inventory_items:
                state = self.get_inventory_state(item["id"])
                if state and state.get("current_quantity", 0) > 50:
                    items_with_stock.append(item)

            if len(items_with_stock) < 2:
                print(f"      ⚠️  Insufficient items with stock, skipping...")
                continue

            # Select 2-5 items
            num_items = random.randint(2, min(5, len(items_with_stock)))
            selected_items = random.sample(items_with_stock, num_items)

            # Create export invoice (pass both real client and export data)
            invoice = self.create_export_invoice(
                client, client_data, establishment, pos, selected_items, i + 1
            )

            if not invoice:
                continue

            self.invoices_created += 1
            invoice_id = invoice["id"]
            total = invoice.get("total", 0)

            print(f"      ✅ Export invoice created: {invoice_id}")
            print(f"         Destination: {client_data['nombre_pais']}")
            print(f"         Client: {client_data['company_name']}")
            print(f"         Establishment: {establishment['nombre']}")
            print(f"         Items: {len(selected_items)}")
            print(f"         Total: ${total:.2f}")
            print(f"         Discounts: NONE (0%)")
            print(f"         Export Docs: {len(invoice.get('export_documents', []))}")

            # Finalize
            print(f"      🔄 Finalizing export invoice...")
            if self.finalize_invoice(invoice):
                print(f"      ✅ Exported! DTE Type 11 submitted to Hacienda")

            print()

        # Summary
        print("=" * 70)
        print("EXPORT SEEDING SUMMARY (NO DISCOUNTS TEST)")
        print("=" * 70)
        print(f"✅ Export Invoices Created: {self.invoices_created}")
        print(f"✅ Export Invoices Finalized: {self.invoices_finalized}")
        print(f"❌ Errors: {self.errors}")
        print("=" * 70)

        if self.errors == 0 and self.invoices_finalized > 0:
            print("\n🎉 All export invoices (NO DISCOUNTS) processed successfully!")
            print("\n💡 If this works, the issue is discount handling!")
        else:
            print(f"\n⚠️  {self.errors} error(s) occurred.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Export invoice tester - NO DISCOUNTS VERSION",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("company_id", help="Company ID")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="API base URL (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--start-date",
        required=True,
        help="Start date (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--end-date",
        required=True,
        help="End date (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--count",
        type=int,
        default=5,
        help="Number of export invoices (default: 5)",
    )

    args = parser.parse_args()

    # Validate dates
    try:
        datetime.strptime(args.start_date, "%Y-%m-%d")
        datetime.strptime(args.end_date, "%Y-%m-%d")
    except ValueError:
        print("❌ Error: Dates must be in YYYY-MM-DD format")
        sys.exit(1)

    if args.count < 1:
        print("❌ Error: --count must be at least 1")
        sys.exit(1)

    seeder = ExportInvoiceSeeder(args.base_url, args.company_id)
    seeder.seed_export_invoices(args.start_date, args.end_date, args.count)


if __name__ == "__main__":
    main()
