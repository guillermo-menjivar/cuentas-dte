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
                "receptor_cod_pais": client_data["co
