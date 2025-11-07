#!/usr/bin/env python3
"""
Automated Nota de Crédito Seeder
Creates diverse credit notes: partial credits, full annulments, defects, returns, etc.
"""

import requests
import random
import sys
from datetime import datetime
from typing import List, Dict, Optional


class NotaCreditoSeeder:
    """Creates diverse Notas de Crédito for testing"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}
        self.notas_created = 0
        self.notas_finalized = 0
        self.errors = 0

    def get_finalized_ccfs(self) -> List[Dict]:
        """Get all finalized CCF invoices"""
        print("🔍 Fetching finalized CCF invoices...")
        url = f"{self.base_url}/v1/invoices"
        params = {"status": "finalized"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            data = response.json()

            invoices = data.get("invoices", []) if data else []

            # Filter for CCF only (type 03)
            ccfs = [inv for inv in invoices if inv.get("dte_type") == "03"]

            print(f"✅ Found {len(ccfs)} finalized CCF invoices\n")
            return ccfs
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get invoices: {e}")
            return []

    def get_ccf_details(self, ccf_id: str) -> Optional[Dict]:
        """Get full CCF details with line items"""
        url = f"{self.base_url}/v1/invoices/{ccf_id}"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ⚠️  Failed to fetch CCF {ccf_id}: {e}")
            return None

    def get_existing_credits(self, ccf_line_item_id: str) -> float:
        """Check if line item already has credits (simplified - would need API endpoint)"""
        # In production, you'd have an API endpoint to check this
        # For now, we'll assume no credits exist
        return 0.0

    def create_nota_credito(
        self,
        ccf: Dict,
        line_items: List[Dict],
        credit_reason: str,
        description: str,
        notes: str,
    ) -> Optional[Dict]:
        """Create a nota de crédito"""
        url = f"{self.base_url}/v1/notas/credito"

        payload = {
            "ccf_ids": [ccf["id"]],
            "credit_reason": credit_reason,
            "credit_description": description,
            "line_items": line_items,
            "payment_terms": "contado",
            "notes": notes,
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json().get("nota")
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create nota: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def finalize_nota(self, nota_id: str) -> bool:
        """Finalize a nota de crédito"""
        url = f"{self.base_url}/v1/notas/credito/{nota_id}/finalize"

        try:
            response = requests.post(url, headers=self.headers)
            response.raise_for_status()
            self.notas_finalized += 1
            return True
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to finalize nota: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return False

    def generate_partial_credit(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate a partial credit (50% of one item)"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        # Pick random line item
        line = random.choice(line_items)

        # Credit 50% of quantity at full price
        qty_to_credit = line["quantity"] * 0.5

        credit_line_items = [
            {
                "related_ccf_id": ccf_details["id"],
                "ccf_line_item_id": line["id"],
                "quantity_credited": qty_to_credit,
                "credit_amount": line["unit_price"],
                "credit_reason": "Partial credit - customer dissatisfaction",
            }
        ]

        return {
            "line_items": credit_line_items,
            "credit_reason": "quality",
            "description": f"Crédito parcial por calidad - {line['item_name']} (50% de cantidad)",
            "notes": f"Automated partial credit - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "PARTIAL_CREDIT",
        }

    def generate_full_annulment(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate a full annulment (100% of all items)"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        # Credit ALL items at 100%
        credit_line_items = []
        for line in line_items:
            credit_line_items.append(
                {
                    "related_ccf_id": ccf_details["id"],
                    "ccf_line_item_id": line["id"],
                    "quantity_credited": line["quantity"],
                    "credit_amount": line["unit_price"],
                    "credit_reason": "Full annulment",
                }
            )

        return {
            "line_items": credit_line_items,
            "credit_reason": "void",
            "description": f"Anulación total del CCF {ccf_details['invoice_number']}",
            "notes": f"Automated full annulment - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "FULL_ANNULMENT",
        }

    def generate_defect_credit(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate a defect credit (25-75% of one item)"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        line = random.choice(line_items)

        # Credit random % (25-75%)
        credit_pct = random.uniform(0.25, 0.75)
        qty_to_credit = line["quantity"] * credit_pct

        credit_line_items = [
            {
                "related_ccf_id": ccf_details["id"],
                "ccf_line_item_id": line["id"],
                "quantity_credited": qty_to_credit,
                "credit_amount": line["unit_price"],
                "credit_reason": "Defective items returned",
            }
        ]

        return {
            "line_items": credit_line_items,
            "credit_reason": "defect",
            "description": f"Producto defectuoso - {line['item_name']} ({int(credit_pct*100)}% de cantidad)",
            "notes": f"Automated defect credit - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "DEFECT",
        }

    def generate_return_credit(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate a return credit (100% of one item)"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        line = random.choice(line_items)

        credit_line_items = [
            {
                "related_ccf_id": ccf_details["id"],
                "ccf_line_item_id": line["id"],
                "quantity_credited": line["quantity"],
                "credit_amount": line["unit_price"],
                "credit_reason": "Customer returned item",
            }
        ]

        return {
            "line_items": credit_line_items,
            "credit_reason": "return",
            "description": f"Devolución completa - {line['item_name']}",
            "notes": f"Automated return credit - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "RETURN",
        }

    def generate_discount_credit(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate a discount credit (reduce price, same quantity)"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        line = random.choice(line_items)

        # Credit with reduced price (10-30% off)
        discount_pct = random.uniform(0.10, 0.30)
        discounted_price = line["unit_price"] * discount_pct

        credit_line_items = [
            {
                "related_ccf_id": ccf_details["id"],
                "ccf_line_item_id": line["id"],
                "quantity_credited": line["quantity"],
                "credit_amount": discounted_price,
                "credit_reason": "Price adjustment",
            }
        ]

        return {
            "line_items": credit_line_items,
            "credit_reason": "discount",
            "description": f"Ajuste de precio - {line['item_name']} ({int(discount_pct*100)}% descuento)",
            "notes": f"Automated discount credit - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "DISCOUNT",
        }

    def generate_overbilling_credit(self, ccf_details: Dict) -> Optional[Dict]:
        """Generate an overbilling correction"""
        line_items = ccf_details.get("line_items", [])
        if not line_items:
            return None

        line = random.choice(line_items)

        # Credit small quantity (10-30%)
        qty_to_credit = line["quantity"] * random.uniform(0.10, 0.30)

        credit_line_items = [
            {
                "related_ccf_id": ccf_details["id"],
                "ccf_line_item_id": line["id"],
                "quantity_credited": qty_to_credit,
                "credit_amount": line["unit_price"],
                "credit_reason": "Overbilling correction",
            }
        ]

        return {
            "line_items": credit_line_items,
            "credit_reason": "overbilling",
            "description": f"Corrección de sobrefacturación - {line['item_name']}",
            "notes": f"Automated overbilling correction - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}",
            "scenario": "OVERBILLING",
        }

    def seed_notas(self, target_count: int = 50):
        """Generate diverse notas de crédito"""
        print("=" * 70)
        print(" " * 15 + "NOTA DE CRÉDITO SEEDER")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Target: {target_count} notas de crédito\n")

        # Get available CCFs
        ccfs = self.get_finalized_ccfs()

        if len(ccfs) < 10:
            print(f"❌ Not enough CCF invoices! Found {len(ccfs)}, need at least 10.")
            print("   Run the invoice seeder first to create more CCFs.")
            sys.exit(1)

        print(f"📊 Available CCFs: {len(ccfs)}\n")

        # Define scenario distribution
        scenarios = [
            ("PARTIAL_CREDIT", 15, self.generate_partial_credit),
            ("FULL_ANNULMENT", 10, self.generate_full_annulment),
            ("DEFECT", 10, self.generate_defect_credit),
            ("RETURN", 8, self.generate_return_credit),
            ("DISCOUNT", 5, self.generate_discount_credit),
            ("OVERBILLING", 2, self.generate_overbilling_credit),
        ]

        print("📋 Scenario Distribution:")
        for scenario_name, count, _ in scenarios:
            print(f"   {scenario_name}: {count} notas")
        print()

        # Track used CCFs to avoid conflicts
        used_ccfs = set()
        created_count = 0

        for scenario_name, scenario_count, generator_func in scenarios:
            print(f"\n{'='*70}")
            print(f"Generating {scenario_count} {scenario_name} notas...")
            print(f"{'='*70}\n")

            for i in range(scenario_count):
                if created_count >= target_count:
                    break

                # Find unused CCF
                attempts = 0
                ccf = None
                while attempts < 50:
                    candidate = random.choice(ccfs)
                    if candidate["id"] not in used_ccfs:
                        ccf = candidate
                        break
                    attempts += 1

                if not ccf:
                    print(f"   ⚠️  No more unused CCFs available")
                    break

                print(
                    f"[{created_count + 1}/{target_count}] {scenario_name} - CCF {ccf['invoice_number']}..."
                )

                # Get full CCF details
                ccf_details = self.get_ccf_details(ccf["id"])
                if not ccf_details:
                    continue

                # Generate nota config
                nota_config = generator_func(ccf_details)
                if not nota_config:
                    print(f"   ⚠️  Could not generate config for CCF")
                    continue

                # Create nota
                nota = self.create_nota_credito(
                    ccf_details,
                    nota_config["line_items"],
                    nota_config["credit_reason"],
                    nota_config["description"],
                    nota_config["notes"],
                )

                if not nota:
                    continue

                self.notas_created += 1
                created_count += 1
                nota_id = nota["id"]
                nota_number = nota["nota_number"]

                print(f"   ✅ Created: {nota_number}")
                print(f"      Scenario: {nota_config['scenario']}")
                print(f"      Total: ${nota['total']:.2f}")
                print(f"      Full Annulment: {nota.get('is_full_annulment', False)}")

                # Finalize nota
                print(f"   🔄 Finalizing...")
                if self.finalize_nota(nota_id):
                    print(f"   ✅ Finalized successfully!")
                    used_ccfs.add(ccf["id"])

                print()

        # Summary
        print("\n" + "=" * 70)
        print("SEEDING SUMMARY")
        print("=" * 70)
        print(f"✅ Notas Created: {self.notas_created}")
        print(f"✅ Notas Finalized: {self.notas_finalized}")
        print(f"❌ Errors: {self.errors}")
        print("=" * 70)

        if self.errors == 0 and self.notas_finalized > 0:
            print("\n🎉 All notas processed successfully!")
            print("\n💡 Next steps:")
            print("   1. Verify in Hacienda portal")
            print("   2. Check DTE commit log")
            print("   3. Generate reports")
            print()
        else:
            print(f"\n⚠️  {self.errors} error(s) occurred during seeding.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Automated Nota de Crédito seeder with diverse scenarios",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Create 50 diverse notas de crédito
  %(prog)s COMPANY_ID --count 50

  # Create 20 notas for testing
  %(prog)s COMPANY_ID --count 20
        """,
    )
    parser.add_argument("company_id", help="Company ID to create notas for")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Base URL of the API (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--count",
        type=int,
        default=50,
        help="Number of notas to create (default: 50)",
    )

    args = parser.parse_args()

    if args.count < 1:
        print("❌ Error: --count must be at least 1")
        sys.exit(1)

    seeder = NotaCreditoSeeder(args.base_url, args.company_id)
    seeder.seed_notas(args.count)


if __name__ == "__main__":
    main()
