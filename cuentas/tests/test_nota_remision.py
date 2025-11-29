#!/usr/bin/env python3
"""
Automated Nota de Remisión (Type 04) Creation and Testing
Tests remision creation, finalization, DTE submission, and linking to invoices
"""

import requests
import random
import sys
import time
from datetime import datetime
from typing import List, Dict, Optional


class RemisionTester:
    """Tests remision (Type 04) functionality end-to-end"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}

        # Statistics
        self.remisiones_created = 0
        self.remisiones_finalized = 0
        self.invoices_created = 0
        self.links_created = 0
        self.errors = 0
        self.test_results = []

    # Remision types
    REMISION_TYPES = [
        "pre_invoice_delivery",  # Delivery before invoice (most common)
        "inter_branch_transfer",  # Internal transfer (no client)
        "route_sales",  # Route sales (multiple invoices)
        "other",  # Other reasons
    ]

    def log_result(
        self, test_name: str, success: bool, message: str, details: Dict = None
    ):
        """Log test result"""
        self.test_results.append(
            {
                "test": test_name,
                "success": success,
                "message": message,
                "details": details or {},
                "timestamp": datetime.now().isoformat(),
            }
        )

        icon = "✅" if success else "❌"
        print(f"      {icon} {test_name}: {message}")
        if details:
            for key, value in details.items():
                print(f"         {key}: {value}")

    def get_clients(self) -> List[Dict]:
        """Get all clients"""
        print("🔍 Fetching clients...")
        url = f"{self.base_url}/v1/clients"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            data = response.json()
            clients = data.get("clients", []) or []
            print(f"✅ Found {len(clients)} clients\n")
            return clients
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get clients: {e}")
            return []

    def get_establishments(self) -> List[Dict]:
        """Get all establishments"""
        print("🔍 Fetching establishments...")
        url = f"{self.base_url}/v1/establishments"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            data = response.json()
            establishments = data.get("establishments", []) or []
            print(f"✅ Found {len(establishments)} establishments\n")
            return establishments
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get establishments: {e}")
            return []

    def get_points_of_sale(self, establishment_id: str) -> List[Dict]:
        """Get points of sale for an establishment"""
        url = f"{self.base_url}/v1/establishments/{establishment_id}/pos"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            data = response.json()
            return data.get("points_of_sale", []) or []
        except requests.exceptions.RequestException:
            return []

    def get_inventory_items(self) -> List[Dict]:
        """Get all inventory items"""
        print("🔍 Fetching inventory items...")
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}  # Only goods

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            data = response.json()
            items = data.get("items", []) or []
            print(f"✅ Found {len(items)} items\n")
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

    def create_remision(
        self,
        remision_type: str,
        client: Optional[Dict],
        source_establishment: Dict,
        dest_establishment: Optional[Dict],
        pos: Dict,
        items: List[Dict],
        related_docs: List[Dict] = None,
    ) -> Optional[Dict]:
        """Create a draft remision"""
        url = f"{self.base_url}/v1/remisiones"

        # Build line items
        line_items = []
        for item in items:
            quantity = random.randint(5, 20)
            line_items.append(
                {
                    "item_id": item["id"],
                    "quantity": quantity,
                }
            )

        # Build payload
        payload = {
            "establishment_id": source_establishment["id"],
            "point_of_sale_id": pos["id"],
            "remision_type": remision_type,
            "line_items": line_items,
        }

        # ⭐ NEW: Add custom fields for apendice
        custom_fields = []

        # Add delivery person info if present
        delivery_person = None
        vehicle_plate = None

        if remision_type == "inter_branch_transfer":
            if dest_establishment is None:
                raise ValueError(
                    "dest_establishment required for inter_branch_transfer"
                )
            payload["destination_establishment_id"] = dest_establishment["id"]
            payload["notes"] = (
                f"Traslado de {source_establishment['nombre']} a {dest_establishment['nombre']}"
            )

            # ⭐ Custom fields for internal transfer
            delivery_person = "Carlos Rodriguez"
            vehicle_plate = f"P{random.randint(100000, 999999)}"

            custom_fields = [
                {
                    "campo": "Datos del transporte",
                    "etiqueta": "Chofer",
                    "valor": delivery_person,
                },
                {
                    "campo": "Datos del transporte",
                    "etiqueta": "Vehículo",
                    "valor": f"Placa: {vehicle_plate}",
                },
                {
                    "campo": "Datos del documento",
                    "etiqueta": "N° Documento Interno",
                    "valor": f"TRANS-{datetime.now().strftime('%Y%m%d')}-{random.randint(1000, 9999)}",
                },
            ]
        else:
            # External remision - add client
            if client is None:
                raise ValueError(f"client required for {remision_type}")
            payload["client_id"] = client["id"]
            payload["notes"] = (
                f"Remisión {remision_type} para {client['business_name']}"
            )

            # ⭐ Custom fields for external delivery
            delivery_person = "Juan Pérez"
            vehicle_plate = "P123456"

            custom_fields = [
                {
                    "campo": "Datos del cliente",
                    "etiqueta": "Contacto",
                    "valor": client.get("business_name", "N/A"),
                },
                {
                    "campo": "Datos del transporte",
                    "etiqueta": "Chofer",
                    "valor": delivery_person,
                },
                {
                    "campo": "Datos del documento",
                    "etiqueta": "Observaciones",
                    "valor": f"Entrega programada - {remision_type}",
                },
            ]

        # Add delivery info
        if remision_type in ["pre_invoice_delivery", "route_sales"]:
            payload["delivery_person"] = delivery_person or "Juan Pérez"
            payload["vehicle_plate"] = vehicle_plate or "P123456"
            payload["delivery_notes"] = f"Entrega {remision_type}"
        elif remision_type == "inter_branch_transfer":
            payload["delivery_person"] = delivery_person
            payload["vehicle_plate"] = vehicle_plate
            payload["delivery_notes"] = "Traslado interno - manejar con cuidado"

        # ⭐ Add custom fields to payload
        if custom_fields:
            payload["custom_fields"] = custom_fields

        # Add related documents if provided
        if related_docs:
            payload["related_documents"] = related_docs

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create remision: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def _create_remision(
        self,
        remision_type: str,
        client: Optional[Dict],
        source_establishment: Dict,
        dest_establishment: Optional[Dict],  # ⭐ NEW
        pos: Dict,
        items: List[Dict],
        related_docs: List[Dict] = None,
    ) -> Optional[Dict]:
        """Create a draft remision"""
        url = f"{self.base_url}/v1/remisiones"

        # Build line items
        line_items = []
        for item in items:
            quantity = random.randint(5, 20)
            line_items.append(
                {
                    "item_id": item["id"],
                    "quantity": quantity,
                }
            )

        # Build payload
        payload = {
            "establishment_id": source_establishment["id"],
            "point_of_sale_id": pos["id"],
            "remision_type": remision_type,
            "line_items": line_items,
        }

        # ⭐ NEW: Add destination establishment for internal transfers
        if remision_type == "inter_branch_transfer":
            if dest_establishment is None:
                raise ValueError(
                    "dest_establishment required for inter_branch_transfer"
                )
            payload["destination_establishment_id"] = dest_establishment["id"]
            payload["notes"] = (
                f"Traslado de {source_establishment['nombre']} a {dest_establishment['nombre']}"
            )
        else:
            # External remision - add client
            if client is None:
                raise ValueError(f"client required for {remision_type}")
            payload["client_id"] = client["id"]
            payload["notes"] = (
                f"Remisión {remision_type} para {client['business_name']}"
            )

        # Add delivery info
        if remision_type in ["pre_invoice_delivery", "route_sales"]:
            payload["delivery_person"] = "Juan Pérez"
            payload["vehicle_plate"] = "P123456"
            payload["delivery_notes"] = f"Entrega {remision_type}"

        # Add related documents if provided
        if related_docs:
            payload["related_documents"] = related_docs

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create remision: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def finalize_remision(self, remision_id: str) -> Optional[Dict]:
        """Finalize a remision and submit to Hacienda"""
        url = f"{self.base_url}/v1/remisiones/{remision_id}/finalize"

        try:
            response = requests.post(url, headers=self.headers, json={})
            response.raise_for_status()
            self.remisiones_finalized += 1
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to finalize remision: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def create_invoice(
        self,
        client: Dict,
        establishment: Dict,
        pos: Dict,
        items: List[Dict],
        related_docs: List[Dict] = None,
    ) -> Optional[Dict]:
        """Create and finalize an invoice"""
        url = f"{self.base_url}/v1/invoices"

        # Build line items
        line_items = []
        for item in items:
            quantity = random.randint(5, 20)
            discount_pct = random.choice([0, 0, 5, 10])
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
            "client_id": client["id"],
            "payment_terms": "cash",
            "payment_method": "01",
            "contact_email": client.get("correo", "test@example.com"),
            "notes": "Factura asociada a remisión",
            "line_items": line_items,
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            invoice = response.json()

            # Finalize immediately
            finalize_url = f"{self.base_url}/v1/invoices/{invoice['id']}/finalize"
            finalize_payload = {
                "payment": {
                    "amount": invoice["total"],
                    "payment_method": "01",
                    "reference_number": f"PAY-{invoice['id'][:8]}",
                }
            }

            fin_response = requests.post(
                finalize_url, headers=self.headers, json=finalize_payload
            )
            fin_response.raise_for_status()
            self.invoices_created += 1
            return fin_response.json()

        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create/finalize invoice: {e}")
            self.errors += 1
            return None

    def link_remision_to_invoice(self, remision_id: str, invoice_id: str) -> bool:
        """Link a remision to an invoice"""
        url = f"{self.base_url}/v1/remisiones/{remision_id}/link-invoice"
        payload = {"invoice_id": invoice_id}

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            self.links_created += 1
            return True
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to link: {e}")
            self.errors += 1
            return False

    def test_scenario_1_internal_transfer(
        self, establishments_with_pos: List[Dict], items: List[Dict]
    ):
        """
        TEST 1: Internal Transfer (Between Establishments)
        - Create remision with remision_type = "inter_branch_transfer"
        - Source: One establishment
        - Destination: Another establishment (same company)
        - Finalize and submit to Hacienda
        """
        print("\n" + "=" * 70)
        print("TEST 1: Internal Transfer (Between Establishments)")
        print("=" * 70)

        # ⭐ Need at least 2 establishments for internal transfer
        if len(establishments_with_pos) < 2:
            self.log_result(
                "Internal Transfer",
                False,
                "Need at least 2 establishments for internal transfer",
            )
            return

        # Select source and destination establishments
        source_est_with_pos = establishments_with_pos[0]
        dest_est_with_pos = establishments_with_pos[1]

        source_establishment = source_est_with_pos["establishment"]
        dest_establishment = dest_est_with_pos["establishment"]
        pos = source_est_with_pos["points_of_sale"][0]  # Use POS from source

        # Get items with stock
        items_with_stock = [
            item
            for item in items
            if self.get_inventory_state(item["id"])
            and self.get_inventory_state(item["id"]).get("current_quantity", 0) > 20
        ]

        if len(items_with_stock) < 2:
            self.log_result("Internal Transfer", False, "Insufficient items with stock")
            return

        selected_items = random.sample(items_with_stock, 2)

        print(f"   Creating internal transfer remision...")
        print(f"   Source: {source_establishment['nombre']}")
        print(f"   Destination: {dest_establishment['nombre']}")
        print(f"   Items: {len(selected_items)}")

        # Create remision (no client, with destination establishment)
        remision = self.create_remision(
            remision_type="inter_branch_transfer",
            client=None,  # No external client
            source_establishment=source_establishment,
            dest_establishment=dest_establishment,  # ⭐ NEW
            pos=pos,
            items=selected_items,
        )

        if not remision:
            self.log_result("Internal Transfer", False, "Failed to create remision")
            return

        self.remisiones_created += 1
        remision_id = remision["id"]

        self.log_result(
            "Create Internal Transfer",
            True,
            "Remision created",
            {
                "ID": remision_id,
                "Source": source_establishment["nombre"],
                "Destination": dest_establishment["nombre"],
            },
        )

        # Finalize and submit
        print(f"   Finalizing and submitting to Hacienda...")
        result = self.finalize_remision(remision_id)

        if result:
            hacienda_status = result.get("hacienda_status", "unknown")
            dte_numero = result.get("remision", {}).get("dte_numero_control", "N/A")

            self.log_result(
                "Submit Internal Transfer",
                hacienda_status == "PROCESADO",
                f"Hacienda: {hacienda_status}",
                {"NumeroControl": dte_numero, "Type": "04"},
            )
        else:
            self.log_result("Submit Internal Transfer", False, "Finalization failed")

    def test_scenario_2_pre_invoice_delivery(
        self,
        clients: List[Dict],
        establishments_with_pos: List[Dict],
        items: List[Dict],
    ):
        """
        TEST 2: Pre-Invoice Delivery
        - Create remision for delivery before invoice
        - With client
        - Finalize and submit to Hacienda
        """
        print("\n" + "=" * 70)
        print("TEST 2: Pre-Invoice Delivery (With Client)")
        print("=" * 70)

        client = random.choice(clients)
        est_with_pos = random.choice(establishments_with_pos)
        establishment = est_with_pos["establishment"]
        pos = random.choice(est_with_pos["points_of_sale"])

        # Get items with stock
        items_with_stock = [
            item
            for item in items
            if self.get_inventory_state(item["id"])
            and self.get_inventory_state(item["id"]).get("current_quantity", 0) > 20
        ]

        if len(items_with_stock) < 2:
            self.log_result(
                "Pre-Invoice Delivery", False, "Insufficient items with stock"
            )
            return

        selected_items = random.sample(items_with_stock, 2)

        print(f"   Creating pre-invoice delivery remision...")
        print(f"   Client: {client['business_name']}")
        print(f"   Establishment: {establishment['nombre']}")

        remision = self.create_remision(
            remision_type="pre_invoice_delivery",
            client=client,
            source_establishment=establishment,
            dest_establishment=None,  # ⭐ Not internal transfer
            pos=pos,
            items=selected_items,
        )

        if not remision:
            self.log_result("Pre-Invoice Delivery", False, "Failed to create remision")
            return

        self.remisiones_created += 1
        remision_id = remision["id"]

        self.log_result(
            "Create Pre-Invoice Delivery",
            True,
            "Remision created",
            {"ID": remision_id, "Client": client["business_name"]},
        )

        # Finalize
        print(f"   Finalizing and submitting to Hacienda...")
        result = self.finalize_remision(remision_id)

        if result:
            hacienda_status = result.get("hacienda_status", "unknown")
            dte_numero = result.get("remision", {}).get("dte_numero_control", "N/A")

            self.log_result(
                "Submit Pre-Invoice Delivery",
                hacienda_status == "PROCESADO",
                f"Hacienda: {hacienda_status}",
                {"NumeroControl": dte_numero},
            )
        else:
            self.log_result("Submit Pre-Invoice Delivery", False, "Finalization failed")

    def test_scenario_3_route_sales(
        self,
        clients: List[Dict],
        establishments_with_pos: List[Dict],
        items: List[Dict],
    ):
        """
        TEST 3: Route Sales (Remision → Multiple Invoices)
        - Create remision
        - Finalize remision
        - Create invoice for same items
        - Link remision to invoice
        """
        print("\n" + "=" * 70)
        print("TEST 3: Route Sales (Remision + Invoice + Link)")
        print("=" * 70)

        client = random.choice(clients)
        est_with_pos = random.choice(establishments_with_pos)
        establishment = est_with_pos["establishment"]
        pos = random.choice(est_with_pos["points_of_sale"])

        # Get items with stock
        items_with_stock = [
            item
            for item in items
            if self.get_inventory_state(item["id"])
            and self.get_inventory_state(item["id"]).get("current_quantity", 0) > 50
        ]

        if len(items_with_stock) < 3:
            self.log_result("Route Sales", False, "Insufficient items with stock")
            return

        selected_items = random.sample(items_with_stock, 3)

        print(f"   Step 1: Creating route sales remision...")
        print(f"   Client: {client['business_name']}")

        # Step 1: Create remision
        remision = self.create_remision(
            remision_type="route_sales",
            client=client,
            source_establishment=establishment,
            dest_establishment=None,  # ⭐ Not internal transfer
            pos=pos,
            items=selected_items,
        )

        if not remision:
            self.log_result("Route Sales - Create Remision", False, "Failed to create")
            return

        self.remisiones_created += 1
        remision_id = remision["id"]

        self.log_result(
            "Route Sales - Create Remision",
            True,
            "Remision created",
            {"ID": remision_id},
        )

        # Step 2: Finalize remision
        print(f"   Step 2: Finalizing remision...")
        result = self.finalize_remision(remision_id)

        if not result:
            self.log_result("Route Sales - Finalize Remision", False, "Failed")
            return

        hacienda_status = result.get("hacienda_status", "unknown")
        dte_numero = result.get("remision", {}).get("dte_numero_control", "N/A")

        self.log_result(
            "Route Sales - Finalize Remision",
            hacienda_status == "PROCESADO",
            f"Hacienda: {hacienda_status}",
            {"NumeroControl": dte_numero},
        )

        # Step 3: Create invoice for same items
        print(f"   Step 3: Creating invoice for delivered goods...")
        invoice = self.create_invoice(
            client=client,
            establishment=establishment,
            pos=pos,
            items=selected_items,
        )

        if not invoice:
            self.log_result("Route Sales - Create Invoice", False, "Failed")
            return

        invoice_id = invoice["id"]
        invoice_numero = invoice.get("dte_numero_control", "N/A")

        self.log_result(
            "Route Sales - Create Invoice",
            True,
            "Invoice created and finalized",
            {"ID": invoice_id, "NumeroControl": invoice_numero},
        )

        # Step 4: Link remision to invoice
        print(f"   Step 4: Linking remision to invoice...")
        time.sleep(1)  # Brief pause

        if self.link_remision_to_invoice(remision_id, invoice_id):
            self.log_result(
                "Route Sales - Link",
                True,
                "Successfully linked",
                {"RemisionID": remision_id, "InvoiceID": invoice_id},
            )
        else:
            self.log_result("Route Sales - Link", False, "Failed to link")

    def run_tests(self):
        """Run all test scenarios"""
        print("=" * 70)
        print(" " * 15 + "NOTA DE REMISIÓN (TYPE 04) TESTER")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")

        # Load data
        clients = self.get_clients()
        establishments = self.get_establishments()
        items = self.get_inventory_items()

        # Validate
        if not clients:
            print("❌ No clients found")
            sys.exit(1)
        if not establishments:
            print("❌ No establishments found")
            sys.exit(1)
        if not items:
            print("❌ No items found")
            sys.exit(1)

        # Get POS
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
            print("❌ No points of sale found")
            sys.exit(1)

        print(f"📊 Data Summary:")
        print(f"   Clients: {len(clients)}")
        print(f"   Establishments: {len(establishments_with_pos)}")
        print(f"   Inventory Items: {len(items)}\n")

        # Run tests
        self.test_scenario_1_internal_transfer(establishments_with_pos, items)
        time.sleep(2)

        self.test_scenario_2_pre_invoice_delivery(
            clients, establishments_with_pos, items
        )
        time.sleep(2)

        self.test_scenario_3_route_sales(clients, establishments_with_pos, items)

        # Print summary
        self.print_summary()

    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 70)
        print("📊 TEST SUMMARY")
        print("=" * 70)
        print(f"Remisiones Created:   {self.remisiones_created}")
        print(f"Remisiones Finalized: {self.remisiones_finalized}")
        print(f"Invoices Created:     {self.invoices_created}")
        print(f"Links Created:        {self.links_created}")
        print(f"Errors:               {self.errors}")
        print("=" * 70)

        # Test results
        passed = sum(1 for r in self.test_results if r["success"])
        total = len(self.test_results)

        print(f"\n✅ Tests Passed: {passed}/{total}")
        print(f"❌ Tests Failed: {total - passed}/{total}")

        if self.errors == 0 and passed == total:
            print("\n🎉 All tests passed successfully!")
            print("\n💡 Next steps:")
            print("   1. Check Hacienda portal for Type 04 DTEs")
            print("   2. Verify commit_log entries (tipo_dte = '04')")
            print("   3. Check remision_invoice_links table")
            print("   4. Verify destination_establishment_id for internal transfers")
            print("   5. Confirm inventory was NOT deducted\n")
        else:
            print(f"\n⚠️  Some tests failed. Check details above.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Test Nota de Remisión (Type 04) functionality",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("company_id", help="Company ID")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="API base URL (default: http://localhost:8080)",
    )

    args = parser.parse_args()

    tester = RemisionTester(args.base_url, args.company_id)
    tester.run_tests()


if __name__ == "__main__":
    main()
