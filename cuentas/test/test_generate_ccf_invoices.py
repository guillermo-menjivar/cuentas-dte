#!/usr/bin/env python3
"""
Generate CCF (Crédito Fiscal) test invoices
Usage: python generate_ccf_invoices.py <company_id> [count]
"""

import sys
import time
import random
import requests
from typing import Dict, List, Any
from datetime import datetime

BASE_URL = "http://localhost:8080/v1"
MIN_ITEMS_PER_INVOICE = 1
MAX_ITEMS_PER_INVOICE = 5
MIN_QUANTITY = 1
MAX_QUANTITY = 10
MIN_DISCOUNT = 0
MAX_DISCOUNT = 10

# ⭐ HARDCODED VALID TEST CLIENT ID
VALID_TEST_CLIENT_ID = "ef766d8c-4eb3-4f07-acd8-0a3d83952757"

PAYMENT_METHODS = [
    ("01", 70),  # Cash - 70%
    ("02", 20),  # Check - 20%
    ("03", 10),  # Credit card - 10%
]


class CCFGenerator:
    def __init__(self, company_id: str):
        self.company_id = company_id
        self.headers = {"Content-Type": "application/json", "X-Company-ID": company_id}
        self.valid_client = None
        self.establishments = []
        self.pos_by_establishment = {}
        self.items = []

        self.total_created = 0
        self.total_finalized = 0
        self.total_failed = 0
        self.total_amount = 0.0
        self.failed_invoices = []

    def fetch_data(self):
        """Fetch all available data"""
        print("📊 Fetching available data...")

        # ⭐ Fetch the specific valid client by ID
        try:
            response = requests.get(
                f"{BASE_URL}/clients/{VALID_TEST_CLIENT_ID}", headers=self.headers
            )
            response.raise_for_status()
            self.valid_client = response.json()

            print(f"  ✓ Found valid test client: {self.valid_client['business_name']}")
            print(f"    NIT: {self.valid_client['nit']}")
            print(f"    NRC: {self.valid_client['ncr']}")

        except Exception as e:
            print(f"  ✗ Failed to fetch client: {e}")
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

        # Fetch POS for each establishment
        for est in all_establishments:
            est_id = est["id"]
            try:
                response = requests.get(
                    f"{BASE_URL}/establishments/{est_id}/pos", headers=self.headers
                )
                response.raise_for_status()
                pos_list = response.json().get("points_of_sale", [])
                if pos_list:
                    self.establishments.append(est)
                    self.pos_by_establishment[est_id] = pos_list
                    print(f"  ✓ Found {len(pos_list)} POS for {est['nombre']}")
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

        if not self.valid_client or not self.establishments or not self.items:
            print("  ✗ Missing required data!")
            return False

        print("\n✅ Data fetching complete!\n")
        print(
            f"⚠️  All invoices will use client: {self.valid_client['business_name']}\n"
        )
        return True

    def get_random_payment_method(self) -> str:
        methods, weights = zip(*PAYMENT_METHODS)
        return random.choices(methods, weights=weights)[0]

    def create_invoice(self) -> Dict[str, Any]:
        """Create a single CCF invoice"""
        client = self.valid_client
        establishment = random.choice(self.establishments)
        pos = random.choice(self.pos_by_establishment[establishment["id"]])

        num_items = random.randint(MIN_ITEMS_PER_INVOICE, MAX_ITEMS_PER_INVOICE)
        selected_items = random.sample(self.items, min(num_items, len(self.items)))

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

        payload = {
            "establishment_id": establishment["id"],
            "point_of_sale_id": pos["id"],
            "client_id": VALID_TEST_CLIENT_ID,  # ⭐ Use the hardcoded ID
            "payment_terms": "cash",
            "notes": f"CCF Test Invoice #{self.total_created + 1}",
            "payment_method": "01",
            "contact_email": client.get("correo", "test@example.com"),
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
        """Finalize a CCF invoice"""
        payment_method = self.get_random_payment_method()

        payload = {
            "payment": {
                "amount": total_amount,
                "payment_method": payment_method,
                "reference": f"CCF-PAY-{invoice_id[:8]}",
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

            dte_status = result.get("dte_status")

            if result.get("status") == "finalized":
                if dte_status == "signed":
                    return True
                elif dte_status == "failed_signing":
                    error_msg = result.get("dte_hacienda_response", "Unknown error")
                    print(f"    ⚠️  DTE submission failed: {error_msg}")
                    return False
                else:
                    print(f"    ⚠️  DTE status unclear: {dte_status}")
                    return False
            else:
                print(f"    ✗ Not finalized: {result.get('error', 'Unknown')}")
                return False

        except Exception as e:
            print(f"    ✗ Failed to finalize: {e}")
            return False

    def generate_invoices(self, count: int = 10):
        """Generate CCF invoices"""
        print(f"🚀 Generating {count} CCF invoices...\n")

        for i in range(1, count + 1):
            print(f"[{i}/{count}] Creating CCF invoice...")

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

            print(f"    ✓ Created: {invoice_number}")
            print(f"    💰 Total: ${total:.2f}")
            self.total_created += 1

            print(f"    📤 Finalizing and submitting CCF to Hacienda...")
            if self.finalize_invoice(invoice_id, total):
                print(f"    ✅ Successfully submitted CCF!")
                self.total_finalized += 1
                self.total_amount += total
            else:
                print(f"    ❌ Failed to finalize CCF")
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

            if i < count:
                time.sleep(random.uniform(2, 3))

        self.print_summary()

    def print_summary(self):
        """Print summary"""
        print("\n" + "=" * 60)
        print("📊 CCF GENERATION SUMMARY")
        print("=" * 60)
        print(f"Total Created:         {self.total_created}")
        print(f"Successfully Finalized: {self.total_finalized}")
        print(f"Failed:                {self.total_failed}")
        rate = (
            (self.total_finalized / self.total_created * 100)
            if self.total_created > 0
            else 0
        )
        print(f"Success Rate:          {rate:.1f}%")
        print(f"Total Amount Processed: ${self.total_amount:,.2f}")
        print("=" * 60)

        if self.failed_invoices:
            print("\n❌ Failed Invoices:")
            for fail in self.failed_invoices:
                inv_num = fail.get("invoice_number", "N/A")
                print(f"  #{fail['number']}: {inv_num} - Failed at {fail['stage']}")


def main():
    if len(sys.argv) < 2:
        print("Usage: python generate_ccf_invoices.py <company_id> [count]")
        print(
            "Example: python generate_ccf_invoices.py bda93a7d-45dd-4d62-823f-4213806ff68f 10"
        )
        sys.exit(1)

    company_id = sys.argv[1]
    count = int(sys.argv[2]) if len(sys.argv) > 2 else 10

    print("=" * 60)
    print("🏢 CCF INVOICE GENERATOR - Crédito Fiscal")
    print("=" * 60)
    print(f"Company ID: {company_id}")
    print(f"Target: {count} CCF invoices")
    print(f"Valid Test Client ID: {VALID_TEST_CLIENT_ID}")
    print(f"Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60 + "\n")

    generator = CCFGenerator(company_id)

    if not generator.fetch_data():
        print("\n❌ Failed to fetch required data. Exiting.")
        sys.exit(1)

    generator.generate_invoices(count)


if __name__ == "__main__":
    main()
