#!/usr/bin/env python3
"""
Automated FSE (Type 14) Purchase Creation and Testing
Tests FSE purchase creation, finalization, DTE submission to Hacienda

FSE = Factura Sujeto Excluido (purchases from informal suppliers)
Common in El Salvador for:
- Agricultural products from small farmers
- Informal services (cleaning, maintenance)
- Street vendors and market purchases
- Small-scale suppliers without formal registration
"""

import requests
import random
import sys
import time
from datetime import datetime, timedelta
from typing import List, Dict, Optional


class FSETester:
    """Tests FSE (Type 14) purchase functionality end-to-end"""

    def __init__(self, base_url: str, company_id: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.headers = {"X-Company-ID": company_id, "Content-Type": "application/json"}

        # Statistics
        self.purchases_created = 0
        self.purchases_finalized = 0
        self.errors = 0
        self.test_results = []

    # ============================================
    # REALISTIC EL SALVADOR SUPPLIER SCENARIOS
    # ============================================

    SUPPLIER_SCENARIOS = [
        {
            "name": "María González - Vendedora de Tomates",
            "description": "Small farmer selling fresh tomatoes at Mercado Central",
            "document_type": "37",  # Otro
            "activity_code": "01113",  # Cultivo de hortalizas
            "activity_desc": "Cultivo de hortalizas, raíces y tubérculos",
            "department": "06",  # San Salvador
            "municipality": "14",  # San Salvador
            "address": "Mercado Central, Puesto 45-B",
            "phone": "7845-6789",
            "email": "maria.tomates@example.com",
            "items": [
                {
                    "description": "Tomates frescos de cosecha",
                    "quantity": 50,
                    "unit_price": 0.50,
                    "unit_measure": 99,  # Unidad
                }
            ],
        },
        {
            "name": "José Ramírez - Carnicero",
            "description": "Butcher selling fresh beef at local market",
            "document_type": "37",
            "activity_code": "47221",  # Venta al por menor de carne
            "activity_desc": "Venta al por menor de carne y productos cárnicos",
            "department": "06",
            "municipality": "14",
            "address": "Mercado Municipal, Local 12",
            "phone": "7956-3421",
            "email": None,
            "items": [
                {
                    "description": "Carne de res fresca - corte especial",
                    "quantity": 25,
                    "unit_price": 4.50,
                    "unit_measure": 14,  # Kilogramo
                },
                {
                    "description": "Huesos para sopa",
                    "quantity": 10,
                    "unit_price": 0.75,
                    "unit_measure": 14,  # Kilogramo
                },
            ],
        },
        {
            "name": "Doña Carmen - Pupusera",
            "description": "Street vendor selling prepared pupusas",
            "document_type": "13",  # DUI
            "document_number": "03456789-0",
            "activity_code": "56101",  # Actividades de restaurantes
            "activity_desc": "Actividades de restaurantes y de servicio móvil de comidas",
            "department": "06",
            "municipality": "14",
            "address": "Avenida España, frente al Parque Libertad",
            "phone": "7234-5678",
            "email": None,
            "items": [
                {
                    "description": "Pupusas revueltas (orden de 100)",
                    "quantity": 100,
                    "unit_price": 0.50,
                    "unit_measure": 99,  # Unidad
                },
            ],
        },
        {
            "name": "Carlos Méndez - Servicio de Limpieza",
            "description": "Informal cleaning service for offices",
            "document_type": "37",
            "activity_code": "81210",  # Limpieza general de edificios
            "activity_desc": "Limpieza general de edificios",
            "department": "06",
            "municipality": "14",
            "address": "Colonia Escalón, Calle Mirador #234",
            "phone": "7812-9045",
            "email": "carlos.limpieza@example.com",
            "items": [
                {
                    "description": "Servicio de limpieza profunda - oficinas",
                    "quantity": 1,
                    "unit_price": 75.00,
                    "unit_measure": 99,  # Servicio
                },
            ],
        },
        {
            "name": "Ana Portillo - Frutas Tropicales",
            "description": "Fruit vendor at market selling tropical fruits",
            "document_type": "37",
            "activity_code": "01251",  # Cultivo de frutas tropicales
            "activity_desc": "Cultivo de frutas tropicales y subtropicales",
            "department": "13",  # La Libertad
            "municipality": "07",  # Santa Tecla
            "address": "Mercado de Santa Tecla, Sección Frutas",
            "phone": "7623-4512",
            "email": None,
            "items": [
                {
                    "description": "Mangos maduros - variedad Tommy",
                    "quantity": 30,
                    "unit_price": 0.35,
                    "unit_measure": 99,  # Unidad
                },
                {
                    "description": "Papayas grandes",
                    "quantity": 15,
                    "unit_price": 1.25,
                    "unit_measure": 99,  # Unidad
                },
                {
                    "description": "Cocos verdes",
                    "quantity": 20,
                    "unit_price": 0.50,
                    "unit_measure": 99,  # Unidad
                },
            ],
        },
        {
            "name": "Pedro Flores - Carpintero",
            "description": "Informal carpenter for furniture repairs",
            "document_type": "13",  # DUI
            "document_number": "04567890-1",
            "activity_code": "43320",  # Instalación de carpintería
            "activity_desc": "Instalación de carpintería",
            "department": "06",
            "municipality": "14",
            "address": "Soyapango, Calle Principal #567",
            "phone": "7934-5671",
            "email": "pedro.carpintero@example.com",
            "items": [
                {
                    "description": "Reparación de muebles de oficina",
                    "quantity": 5,
                    "unit_price": 15.00,
                    "unit_measure": 99,  # Servicio
                },
            ],
        },
        {
            "name": "Luisa Martínez - Vendedora de Flores",
            "description": "Flower vendor at street corner",
            "document_type": "37",
            "activity_code": "47761",  # Venta al por menor de flores
            "activity_desc": "Venta al por menor de flores, plantas y semillas",
            "department": "06",
            "municipality": "14",
            "address": "Boulevard de los Héroes, esquina opuesta al McDonald's",
            "phone": None,
            "email": None,
            "items": [
                {
                    "description": "Rosas rojas - arreglo especial",
                    "quantity": 12,
                    "unit_price": 3.50,
                    "unit_measure": 99,  # Unidad
                },
            ],
        },
        {
            "name": "Roberto Campos - Lavado de Vehículos",
            "description": "Car wash service - informal business",
            "document_type": "37",
            "activity_code": "45208",  # Mantenimiento y reparación de vehículos
            "activity_desc": "Mantenimiento y reparación de vehículos automotores",
            "department": "06",
            "municipality": "14",
            "address": "Autopista Sur, frente a Plaza Mundo",
            "phone": "7845-3421",
            "email": None,
            "items": [
                {
                    "description": "Lavado completo de vehículo - incluye encerado",
                    "quantity": 3,
                    "unit_price": 12.00,
                    "unit_measure": 99,  # Servicio
                },
            ],
        },
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

    def create_fse_purchase(
        self,
        establishment: Dict,
        pos: Dict,
        supplier_scenario: Dict,
        payment_condition: int = 1,  # 1=Contado, 2=Crédito
        iva_retention: float = 0,
        income_tax_retention: float = 0,
    ) -> Optional[Dict]:
        """Create a draft FSE purchase"""
        url = f"{self.base_url}/v1/purchases/fse"

        # Build supplier info
        supplier = {
            "name": supplier_scenario["name"],
            "document_type": supplier_scenario["document_type"],
            "activity_code": supplier_scenario["activity_code"],
            "activity_description": supplier_scenario["activity_desc"],
            "address": {
                "department": supplier_scenario["department"],
                "municipality": supplier_scenario["municipality"],
                "complement": supplier_scenario["address"],
            },
        }

        # Add document number if present
        if "document_number" in supplier_scenario:
            supplier["document_number"] = supplier_scenario["document_number"]

        # Add contact info
        if supplier_scenario.get("phone"):
            supplier["phone"] = supplier_scenario["phone"]
        if supplier_scenario.get("email"):
            supplier["email"] = supplier_scenario["email"]

        # Build line items from scenario
        line_items = []
        for idx, item in enumerate(supplier_scenario["items"], start=1):
            line_items.append(
                {
                    "line_number": idx,
                    "item_type": 1 if "Servicio" not in item["description"] else 2,
                    "code": f"ITEM-{idx:03d}",
                    "description": item["description"],
                    "quantity": item["quantity"],
                    "unit_of_measure": str(item["unit_measure"]),
                    "unit_price": item["unit_price"],
                    "discount_amount": 0,
                }
            )

        # Build payload
        payload = {
            "establishment_id": establishment["id"],
            "point_of_sale_id": pos["id"],
            "purchase_date": datetime.now().strftime("%Y-%m-%d"),
            "supplier": supplier,
            "line_items": line_items,
            "payment": {
                "condition": payment_condition,
                "method": "01" if payment_condition == 1 else "02",  # Efectivo / Cuenta
            },
            "iva_retained": iva_retention,
            "income_tax_retained": income_tax_retention,
            "notes": f"Compra FSE - {supplier_scenario['description']}",
        }

        # Add payment term for credit
        if payment_condition == 2:
            payload["payment"]["term"] = "02"  # Plazo (días)
            payload["payment"]["period"] = 30

        try:
            response = requests.post(url, headers=self.headers, json=payload)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to create FSE purchase: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def finalize_purchase(self, purchase_id: str) -> Optional[Dict]:
        """Finalize a purchase and submit FSE to Hacienda"""
        url = f"{self.base_url}/v1/purchases/{purchase_id}/finalize"

        try:
            response = requests.post(url, headers=self.headers, json={})
            response.raise_for_status()
            self.purchases_finalized += 1
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"      ❌ Failed to finalize purchase: {e}")
            if hasattr(e, "response") and e.response is not None:
                try:
                    error_detail = e.response.json()
                    print(f"         Error: {error_detail}")
                except:
                    print(f"         Error: {e.response.text}")
            self.errors += 1
            return None

    def get_purchase(self, purchase_id: str) -> Optional[Dict]:
        """Get purchase details"""
        url = f"{self.base_url}/v1/purchases/{purchase_id}"

        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException:
            return None

    def test_scenario_cash_purchase(
        self, establishment: Dict, pos: Dict, supplier_scenario: Dict
    ):
        """
        TEST: Cash Purchase from Informal Supplier
        - Create FSE purchase (contado)
        - Finalize and submit to Hacienda
        - Verify Type 14 DTE is created
        """
        scenario_name = supplier_scenario["name"]
        print("\n" + "=" * 70)
        print(f"TEST: Cash Purchase - {scenario_name}")
        print("=" * 70)
        print(f"📋 Scenario: {supplier_scenario['description']}")
        print(f"💰 Payment: Contado (Cash)")
        print(f"📍 Location: {supplier_scenario['address']}")

        # Calculate totals
        total = sum(
            item["quantity"] * item["unit_price"] for item in supplier_scenario["items"]
        )
        print(f"💵 Total: ${total:.2f}\n")

        # Create purchase
        print(f"   Creating FSE purchase...")
        purchase = self.create_fse_purchase(
            establishment=establishment,
            pos=pos,
            supplier_scenario=supplier_scenario,
            payment_condition=1,  # Contado
        )

        if not purchase:
            self.log_result(f"Create FSE - {scenario_name}", False, "Failed to create")
            return

        self.purchases_created += 1
        purchase_id = purchase["id"]
        purchase_number = purchase.get("purchase_number", "N/A")

        self.log_result(
            f"Create FSE - {scenario_name}",
            True,
            "Purchase created (draft)",
            {
                "ID": purchase_id,
                "Number": purchase_number,
                "Total": f"${purchase['total']:.2f}",
                "Status": purchase["status"],
            },
        )

        # Finalize and submit
        print(f"   Finalizing and submitting Type 14 DTE to Hacienda...")
        result = self.finalize_purchase(purchase_id)

        if result:
            dte_status = result.get("dte_status")
            dte_numero = result.get("dte_numero_control", "N/A")
            sello = result.get("dte_sello_recibido", "N/A")

            success = dte_status == "PROCESADO"
            self.log_result(
                f"Submit FSE - {scenario_name}",
                success,
                f"Hacienda: {dte_status}",
                {
                    "NumeroControl": dte_numero,
                    "TipoDTE": "14",
                    "Sello": sello[:30] + "..." if len(sello) > 30 else sello,
                },
            )
        else:
            self.log_result(
                f"Submit FSE - {scenario_name}", False, "Finalization failed"
            )

    def run_tests(self):
        """Run all test scenarios"""
        print("=" * 70)
        print(" " * 10 + "FSE (TYPE 14) PURCHASE TESTER")
        print(" " * 5 + "Factura Sujeto Excluido - Informal Suppliers")
        print("=" * 70)
        print(f"Company ID: {self.company_id}")
        print(f"Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")

        # Load data
        establishments = self.get_establishments()

        # Validate
        if not establishments:
            print("❌ No establishments found")
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
        print(f"   Establishments: {len(establishments_with_pos)}")
        print(f"   Supplier Scenarios: {len(self.SUPPLIER_SCENARIOS)}")
        print()

        # Select establishment and POS
        est_with_pos = random.choice(establishments_with_pos)
        establishment = est_with_pos["establishment"]
        pos = random.choice(est_with_pos["points_of_sale"])

        print(f"📍 Using Establishment: {establishment['nombre']}")
        print(f"🏪 Point of Sale: {pos['nombre']}\n")

        # Run test scenarios
        # Test 1-8: Cash purchases from different suppliers (no retentions for informal suppliers)
        print("=" * 70)
        print("SECTION 1: CASH PURCHASES FROM INFORMAL SUPPLIERS")
        print("=" * 70)

        for supplier in self.SUPPLIER_SCENARIOS:
            self.test_scenario_cash_purchase(establishment, pos, supplier)
            time.sleep(2)

        # Print summary
        self.print_summary()

    def print_summary(self):
        """Print test summary"""
        print("\n" + "=" * 70)
        print("📊 TEST SUMMARY")
        print("=" * 70)
        print(f"FSE Purchases Created:   {self.purchases_created}")
        print(f"FSE Purchases Finalized: {self.purchases_finalized}")
        print(f"Errors:                  {self.errors}")
        print("=" * 70)

        # Test results
        passed = sum(1 for r in self.test_results if r["success"])
        total = len(self.test_results)

        print(f"\n✅ Tests Passed: {passed}/{total}")
        print(f"❌ Tests Failed: {total - passed}/{total}")

        if self.errors == 0 and passed == total:
            print("\n🎉 All FSE tests passed successfully!")
            print("\n💡 Next steps:")
            print("   1. Check Hacienda portal for Type 14 DTEs")
            print("   2. Verify dte_commit_log entries (tipo_dte = '14')")
            print("   3. Confirm purchases table has correct data")
            print("   4. Verify tax retentions (iva_retained, income_tax_retained)")
            print("   5. Check payment conditions (contado vs crédito)")
            print("\n📚 Supplier Scenarios Tested:")
            for scenario in self.SUPPLIER_SCENARIOS:
                print(f"   ✓ {scenario['name']} - {scenario['description']}")
            print()
        else:
            print(f"\n⚠️  Some tests failed. Check details above.\n")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Test FSE (Type 14) purchase functionality with realistic El Salvador scenarios",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python test_fse.py YOUR_COMPANY_ID
  python test_fse.py YOUR_COMPANY_ID --base-url http://api.example.com

Scenarios tested:
  1. María González - Tomates (Mercado Central)
  2. José Ramírez - Carne (Carnicero)
  3. Doña Carmen - Pupusas (Street vendor)
  4. Carlos Méndez - Limpieza (Cleaning service)
  5. Ana Portillo - Frutas Tropicales
  6. Pedro Flores - Carpintero (Carpenter)
  7. Luisa Martínez - Flores (Flower vendor)
  8. Roberto Campos - Lavado de Vehículos (Car wash)
        """,
    )
    parser.add_argument("company_id", help="Company ID")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="API base URL (default: http://localhost:8080)",
    )

    args = parser.parse_args()

    tester = FSETester(args.base_url, args.company_id)
    tester.run_tests()


if __name__ == "__main__":
    main()
