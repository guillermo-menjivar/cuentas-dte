#!/usr/bin/env python3
"""
Generate 90 test invoices and submit them to Hacienda
Usage: python generate_90_facturas.py <company_id>
"""

import sys
import time
import random
import requests
from typing import Dict, List, Any
from datetime import datetime

# Configuration
BASE_URL = "http://localhost:8080/v1"
MIN_ITEMS_PER_INVOICE = 1
MAX_ITEMS_PER_INVOICE = 5
MIN_QUANTITY = 1
MAX_QUANTITY = 10
MIN_DISCOUNT = 0
MAX_DISCOUNT = 10

# Payment method distribution (weights)
PAYMENT_METHODS = [
    ("01", 85),  # Cash - 85%
    ("02", 10),  # Check - 10%
    ("03", 5),  # Credit card - 5%
]


class FacturaGenerator:
    def __init__(self, company_id: str):
        self.company_id = company_id
        self.headers = {"Content-Type": "application/json", "X-Company-ID": company_id}
        self.clients = []
        self.establishments = []
        self.pos_by_establishment = {}
        self.items = []

        # Statistics
        self.total_created = 0
        self.total_finalized = 0
        self.total_failed = 0
        self.total_amount = 0.0
        self.failed_invoices = []

    def fetch_data(self):
        """Fetch all available clients, establishments, POS, and items"""
        print("📊 Fetching available data...")

        # Fetch clients
        try:
            response = requests.get(f"{BASE_URL}/clients", headers=self.headers)
            response.raise_for_status()
            all_clients = response.json().get("clients", [])

            # ⭐ FILTER: Only tipo_persona = "1" (individuals/Consumidor Final)
            self.clients = [c for c in all_clients if c.get("tipo_persona") == "1"]

            print(f"  ✓ Found {len(all_clients)} total clients")
            print(
                f"  ✓ Using {len(self.clients)} Consumidor Final clients (tipo_persona=1)"
            )

            if not self.clients:
                print(f"  ⚠ No tipo_persona=1 clients found! Need to create some.")
                return False

        except Exception as e:
            print(f"  ✗ Failed to fetch clients: {e}")
            return False

        # Fetch establishments
        try:
            response = requests.get(f"{BASE_URL}/establishments", headers=self.headers)
            response.raise_for_status()
            all_establishments = response.json().get("establishments", [])
            print(f"  ✓ Found {len(all_establishments)} establishments")
        except Exception as e:
            print(f"  ✗ Failed to fetch establishments: {e}")
            return False

        # Fetch POS for each establishment and only keep establishments with POS
        for est in all_establishments:
            est_id = est["id"]
            try:
                response = requests.get(
                    f"{BASE_URL}/establishments/{est_id}/pos", headers=self.headers
                )
                response.raise_for_status()
                pos_list = response.json().get("points_of_sale", [])
                # Only add establishment if it has at least one POS
                if pos_list:
                    self.establishments.append(est)  # ⭐ Add to list only if has POS
                    self.pos_by_establishment[est_id] = pos_list
                    print(
                        f"  ✓ Found {len(pos_list)} POS for establishment {est['nombre']}"
                    )
                else:
                    print(f"  ⚠ Skipping establishment {est['nombre']} - no POS found")
            except Exception as e:
                print(f"  ✗ Failed to fetch POS for establishment {est_id}: {e}")

        # Fetch items
        try:
            response = requests.get(f"{BASE_URL}/inventory/items", headers=self.headers)
            response.raise_for_status()
            self.items = response.json().get("items", [])
            print(f"  ✓ Found {len(self.items)} items")
        except Exception as e:
            print(f"  ✗ Failed to fetch items: {e}")
            return False

        # Validate we have enough data
        if not self.clients:
            print("  ✗ No clients found!")
            return False
        if not self.establishments:
            print("  ✗ No establishments with POS found!")
            return False
        if not self.items:
            print("  ✗ No items found!")
            return False

        print("\n✅ Data fetching complete!")
        print(f"  Using {len(self.establishments)} establishments with POS\n")
        return True

    def get_random_payment_method(self) -> str:
        """Select random payment method based on distribution"""
        methods, weights = zip(*PAYMENT_METHODS)
        return random.choices(methods, weights=weights)[0]

    def create_invoice(self) -> Dict[str, Any]:
        """Create a single invoice with random data"""
        # Select random data
        client = random.choice(self.clients)
        establishment = random.choice(self.establishments)
        pos = random.choice(self.pos_by_establishment[establishment["id"]])

        # Select 1-5 random items
        num_items = random.randint(MIN_ITEMS_PER_INVOICE, MAX_ITEMS_PER_INVOICE)
        selected_items = random.sample(self.items, min(num_items, len(self.items)))

        # Build line items with random quantities and discounts
        line_items = []
        for item in selected_items:
            quantity = random.randint(MIN_QUANTITY, MAX_QUANTITY)
            discount_pct = random.randint(MIN_DISCOUNT, MAX_DISCOUNT)
            line_items.append(
                {
                    "item_id": item["id"],
                    "quantity": quantity,
                    "discount_percentage": discount_pct,
                }
            )

        # Build invoice payload
        payload = {
            "establishment_id": establishment["id"],
            "point_of_sale_id": pos["id"],
            "client_id": client["id"],
            "payment_terms": "cash",
            "notes": f"Auto-generated test invoice #{self.total_created + 1}",
            "payment_method": "01",
            "contact_email": "test@example.com",
            "contact_whatsapp": "+50312345678",
            "line_items": line_items,
        }

        try:
            response = requests.post(
                f"{BASE_URL}/invoices", headers=self.headers, json=payload
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"    ✗ Failed to create invoice: {e}")
            return None

    def finalize_invoice(self, invoice_id: str, total_amount: float) -> bool:
        """Finalize an invoice with payment"""
        payment_method = self.get_random_payment_method()

        payload = {
            "payment": {
                "amount": total_amount,
                "payment_method": payment_method,
                "reference": f"AUTO-PAY-{invoice_id[:8]}",
            }
        }

        try:
            response = requests.post(
                f"{BASE_URL}/invoices/{invoice_id}/finalize",
                headers=self.headers,
                json=payload,
            )
            response.raise_for_status()
            result = response.json()

            # ⭐ CHECK DTE STATUS IN RESPONSE
            dte_status = result.get("dte_status")

            if result.get("status") == "finalized":
                # Check if DTE was successfully submitted
                if dte_status == "signed":
                    return True
                elif dte_status == "failed_signing":
                    error_msg = result.get("dte_hacienda_response", "Unknown error")
                    print(
                        f"    ⚠️  Invoice finalized but DTE submission failed: {error_msg}"
                    )
                    return False
                else:
                    print(
                        f"    ⚠️  Invoice finalized but DTE status unclear: {dte_status}"
                    )
                    return False
            else:
                print(
                    f"    ✗ Invoice not finalized: {result.get('error', 'Unknown error')}"
                )
                return False

        except Exception as e:
            print(f"    ✗ Failed to finalize invoice: {e}")
            return False

    def generate_invoices(self, count: int = 90):
        """Generate specified number of invoices"""
        print(f"🚀 Generating {count} invoices...\n")

        for i in range(1, count + 1):
            print(f"[{i}/{count}] Creating invoice...")

            # Create invoice
            invoice = self.create_invoice()
            if not invoice:
                self.total_failed += 1
                self.failed_invoices.append(
                    {"number": i, "stage": "creation", "reason": "API error"}
                )
                time.sleep(2)
                continue

            invoice_id = invoice.get("id")
            invoice_number = invoice.get("invoice_number")
            total = invoice.get("total", 0)

            print(f"    ✓ Created: {invoice_number} (ID: {invoice_id})")
            print(f"    💰 Total: ${total:.2f}")
            self.total_created += 1

            # Finalize invoice
            print(f"    📤 Finalizing and submitting to Hacienda...")
            if self.finalize_invoice(invoice_id, total):
                print(f"    ✅ Successfully finalized and submitted!")
                self.total_finalized += 1
                self.total_amount += total
            else:
                print(f"    ❌ Failed to finalize")
                self.total_failed += 1
                self.failed_invoices.append(
                    {
                        "number": i,
                        "invoice_id": invoice_id,
                        "invoice_number": invoice_number,
                        "stage": "finalization",
                        "total": total,
                    }
                )

            print()

            # Sleep between requests
            if i < count:
                sleep_time = random.uniform(2, 3)
                time.sleep(sleep_time)

        self.print_summary()

    def print_summary(self):
        """Print generation summary"""
        print("\n" + "=" * 60)
        print("📊 GENERATION SUMMARY")
        print("=" * 60)
        print(f"Total Created:         {self.total_created}")
        print(f"Successfully Finalized: {self.total_finalized}")
        print(f"Failed:                {self.total_failed}")
        print(
            f"Success Rate:          {(self.total_finalized/self.total_created*100) if self.total_created > 0 else 0:.1f}%"
        )
        print(f"Total Amount Processed: ${self.total_amount:,.2f}")
        print("=" * 60)

        if self.failed_invoices:
            print("\n❌ Failed Invoices:")
            for fail in self.failed_invoices:
                print(
                    f"  #{fail['number']}: {fail.get('invoice_number', 'N/A')} - Failed at {fail['stage']}"
                )

        print(
            f"\n✅ Generation completed at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        )


def main():
    if len(sys.argv) < 2:
        print("Usage: python generate_90_facturas.py <company_id>")
        print(
            "Example: python generate_90_facturas.py bda93a7d-45dd-4d62-823f-4213806ff68f"
        )
        sys.exit(1)

    company_id = sys.argv[1]

    print("=" * 60)
    print("🏭 FACTURA GENERATOR - El Salvador DTE System")
    print("=" * 60)
    print(f"Company ID: {company_id}")
    print(f"Target: 90 invoices")
    print(f"Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60 + "\n")

    generator = FacturaGenerator(company_id)

    # Fetch data
    if not generator.fetch_data():
        print("\n❌ Failed to fetch required data. Exiting.")
        sys.exit(1)

    # Generate invoices
    generator.generate_invoices(90)


if __name__ == "__main__":
    main()
