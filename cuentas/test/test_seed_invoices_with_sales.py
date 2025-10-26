#!/usr/bin/env python3
"""
Automated Invoice Creation and Finalization with Sales
Creates realistic invoices, finalizes them, and tests inventory integration
"""

import requests
import random
import sys
from datetime import datetime, timedelta
from typing import List, Dict, Optional


class InvoiceSalesSeeder:
    """Creates and finalizes invoices to test sales integration"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}
        self.invoices_created = 0
        self.invoices_finalized = 0
        self.errors = 0

    def get_clients(self) -> List[Dict]:
        """Get all clients"""
        print("🔍 Fetching clients...")
        url = f"{self.base_url}/v1/clients"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            clients = response.json().get("clients", [])
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
            establishments = response.json().get("establishments", [])
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
            return response.json().get("points_of_sale", [])
        except requests.exceptions.RequestException:
            return []

    def get_inventory_items(self) -> List[Dict]:
        """Get all inventory items (goods only)"""
        print("🔍 Fetching inventory items...")
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            items = response.json().get("items", [])
            print(f"✅ Found {len(items)} inventory items\n")
            return items
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

    def create_invoice(
        self,
        client: Dict,
        establishment: Dict,
        pos: Dict,
        items: List[Dict],
        invoice_num: int,
    ) -> Optional[Dict]:
        """Create a draft invoice"""
        url = f"{self.base_url}/v1/invoices"

        # Build line items
        line_items = []
        for item in items:
            # Random quantity (1-10)
            quantity = random.randint(1, 10)

            # Random discount (0-20%)
            discount_pct = random.choice([0, 0, 0, 5, 10, 15, 20])

            line_items.append(
                {
                    "item_id": item["id"],
                    "quantity": quantity,
                    "discount_percentage": discount_pct,
                }
            )

        # Determine payment terms
        client_type = client.get("tipo_contribuyente", "02")
        payment_terms = "cash"
        if client_type in ["01", "03"]:  # Business clients
            payment_terms = random.choice(["cash", "cash", "net_30", "cuenta"])

        payload = {
            "client_id": client["id"],
            "establishment_id": establishment["id"],
            "point_of_sale_id": pos["id"],
            "payment_terms": payment_terms,
            "payment_method": "01",  # Cash
            "contact_email": client.get("correo"),
            "contact_whatsapp": client.get("telefono"),
            "notes": f"Factura de prueba automatizada #{invoice_num}",
            "line_items": line_items,
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def finalize_invoice(self, invoice: Dict) -> bool:
        """Finalize an invoice"""
        url = f"{self.base_url}/v1/invoices/{invoice['id']}/finalize"

        # Calculate payment amount
        total = invoice.get("total", 0)
        payment_amount = total  # Pay in full

        payload = {
            "payment": {
                "amount": payment_amount,
                "payment_method": "01",  # Cash
                "reference_number": f"PAY-{invoice['id'][:8]}",
            }
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            self.invoices_finalized += 1
            return True
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to finalize invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return False

    def seed_invoices(
        self,
        start_date: str,
        end_date: str,
        num_invoices: int = 10,
    ):
        """Create and finalize random invoices"""
        print("=" * 70)
        print(" " * 15 + "INVOICE SALES SEEDER")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Date Range: {start_date} to {end_date}")
        print(f"Invoices to Create: {num_invoices}\n")

        # Load data
        clients = self.get_clients()
        establishments = self.get_establishments()
        inventory_items = self.get_inventory_items()

        # Validate we have necessary data
        if not clients:
            print("❌ No clients found. Please create clients first.")
            sys.exit(1)

        if not establishments:
            print("❌ No establishments found. Please create establishments first.")
            sys.exit(1)

        if not inventory_items:
            print("❌ No inventory items found. Please create inventory first.")
            sys.exit(1)

        # Get points of sale for establishments
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
            print("❌ No points of sale found. Please create POS first.")
            sys.exit(1)

        print(f"📊 Data Summary:")
        print(f"   Clients: {len(clients)}")
        print(f"   Establishments: {len(establishments_with_pos)}")
        print(f"   Inventory Items: {len(inventory_items)}")
        print()

        # Separate clients by type
        ccf_clients = [
            c for c in clients if c.get("tipo_contribuyente") in ["01", "03"]
        ]
        consumidor_final = [c for c in clients if c.get("tipo_contribuyente") == "02"]

        print(f"   CCF Clients: {len(ccf_clients)}")
        print(f"   Consumidor Final: {len(consumidor_final)}")
        print()

        # Parse date range
        start = datetime.strptime(start_date, "%Y-%m-%d")
        end = datetime.strptime(end_date, "%Y-%m-%d")
        date_range_days = (end - start).days

        # Create invoices
        for i in range(num_invoices):
            print(f"[{i+1}/{num_invoices}] Creating invoice...")

            # Random date within range
            random_days = random.randint(0, max(date_range_days, 1))
            invoice_date = start + timedelta(days=random_days)

            # Select random client (70% consumidor final, 30% CCF)
            if consumidor_final and (not ccf_clients or random.random() < 0.7):
                client = random.choice(consumidor_final)
                client_type = "Consumidor Final"
            else:
                client = random.choice(ccf_clients if ccf_clients else consumidor_final)
                client_type = "CCF"

            # Select random establishment and POS
            est_with_pos = random.choice(establishments_with_pos)
            establishment = est_with_pos["establishment"]
            pos = random.choice(est_with_pos["points_of_sale"])

            # Select 1-5 random items with inventory
            items_with_stock = []
            max_attempts = len(inventory_items)
            attempts = 0

            while (
                len(items_with_stock) < min(5, len(inventory_items))
                and attempts < max_attempts
            ):
                item = random.choice(inventory_items)
                if item not in items_with_stock:
                    state = self.get_inventory_state(item["id"])
                    if state and state.get("current_quantity", 0) > 5:
                        items_with_stock.append(item)
                attempts += 1

            if not items_with_stock:
                print(f"      ⚠️  No items with sufficient stock, skipping...")
                continue

            # Select 1-3 items for this invoice
            num_items = random.randint(1, min(3, len(items_with_stock)))
            selected_items = random.sample(items_with_stock, num_items)

            # Create invoice
            invoice = self.create_invoice(
                client, establishment, pos, selected_items, i + 1
            )

            if not invoice:
                continue

            self.invoices_created += 1
            invoice_id = invoice["id"]
            total = invoice.get("total", 0)

            print(f"      ✅ Invoice created: {invoice_id}")
            print(f"         Client: {client['nombre_legal']} ({client_type})")
            print(f"         Establishment: {establishment['nombre']}")
            print(f"         POS: {pos['nombre']}")
            print(f"         Items: {len(selected_items)}")
            print(f"         Total: ${total:.2f}")

            # Finalize invoice
            print(f"      🔄 Finalizing invoice...")
            if self.finalize_invoice(invoice):
                print(f"      ✅ Invoice finalized - Inventory deducted!")

            print()

        # Summary
        print("=" * 70)
        print("SEEDING SUMMARY")
        print("=" * 70)
        print(f"✅ Invoices Created: {self.invoices_created}")
        print(f"✅ Invoices Finalized: {self.invoices_finalized}")
        print(f"❌ Errors: {self.errors}")
        print("=" * 70)

        if self.errors == 0 and self.invoices_finalized > 0:
            print("\n🎉 All invoices processed successfully!")
            print("\n💡 Next steps:")
            print("   1. Check inventory states - quantities should be reduced")
            print(
                "   2. Query SALE events: SELECT * FROM inventory_events WHERE event_type = 'SALE'"
            )
            print("   3. Run export script to see sales in legal registers")
            print(
                f"   4. ./test_audit_inventory_reports.py {self.company_id} --mode audit ...\n"
            )
        else:
            print(f"\n⚠️  {self.errors} error(s) occurred during seeding.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Automated invoice creation and finalization with sales",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Create 10 random invoices in October
  %(prog)s COMPANY_ID --start-date 2025-10-01 --end-date 2025-10-31 --count 10

  # Create 50 invoices for the year
  %(prog)s COMPANY_ID --start-date 2025-01-01 --end-date 2025-12-31 --count 50

  # Create just a few test invoices
  %(prog)s COMPANY_ID --start-date 2025-10-01 --end-date 2025-10-31 --count 3
        """,
    )
    parser.add_argument("company_id", help="Company ID to create invoices for")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Base URL of the API (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--start-date",
        required=True,
        help="Start date for invoice dates (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--end-date",
        required=True,
        help="End date for invoice dates (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--count",
        type=int,
        default=10,
        help="Number of invoices to create (default: 10)",
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

    seeder = InvoiceSalesSeeder(args.base_url, args.company_id)
    seeder.seed_invoices(args.start_date, args.end_date, args.count)


if __name__ == "__main__":
    main()
