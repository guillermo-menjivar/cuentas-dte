#!/usr/bin/env python3
"""
E2E Test: CCF Contingency Flow

Tests the complete contingency system:
1. Firmador fails → invoices queued for contingency
2. Firmador recovers → workers process queue
3. Invoices submitted to Hacienda via contingency event + lote
"""

import requests
import time
import sys
from datetime import datetime
from typing import List, Dict, Optional


class CCFContingencyTester:
    """Tests CCF contingency flow end-to-end"""

    def __init__(self, base_url: str, mock_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.mock_url = mock_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}

        # Track test state
        self.created_invoices = []
        self.contingency_period_id = None
        self.test_results = []

    # ============================================
    # MOCK FIRMADOR CONTROL
    # ============================================

    def set_firmador_mode(self, mode: str) -> bool:
        """Set mock firmador mode: 'fail' or 'success'"""
        try:
            response = requests.post(
                f"{self.mock_url}/control/mode", json={"mode": mode}, timeout=5
            )
            response.raise_for_status()
            result = response.json()
            print(f"   ✅ Firmador mode set to: {result['mode']}")
            return True
        except Exception as e:
            print(f"   ❌ Failed to set firmador mode: {e}")
            return False

    def get_firmador_status(self) -> Optional[Dict]:
        """Get mock firmador status"""
        try:
            response = requests.get(f"{self.mock_url}/control/status", timeout=5)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"   ❌ Failed to get firmador status: {e}")
            return None

    def reset_firmador(self) -> bool:
        """Reset mock firmador state"""
        try:
            response = requests.post(f"{self.mock_url}/control/reset", timeout=5)
            response.raise_for_status()
            print("   ✅ Firmador mock reset")
            return True
        except Exception as e:
            print(f"   ❌ Failed to reset firmador: {e}")
            return False

    # ============================================
    # API HELPERS
    # ============================================

    def get_establishments(self) -> List[Dict]:
        """Get all establishments"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/establishments", headers=self.headers, timeout=10
            )
            response.raise_for_status()
            data = response.json()
            return data.get("establishments", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get establishments: {e}")
            return []

    def get_points_of_sale(self, establishment_id: str) -> List[Dict]:
        """Get points of sale for an establishment"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/establishments/{establishment_id}/pos",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            data = response.json()
            return data.get("points_of_sale", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get POS: {e}")
            return []

    def get_clients(self) -> List[Dict]:
        """Get clients for invoicing"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/clients", headers=self.headers, timeout=10
            )
            response.raise_for_status()
            data = response.json()
            return data.get("clients", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get clients: {e}")
            return []

    # ============================================
    # INVOICE OPERATIONS
    # ============================================

    def create_ccf_invoice(
        self, establishment_id: str, pos_id: str, client_id: str, index: int
    ) -> Optional[Dict]:
        """Create a CCF invoice (Type 03)"""
        payload = {
            "establishment_id": establishment_id,
            "point_of_sale_id": pos_id,
            "client_id": client_id,
            "payment_terms": "contado",
            "payment_method": "01",
            "notes": f"Contingency Test Invoice #{index}",
            "line_items": [
                {
                    "line_number": 1,
                    "item_type": 1,
                    "item_sku": f"TEST-{index:03d}",
                    "item_name": f"Test Product for Contingency #{index}",
                    "quantity": 1,
                    "unit_of_measure": "99",
                    "unit_price": 10.00 + index,
                    "discount_amount": 0,
                }
            ],
        }

        try:
            response = requests.post(
                f"{self.base_url}/v1/invoices",
                headers=self.headers,
                json=payload,
                timeout=10,
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"   ❌ Failed to create invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    print(f"      Error: {e.response.json()}")
                except:
                    print(f"      Error: {e.response.text}")
            return None

    def finalize_invoice(self, invoice_id: str) -> Optional[Dict]:
        """Finalize an invoice (triggers DTE processing)"""
        try:
            response = requests.post(
                f"{self.base_url}/v1/invoices/{invoice_id}/finalize",
                headers=self.headers,
                json={},
                timeout=30,
            )
            # Don't raise_for_status - we expect this might fail/queue for contingency
            return {
                "status_code": response.status_code,
                "body": response.json() if response.content else None,
            }
        except Exception as e:
            print(f"   ❌ Failed to finalize invoice: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    print(f"      Error: {e.response.json()}")
                except:
                    print(f"      Error: {e.response.text}")
            return None

    def get_invoice(self, invoice_id: str) -> Optional[Dict]:
        """Get invoice details"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/invoices/{invoice_id}",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"   ❌ Failed to get invoice: {e}")
            return None

    # ============================================
    # CONTINGENCY API
    # ============================================

    def get_contingency_periods(self) -> List[Dict]:
        """Get all contingency periods"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/contingency/periods",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json().get("periods", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get contingency periods: {e}")
            return []

    def get_contingency_period(self, period_id: str) -> Optional[Dict]:
        """Get contingency period details"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/contingency/periods/{period_id}",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"   ❌ Failed to get contingency period: {e}")
            return None

    def get_contingency_period_invoices(self, period_id: str) -> List[Dict]:
        """Get invoices in a contingency period"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/contingency/periods/{period_id}/invoices",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json().get("invoices", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get period invoices: {e}")
            return []

    def get_contingency_events(self) -> List[Dict]:
        """Get all contingency events"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/contingency/events",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json().get("events", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get contingency events: {e}")
            return []

    def get_contingency_lotes(self) -> List[Dict]:
        """Get all contingency lotes"""
        try:
            response = requests.get(
                f"{self.base_url}/v1/contingency/lotes",
                headers=self.headers,
                timeout=10,
            )
            response.raise_for_status()
            return response.json().get("lotes", []) or []
        except Exception as e:
            print(f"   ❌ Failed to get contingency lotes: {e}")
            return []

    # ============================================
    # TEST PHASES
    # ============================================

    def phase_1_setup(self) -> bool:
        """Phase 1: Setup - verify environment and set firmador to fail"""
        print("\n" + "=" * 70)
        print("PHASE 1: SETUP")
        print("=" * 70)

        # Check firmador mock
        print("\n1.1 Checking mock firmador...")
        status = self.get_firmador_status()
        if not status:
            print("   ❌ Mock firmador not reachable")
            return False
        print(f"   ✅ Mock firmador running - mode: {status['mode']}")

        # Reset firmador state
        print("\n1.2 Resetting firmador mock...")
        if not self.reset_firmador():
            return False

        # Set to fail mode
        print("\n1.3 Setting firmador to FAIL mode...")
        if not self.set_firmador_mode("fail"):
            return False

        # Get establishments
        print("\n1.4 Loading establishments...")
        establishments = self.get_establishments()
        if not establishments:
            print("   ❌ No establishments found")
            return False
        print(f"   ✅ Found {len(establishments)} establishments")

        # Get POS for first establishment
        self.establishment = establishments[0]
        print(f"\n1.5 Loading POS for establishment: {self.establishment['nombre']}...")
        pos_list = self.get_points_of_sale(self.establishment["id"])
        if not pos_list:
            print("   ❌ No POS found")
            return False
        self.pos = pos_list[0]
        print(f"   ✅ Using POS: {self.pos['nombre']}")

        # Get clients (need one with tipo_persona for CCF)
        print("\n1.6 Loading clients...")
        clients = self.get_clients()
        if not clients:
            print("   ❌ No clients found")
            return False

        # Find a client suitable for CCF (contribuyente)
        self.client = None
        for client in clients:
            if client.get("nit") or client.get("nrc"):
                self.client = client
                break

        if not self.client:
            self.client = clients[0]  # Use first client if no contribuyente found

        print(f"   ✅ Using client: {self.client['name']}")

        print("\n✅ PHASE 1 COMPLETE - Environment ready")
        return True

    def phase_2_create_invoices(self, count: int = 3) -> bool:
        """Phase 2: Create and finalize invoices (will fail and queue for contingency)"""
        print("\n" + "=" * 70)
        print(f"PHASE 2: CREATE {count} INVOICES (Firmador will fail)")
        print("=" * 70)

        for i in range(1, count + 1):
            print(f"\n2.{i}a Creating invoice #{i}...")
            invoice = self.create_ccf_invoice(
                self.establishment["id"], self.pos["id"], self.client["id"], i
            )
            if not invoice:
                print(f"   ❌ Failed to create invoice #{i}")
                continue

            invoice_id = invoice["id"]
            invoice_number = invoice.get("invoice_number", "N/A")
            print(f"   ✅ Created invoice: {invoice_number} (ID: {invoice_id})")
            self.created_invoices.append(invoice)

            print(f"\n2.{i}b Finalizing invoice #{i} (expecting contingency queue)...")
            result = self.finalize_invoice(invoice_id)
            if result:
                print(f"   Status: {result['status_code']}")
                if result["body"]:
                    # Check if queued for contingency
                    body = result["body"]
                    if (
                        "error" in str(body).lower()
                        and "contingency" in str(body).lower()
                    ):
                        print(f"   ✅ Invoice queued for contingency (as expected)")
                    else:
                        print(f"   Response: {body}")

        print(f"\n✅ PHASE 2 COMPLETE - Created {len(self.created_invoices)} invoices")
        return len(self.created_invoices) > 0

    def phase_3_verify_contingency(self) -> bool:
        """Phase 3: Verify invoices are queued in contingency"""
        print("\n" + "=" * 70)
        print("PHASE 3: VERIFY CONTINGENCY STATE")
        print("=" * 70)

        print("\n3.1 Checking contingency periods...")
        periods = self.get_contingency_periods()

        if not periods:
            print("   ❌ No contingency periods found")
            return False

        # Find active period
        active_periods = [p for p in periods if p.get("status") == "active"]
        if not active_periods:
            print("   ❌ No active contingency period found")
            print(f"   Found periods: {[p.get('status') for p in periods]}")
            return False

        self.contingency_period_id = active_periods[0]["id"]
        print(f"   ✅ Active period found: {self.contingency_period_id}")

        print("\n3.2 Checking invoices in period...")
        period_detail = self.get_contingency_period(self.contingency_period_id)
        if period_detail:
            print(f"   Period status: {period_detail.get('status')}")
            print(f"   Invoice count: {period_detail.get('invoice_count', 'N/A')}")

        invoices = self.get_contingency_period_invoices(self.contingency_period_id)
        if not invoices:
            print("   ❌ No invoices found in contingency period")
            return False

        print(f"   ✅ Found {len(invoices)} invoices in contingency")

        # Check transmission status
        for inv in invoices[:5]:  # Show first 5
            status = inv.get("dte_transmission_status", "unknown")
            print(f"      - {inv.get('invoice_number', inv.get('id'))}: {status}")

        print("\n✅ PHASE 3 COMPLETE - Contingency state verified")
        return True

    def phase_4_restore_firmador(self) -> bool:
        """Phase 4: Restore firmador to success mode"""
        print("\n" + "=" * 70)
        print("PHASE 4: RESTORE FIRMADOR")
        print("=" * 70)

        print("\n4.1 Setting firmador to SUCCESS mode (proxy to real)...")
        if not self.set_firmador_mode("success"):
            return False

        # Verify
        status = self.get_firmador_status()
        if status and status.get("mode") == "success":
            print(
                f"   ✅ Firmador restored - will proxy to: {status.get('real_firmador')}"
            )
        else:
            print("   ❌ Failed to verify firmador mode")
            return False

        print("\n✅ PHASE 4 COMPLETE - Firmador restored")
        return True

    def phase_5_wait_for_workers(
        self, timeout_seconds: int = 180, poll_interval: int = 10
    ) -> bool:
        """Phase 5: Wait for workers to process contingency"""
        print("\n" + "=" * 70)
        print(f"PHASE 5: WAIT FOR WORKERS (timeout: {timeout_seconds}s)")
        print("=" * 70)

        if not self.contingency_period_id:
            print("   ❌ No contingency period ID")
            return False

        start_time = time.time()
        completed = False

        while time.time() - start_time < timeout_seconds:
            elapsed = int(time.time() - start_time)
            print(f"\n5.x Polling... ({elapsed}s elapsed)")

            period = self.get_contingency_period(self.contingency_period_id)
            if not period:
                print("   ❌ Failed to get period status")
                time.sleep(poll_interval)
                continue

            status = period.get("status", "unknown")
            print(f"   Period status: {status}")

            if status == "completed":
                print("   ✅ Period completed!")
                completed = True
                break
            elif status == "reporting":
                print("   ⏳ Period in reporting phase...")
            elif status == "active":
                print(
                    "   ⏳ Period still active (waiting for service recovery detection)..."
                )

            # Check lotes
            lotes = self.get_contingency_lotes()
            if lotes:
                for lote in lotes[-3:]:  # Show last 3
                    print(
                        f"   Lote {lote.get('id', 'N/A')[:8]}: {lote.get('status', 'unknown')}"
                    )

            time.sleep(poll_interval)

        if completed:
            print(
                f"\n✅ PHASE 5 COMPLETE - Workers finished in {int(time.time() - start_time)}s"
            )
        else:
            print(
                f"\n⚠️  PHASE 5 TIMEOUT - Workers did not complete in {timeout_seconds}s"
            )
            print("   This may be due to worker intervals. Check logs.")

        return completed

    def phase_6_verify_recovery(self) -> bool:
        """Phase 6: Verify all invoices were successfully submitted"""
        print("\n" + "=" * 70)
        print("PHASE 6: VERIFY RECOVERY")
        print("=" * 70)

        all_success = True

        print("\n6.1 Checking period final status...")
        if self.contingency_period_id:
            period = self.get_contingency_period(self.contingency_period_id)
            if period:
                print(f"   Period status: {period.get('status')}")
                if period.get("status") != "completed":
                    all_success = False

        print("\n6.2 Checking contingency events...")
        events = self.get_contingency_events()
        if events:
            print(f"   Found {len(events)} contingency events")
            for event in events[-3:]:
                estado = event.get("hacienda_estado", "unknown")
                print(f"   Event {event.get('id', 'N/A')[:8]}: {estado}")
                if estado != "PROCESADO":
                    all_success = False
        else:
            print("   ⚠️  No contingency events found")

        print("\n6.3 Checking lotes...")
        lotes = self.get_contingency_lotes()
        if lotes:
            print(f"   Found {len(lotes)} lotes")
            for lote in lotes[-3:]:
                status = lote.get("status", "unknown")
                print(f"   Lote {lote.get('id', 'N/A')[:8]}: {status}")
                if status != "completed":
                    all_success = False
        else:
            print("   ⚠️  No lotes found")

        print("\n6.4 Checking invoice sello_recibido...")
        for invoice in self.created_invoices[:5]:
            inv_detail = self.get_invoice(invoice["id"])
            if inv_detail:
                sello = inv_detail.get("dte_sello_recibido")
                status = inv_detail.get("dte_status")
                inv_num = inv_detail.get("invoice_number", invoice["id"])
                if sello:
                    print(f"   ✅ {inv_num}: {status} - sello: {sello[:20]}...")
                else:
                    print(f"   ❌ {inv_num}: {status} - NO SELLO")
                    all_success = False

        if all_success:
            print("\n✅ PHASE 6 COMPLETE - All invoices recovered successfully!")
        else:
            print("\n⚠️  PHASE 6 COMPLETE - Some issues detected")

        return all_success

    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 70)
        print("TEST SUMMARY")
        print("=" * 70)
        print(f"Invoices created:     {len(self.created_invoices)}")
        print(f"Contingency period:   {self.contingency_period_id or 'N/A'}")

        # Final firmador status
        status = self.get_firmador_status()
        if status:
            print(f"Firmador requests:    {status.get('request_count', 'N/A')}")

        print("=" * 70)

    def run(self, invoice_count: int = 3, worker_timeout: int = 180):
        """Run the full E2E test"""
        print("=" * 70)
        print("    CCF CONTINGENCY E2E TEST")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"API URL:    {self.base_url}")
        print(f"Mock URL:   {self.mock_url}")
        print(f"Started:    {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

        try:
            # Phase 1: Setup
            if not self.phase_1_setup():
                print("\n❌ TEST FAILED - Setup failed")
                return False

            # Phase 2: Create invoices (firmador failing)
            if not self.phase_2_create_invoices(invoice_count):
                print("\n❌ TEST FAILED - Could not create invoices")
                return False

            # Phase 3: Verify contingency state
            if not self.phase_3_verify_contingency():
                print("\n⚠️  WARNING - Contingency verification failed")
                print("   Invoices may not have been queued for contingency")
                # Continue anyway to see what happens

            # Phase 4: Restore firmador
            if not self.phase_4_restore_firmador():
                print("\n❌ TEST FAILED - Could not restore firmador")
                return False

            # Phase 5: Wait for workers
            workers_completed = self.phase_5_wait_for_workers(worker_timeout)

            # Phase 6: Verify recovery
            recovery_success = self.phase_6_verify_recovery()

            # Summary
            self.print_summary()

            if workers_completed and recovery_success:
                print("\n🎉 TEST PASSED - Full contingency flow completed!")
                return True
            else:
                print("\n⚠️  TEST COMPLETED WITH WARNINGS")
                return False

        except KeyboardInterrupt:
            print("\n\n⚠️  Test interrupted by user")
            return False
        except Exception as e:
            print(f"\n❌ TEST FAILED - Unexpected error: {e}")
            import traceback

            traceback.print_exc()
            return False
        finally:
            # Always try to restore firmador to success mode
            print("\n🔄 Cleanup: Restoring firmador to success mode...")
            self.set_firmador_mode("success")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="E2E Test for CCF Contingency Flow",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python test_ccf_contingency.py COMPANY_ID
  python test_ccf_contingency.py COMPANY_ID --invoices 5
  python test_ccf_contingency.py COMPANY_ID --timeout 300

Test Flow:
  1. Set mock firmador to FAIL mode
  2. Create CCF invoices → queued for contingency
  3. Verify invoices in contingency period
  4. Restore firmador to SUCCESS mode (proxy to real)
  5. Wait for workers to process
  6. Verify all invoices have sello_recibido
        """,
    )
    parser.add_argument("company_id", help="Company ID")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="API base URL (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--mock-url",
        default="http://localhost:8114",
        help="Mock firmador URL (default: http://localhost:8114)",
    )
    parser.add_argument(
        "--invoices",
        type=int,
        default=3,
        help="Number of invoices to create (default: 3)",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=180,
        help="Worker timeout in seconds (default: 180)",
    )

    args = parser.parse_args()

    tester = CCFContingencyTester(args.base_url, args.mock_url, args.company_id)
    success = tester.run(args.invoices, args.timeout)

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
