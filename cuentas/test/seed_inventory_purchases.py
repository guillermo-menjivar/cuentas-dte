#!/usr/bin/env python3
"""
Seed Inventory with Random Purchase Transactions
Generates 1-10 random purchases for each product to populate legal registers
"""

import requests
import random
import sys
from datetime import datetime, timedelta
from typing import List, Dict


class InventoryPurchaseSeeder:
    """Seeds inventory with random compliant purchase transactions"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}
        self.purchases_created = 0
        self.errors = 0

    def get_all_products(self) -> List[Dict]:
        """Get all product items"""
        print("🔍 Fetching all product items...")
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            items = response.json().get("items", [])
            print(f"✅ Found {len(items)} product items\n")
            return items
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get items: {e}")
            sys.exit(1)

    def generate_random_dte_number(self, doc_type: str, seed: int) -> str:
        """Generate a valid random DTE number"""
        # Format: DTE-02-MSBP003P001-000000000000001
        tipo = doc_type
        establecimiento_tipo = random.choice(["M", "S", "B", "P"])
        establecimiento = f"{random.randint(1, 999):03d}"
        punto_venta = f"{random.randint(1, 999):03d}"
        correlativo = f"{seed:015d}"

        return f"DTE-{tipo}-{establecimiento_tipo}{establecimiento}P{punto_venta}-{correlativo}"

    def generate_suppliers(self) -> List[Dict]:
        """Generate a pool of realistic suppliers with normalized NITs"""
        return [
            {
                "name": "Distribuidora Nacional S.A. de C.V.",
                "nit": "0614-123456-001-2",  # Normalized (removed internal dashes)
            },
            {
                "name": "Comercial El Salvador Import Export",
                "nit": "0614-234567-002-3",
            },
            {
                "name": "Importadora Tech Central America",
                "nit": "0614-345678-003-4",
            },
            {
                "name": "Proveedor Mayorista San Salvador",
                "nit": "06144-567892-004-5",
            },
            {
                "name": "Suministros Industriales SV",
                "nit": "0614-567890-005-6",
            },
            {
                "name": "Global Trade El Salvador S.A.",
                "nit": "0614-678901-006-7",
            },
            {
                "name": "Distribuciones del Pacífico",
                "nit": "0614-789012-007-8",
            },
        ]

    def generate_random_purchase(
        self, item: Dict, suppliers: List[Dict], purchase_num: int, date: datetime
    ) -> Dict:
        """Generate a random purchase payload"""
        # Random document type (70% CCF, 30% Factura)
        doc_type = "03" if random.random() < 0.7 else "01"

        # Random supplier
        supplier = random.choice(suppliers)

        # Random quantity based on product type
        quantity = random.randint(5, 100)

        # Random unit cost (realistic range based on item)
        base_price = item.get("unit_price", 100)
        # Cost is typically 50-80% of unit price
        unit_cost = round(base_price * random.uniform(0.5, 0.8), 2)

        # Generate DTE number
        dte_seed = int(date.timestamp()) + purchase_num
        document_number = self.generate_random_dte_number(doc_type, dte_seed)

        # CRITICAL FIX: ALWAYS include supplier name and NIT
        payload = {
            "quantity": quantity,
            "unit_cost": unit_cost,
            "document_type": doc_type,
            "document_number": document_number,
            "supplier_name": supplier["name"],  # ✅ ALWAYS present
            "supplier_nit": supplier["nit"],  # ✅ ALWAYS present (even for tipo 01)
            "supplier_nationality": "Nacional",  # ✅ ALWAYS present
            "cost_source_ref": f"Libro de Compras Folio {random.randint(10, 200)}",
            "notes": f"Compra automática de prueba - {date.strftime('%Y-%m-%d')}",
        }

        return payload

    def create_purchase(self, item_id: str, payload: Dict) -> bool:
        """Create a purchase transaction"""
        url = f"{self.base_url}/v1/inventory/items/{item_id}/purchase"

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            self.purchases_created += 1
            return True
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed: {e}")
            self.errors += 1
            return False

    def seed_purchases(
        self,
        start_date: str,
        end_date: str,
        min_purchases: int = 1,
        max_purchases: int = 10,
    ):
        """Seed random purchases for all products"""
        print("=" * 70)
        print(" " * 15 + "INVENTORY PURCHASE SEEDER")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Date Range: {start_date} to {end_date}")
        print(f"Purchases per Product: {min_purchases}-{max_purchases} random\n")

        # Get all products
        items = self.get_all_products()

        if not items:
            print("❌ No product items found. Please create products first.")
            sys.exit(1)

        # Generate supplier pool
        suppliers = self.generate_suppliers()

        # Parse date range
        start = datetime.strptime(start_date, "%Y-%m-%d")
        end = datetime.strptime(end_date, "%Y-%m-%d")
        date_range_days = (end - start).days

        # Seed each product
        for idx, item in enumerate(items, 1):
            sku = item["sku"]
            name = item["name"]
            item_id = item["id"]

            # Random number of purchases for this product
            num_purchases = random.randint(min_purchases, max_purchases)

            print(f"[{idx}/{len(items)}] 📦 {sku} - {name}")
            print(f"         Generating {num_purchases} purchases...")

            for i in range(num_purchases):
                # Random date within range
                random_days = random.randint(0, max(date_range_days, 1))
                purchase_date = start + timedelta(days=random_days)

                # Generate purchase
                payload = self.generate_random_purchase(
                    item, suppliers, i + 1, purchase_date
                )

                # Create purchase
                doc_type_name = "CCF" if payload["document_type"] == "03" else "Factura"
                if self.create_purchase(item_id, payload):
                    print(
                        f"         ✅ Purchase {i+1}/{num_purchases}: {payload['quantity']} units @ ${payload['unit_cost']} ({doc_type_name}) - {payload['supplier_name']}"
                    )
                else:
                    print(f"         ❌ Purchase {i+1}/{num_purchases} failed")

            print()

        # Summary
        print("=" * 70)
        print("SEEDING SUMMARY")
        print("=" * 70)
        print(f"✅ Purchases Created: {self.purchases_created}")
        print(f"❌ Errors: {self.errors}")
        print(f"📦 Products Seeded: {len(items)}")
        print("=" * 70)

        if self.errors == 0:
            print("\n🎉 All purchases created successfully!")
            print("\n💡 Next steps:")
            print(
                "   1. Run: ./test_audit_inventory_reports.py [COMPANY_ID] --mode audit ..."
            )
            print("   2. Check legal registers - all products should have data now!\n")
        else:
            print(f"\n⚠️  {self.errors} error(s) occurred during seeding.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Seed inventory with random purchase transactions",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Seed October 2025 with 1-10 purchases per product
  %(prog)s COMPANY_ID --start-date 2025-10-01 --end-date 2025-10-31
  
  # Seed with exactly 5 purchases per product
  %(prog)s COMPANY_ID --start-date 2025-10-01 --end-date 2025-10-31 --min 5 --max 5
  
  # Seed full year with lots of transactions
  %(prog)s COMPANY_ID --start-date 2025-01-01 --end-date 2025-12-31 --min 20 --max 50
        """,
    )
    parser.add_argument("company_id", help="Company ID to seed purchases for")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Base URL of the API (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--start-date",
        required=True,
        help="Start date for purchase transactions (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--end-date",
        required=True,
        help="End date for purchase transactions (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--min",
        type=int,
        default=1,
        help="Minimum purchases per product (default: 1)",
    )
    parser.add_argument(
        "--max",
        type=int,
        default=10,
        help="Maximum purchases per product (default: 10)",
    )

    args = parser.parse_args()

    # Validate dates
    try:
        datetime.strptime(args.start_date, "%Y-%m-%d")
        datetime.strptime(args.end_date, "%Y-%m-%d")
    except ValueError:
        print("❌ Error: Dates must be in YYYY-MM-DD format")
        sys.exit(1)

    if args.min > args.max:
        print("❌ Error: --min cannot be greater than --max")
        sys.exit(1)

    seeder = InventoryPurchaseSeeder(args.base_url, args.company_id)
    seeder.seed_purchases(args.start_date, args.end_date, args.min, args.max)


if __name__ == "__main__":
    main()
