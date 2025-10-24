#!/usr/bin/env python3
"""
Test Script for Inventory Legal Compliance (Article 142-A)
Tests purchase recording with legal fields and validates reports
"""

import requests
import sys
from datetime import datetime
from pathlib import Path


class InventoryLegalComplianceTest:
    """Tests legal compliance features for inventory"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}
        self.test_item_id = None
        self.passed = 0
        self.failed = 0

    def log_test(self, test_name: str, passed: bool, message: str = ""):
        """Log test result"""
        if passed:
            print(f"✅ PASS: {test_name}")
            if message:
                print(f"   {message}")
            self.passed += 1
        else:
            print(f"❌ FAIL: {test_name}")
            if message:
                print(f"   {message}")
            self.failed += 1
        print()

    def get_test_item(self):
        """Get a test product item"""
        print("🔍 Getting test product item...")
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            items = response.json().get("items", [])

            if not items:
                print("❌ No product items found. Please create a product first.")
                sys.exit(1)

            self.test_item_id = items[0]["id"]
            print(f"✅ Using item: {items[0]['sku']} - {items[0]['name']}")
            print(f"   Item ID: {self.test_item_id}\n")
            return True

        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to get items: {e}")
            sys.exit(1)

    def test_invalid_document_format_should_fail(self):
        """Test 7: Invalid document number format should fail"""
        print("📝 Test 7: Invalid Document Number Format (should fail)")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 10,
            "unit_cost": 400.00,
            "document_type": "03",
            "document_number": "FAKE-DOC-12345",  # Invalid format
            "supplier_name": "Proveedor Test",
            "supplier_nit": "0614-123456-001-2",
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)

            if response.status_code == 400:
                error = response.json().get("error", "")
                if "valid DTE numero de control" in error:
                    self.log_test(
                        "Document format validation",
                        True,
                        f"Correctly rejected: {error}",
                    )
                else:
                    self.log_test(
                        "Document format validation",
                        False,
                        f"Wrong error message: {error}",
                    )
            else:
                self.log_test(
                    "Document format validation",
                    False,
                    f"Expected 400, got {response.status_code}",
                )

        except requests.exceptions.RequestException as e:
            self.log_test("Document format validation", False, f"Request failed: {e}")

    def test_ccf_purchase_with_nit(self):
        """Test 1: Record CCF purchase with NIT (should succeed)"""
        print("📝 Test 1: CCF Purchase with NIT")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 50,
            "unit_cost": 425.00,
            "document_type": "03",
            "document_number": "DTE-03-M001P001-000000000010011",
            "supplier_name": "Distribuidora Tech El Salvador S.A. de C.V.",
            "supplier_nit": "0614-987654-001-1",
            "cost_source_ref": "Libro de Compras Folio 45",
            "notes": "Compra de prueba con CCF",
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            data = response.json()

            # Verify response has legal fields
            has_doc_type = data.get("document_type") == "03"
            has_supplier = data.get("supplier_name") == payload["supplier_name"]
            has_nit = data.get("supplier_nit") == payload["supplier_nit"]
            has_nationality = data.get("supplier_nationality") == "Nacional"

            if has_doc_type and has_supplier and has_nit and has_nationality:
                self.log_test(
                    "CCF with NIT",
                    True,
                    f"Event ID: {data.get('event_id')}, All legal fields present",
                )
            else:
                self.log_test("CCF with NIT", False, "Missing legal fields in response")

        except requests.exceptions.RequestException as e:
            self.log_test("CCF with NIT", False, f"Request failed: {e}")

    def test_factura_purchase_without_nit(self):
        """Test 2: Record Factura purchase without NIT (should succeed)"""
        print("📝 Test 2: Factura Purchase without NIT")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 25,
            "unit_cost": 400.00,
            "document_type": "01",
            "document_number": "FAC-00056789",
            "supplier_name": "Tienda Local de Computadoras",
            "cost_source_ref": "Libro de Compras Folio 46",
            "notes": "Compra menor con factura",
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            data = response.json()

            has_doc_type = data.get("document_type") == "01"
            has_supplier = data.get("supplier_name") == payload["supplier_name"]
            nit_is_none = data.get("supplier_nit") is None

            if has_doc_type and has_supplier and nit_is_none:
                self.log_test(
                    "Factura without NIT",
                    True,
                    f"Event ID: {data.get('event_id')}, NIT correctly null",
                )
            else:
                self.log_test("Factura without NIT", False, "Unexpected field values")

        except requests.exceptions.RequestException as e:
            self.log_test("Factura without NIT", False, f"Request failed: {e}")

    def test_ccf_without_nit_should_fail(self):
        """Test 3: CCF without NIT should fail validation"""
        print("📝 Test 3: CCF without NIT (should fail)")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 10,
            "unit_cost": 400.00,
            "document_type": "03",
            "document_number": "DTE-03-00099999",
            "supplier_name": "Proveedor Sin NIT",
            "notes": "Esta compra debe fallar",
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)

            if response.status_code == 400:
                error = response.json().get("error", "")
                if "supplier_nit is required" in error:
                    self.log_test(
                        "CCF validation (no NIT)", True, f"Correctly rejected: {error}"
                    )
                else:
                    self.log_test(
                        "CCF validation (no NIT)",
                        False,
                        f"Wrong error message: {error}",
                    )
            else:
                self.log_test(
                    "CCF validation (no NIT)",
                    False,
                    f"Expected 400, got {response.status_code}",
                )

        except requests.exceptions.RequestException as e:
            self.log_test("CCF validation (no NIT)", False, f"Request failed: {e}")

    def test_invalid_document_type_should_fail(self):
        """Test 4: Invalid document type should fail validation"""
        print("📝 Test 4: Invalid Document Type (should fail)")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 10,
            "unit_cost": 400.00,
            "document_type": "99",
            "document_number": "DOC-12345",
            "supplier_name": "Proveedor Test",
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)

            if response.status_code == 400:
                error = response.json().get("error", "")
                if "document_type must be" in error:
                    self.log_test(
                        "Document type validation",
                        True,
                        f"Correctly rejected: {error}",
                    )
                else:
                    self.log_test(
                        "Document type validation",
                        False,
                        f"Wrong error message: {error}",
                    )
            else:
                self.log_test(
                    "Document type validation",
                    False,
                    f"Expected 400, got {response.status_code}",
                )

        except requests.exceptions.RequestException as e:
            self.log_test("Document type validation", False, f"Request failed: {e}")

    def test_legal_register_has_data(self):
        """Test 5: Legal register CSV should contain populated fields"""
        print("📝 Test 5: Legal Register Contains Data")

        today = datetime.now().strftime("%Y-%m-%d")
        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/legal-register"
        params = {
            "start_date": "2025-01-01",
            "end_date": today,
            "format": "csv",
            "language": "es",
        }

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()

            csv_content = response.text
            lines = csv_content.split("\n")

            # Check header exists
            has_header = "REGISTRO DE CONTROL DE INVENTARIOS" in lines[0]

            # Check for data rows (skip header rows, look for correlativo)
            data_rows = [
                line
                for line in lines
                if line
                and not line.startswith(
                    (
                        "REGISTRO",
                        "Empresa",
                        "NIT",
                        "NRC",
                        "Período",
                        "Artículo",
                        "Correlativo",
                    )
                )
            ]

            # Check if any row has populated supplier name (not empty)
            has_supplier_data = any(
                len(row.split(",")) >= 5 and row.split(",")[4].strip()
                for row in data_rows
            )

            if has_header and has_supplier_data:
                self.log_test(
                    "Legal register populated",
                    True,
                    f"Found {len(data_rows)} data rows with supplier info",
                )
            else:
                self.log_test(
                    "Legal register populated",
                    False,
                    f"Header OK: {has_header}, Data: {has_supplier_data}",
                )

        except requests.exceptions.RequestException as e:
            self.log_test("Legal register populated", False, f"Request failed: {e}")

    def test_invalid_nit_format_should_fail(self):
        """Test 6: Invalid NIT format should fail validation"""
        print("📝 Test 6: Invalid NIT Format (should fail)")

        url = f"{self.base_url}/v1/inventory/items/{self.test_item_id}/purchase"
        payload = {
            "quantity": 10,
            "unit_cost": 400.00,
            "document_type": "03",
            "document_number": "DTE-03-00077777",
            "supplier_name": "Proveedor con NIT Invalido",
            "supplier_nit": "123456789",  # Invalid format
        }

        try:
            response = requests.post(url, headers=self.headers, json=payload)

            if response.status_code == 400:
                error = response.json().get("error", "")
                if "invalid supplier_nit format" in error.lower():
                    self.log_test(
                        "NIT format validation", True, f"Correctly rejected: {error}"
                    )
                else:
                    self.log_test(
                        "NIT format validation",
                        False,
                        f"Wrong error message: {error}",
                    )
            else:
                self.log_test(
                    "NIT format validation",
                    False,
                    f"Expected 400, got {response.status_code}",
                )

        except requests.exceptions.RequestException as e:
            self.log_test("NIT format validation", False, f"Request failed: {e}")

    def run_all_tests(self):
        """Run all tests"""
        print("=" * 70)
        print(" " * 15 + "INVENTORY LEGAL COMPLIANCE TEST SUITE")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Base URL: {self.base_url}")
        print(f"Date: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")

        # Get test item
        self.get_test_item()

        # Run tests
        self.test_ccf_purchase_with_nit()
        self.test_factura_purchase_without_nit()
        self.test_ccf_without_nit_should_fail()
        self.test_invalid_document_type_should_fail()
        self.test_invalid_nit_format_should_fail()
        self.test_legal_register_has_data()
        self.self.test_invalid_document_format_should_fail()

        # Summary
        print("=" * 70)
        print("TEST SUMMARY")
        print("=" * 70)
        print(f"✅ Passed: {self.passed}")
        print(f"❌ Failed: {self.failed}")
        print(f"📊 Total:  {self.passed + self.failed}")
        print("=" * 70)

        if self.failed == 0:
            print("\n🎉 All tests passed! Legal compliance is working correctly.\n")
            return 0
        else:
            print(f"\n⚠️  {self.failed} test(s) failed. Please review.\n")
            return 1


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Test inventory legal compliance features"
    )
    parser.add_argument("company_id", help="Company ID to test with")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Base URL of the API (default: http://localhost:8080)",
    )

    args = parser.parse_args()

    tester = InventoryLegalComplianceTest(args.base_url, args.company_id)
    exit_code = tester.run_all_tests()
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
