#!/usr/bin/env python3
"""
Inventory Cost Reports Exporter
Exports inventory cost tracking data to CSV files for accounting review
"""

import argparse
import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import List
import requests


class InventoryReportExporter:
    """Exports inventory cost reports to CSV format"""

    def __init__(
        self, base_url: str, company_id: str, output_dir: str, language: str = "es"
    ):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.output_dir = Path(output_dir)
        self.language = language  # Default: Spanish
        self.timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.headers = {"X-Company-ID": company_id}

        # Create output directory
        self.output_dir.mkdir(parents=True, exist_ok=True)

    def export_legal_register_for_item(
        self, item_id: str, start_date: str, end_date: str
    ) -> str:
        """Export legal inventory register for a specific item (Article 142-A compliant)"""
        print(
            f"📊 Exporting Legal Register for Item {item_id} ({start_date} to {end_date})"
        )

        filename = (
            f"registro_legal_{item_id}_{start_date}_to_{end_date}_{self.timestamp}.csv"
        )
        filepath = self._download_csv(
            f"/v1/inventory/items/{item_id}/legal-register",
            {"start_date": start_date, "end_date": end_date},
            filename,
        )

        if filepath:
            print(f"   ✅ Saved to: {filepath}\n")
        return filepath

    def export_legal_registers_all_items(
        self, start_date: str, end_date: str
    ) -> List[str]:
        """Export legal registers for ALL product items (Article 142-A compliance)"""
        print(
            f"📊 Exporting Legal Registers for All Products ({start_date} to {end_date})"
        )

        # Get all product items
        url = f"{self.base_url}/v1/inventory/items"
        params = {"active": "true", "tipo_item": "1"}  # Only products

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()
            items = response.json().get("items", [])

            if not items:
                print("   ⚠️  No product items found\n")
                return []

            print(f"   Found {len(items)} product items\n")

            files = []
            for item in items:
                item_id = item["id"]
                sku = item["sku"]
                name = item["name"]
                print(f"   📄 Generating register for: {sku} - {name}")

                file = self.export_legal_register_for_item(
                    item_id, start_date, end_date
                )
                if file:
                    files.append(file)

            return files

        except requests.exceptions.RequestException as e:
            print(f"❌ Error getting items: {e}")
            return []

    def _download_csv(self, endpoint: str, params: dict, filename: str) -> str:
        """Download CSV directly from API"""
        url = f"{self.base_url}{endpoint}"
        params["format"] = "csv"
        params["language"] = self.language  # Add language parameter

        try:
            response = requests.get(url, headers=self.headers, params=params)
            response.raise_for_status()

            filepath = self.output_dir / filename
            with open(filepath, "wb") as f:
                f.write(response.content)

            return str(filepath)
        except requests.exceptions.RequestException as e:
            print(f"❌ Error downloading {endpoint}: {e}")
            return ""

    def export_current_states(self) -> str:
        """Export current inventory states"""
        print("📊 Exporting Report 1: Current Inventory States")

        filename = f"inventory_states_{self.timestamp}.csv"
        filepath = self._download_csv("/v1/inventory/states", {}, filename)

        if filepath:
            print(f"   ✅ Saved to: {filepath}\n")
        return filepath

    def export_in_stock_items(self) -> str:
        """Export only in-stock items"""
        print("📊 Exporting Report 2: In-Stock Items")

        filename = f"inventory_in_stock_{self.timestamp}.csv"
        filepath = self._download_csv(
            "/v1/inventory/states", {"in_stock_only": "true"}, filename
        )

        if filepath:
            print(f"   ✅ Saved to: {filepath}\n")
        return filepath

    def export_events_by_date_range(
        self, start_date: str, end_date: str, event_type: str = None
    ) -> str:
        """Export all inventory events within a date range"""
        if event_type:
            print(f"📊 Exporting {event_type} Events from {start_date} to {end_date}")
            filename = f"inventory_{event_type.lower()}_events_{start_date}_to_{end_date}_{self.timestamp}.csv"
        else:
            print(f"📊 Exporting All Events from {start_date} to {end_date}")
            filename = (
                f"inventory_events_{start_date}_to_{end_date}_{self.timestamp}.csv"
            )

        params = {"start_date": start_date, "end_date": end_date, "sort": "asc"}

        if event_type:
            params["event_type"] = event_type

        filepath = self._download_csv("/v1/inventory/events", params, filename)

        if filepath:
            print(f"   ✅ Saved to: {filepath}\n")
        return filepath

    def export_valuation_at_date(self, as_of_date: str) -> str:
        """Export inventory valuation at a specific date"""
        print(f"📊 Exporting Inventory Valuation as of {as_of_date}")

        filename = f"inventory_valuation_as_of_{as_of_date}_{self.timestamp}.csv"
        filepath = self._download_csv(
            "/v1/inventory/valuation", {"as_of_date": as_of_date}, filename
        )

        if filepath:
            print(f"   ✅ Saved to: {filepath}\n")
        return filepath

    def export_all(self):
        """Export all current state reports"""
        print("╔" + "=" * 60 + "╗")
        print("║" + " " * 15 + "INVENTORY COST REPORTS - CSV EXPORT" + " " * 10 + "║")
        print("╚" + "=" * 60 + "╝")
        print(f"Company ID: {self.company_id}")
        print(f"Output Directory: {self.output_dir}")
        print(f"Timestamp: {self.timestamp}\n")

        files = []

        try:
            files.append(self.export_current_states())
            files.append(self.export_in_stock_items())

            # Create manifest
            files.append(self.create_current_manifest(files))

            print("=" * 63)
            print("✅ ALL REPORTS EXPORTED SUCCESSFULLY")
            print("=" * 63 + "\n")

            print("📁 Output Directory:", self.output_dir)
            print("📄 Files Generated:")
            for file in files:
                if file:
                    size = Path(file).stat().st_size
                    print(f"   {Path(file).name} ({size} bytes)")

            print("\n💡 To view in Excel/Google Sheets:")
            print("   1. Open any CSV file")
            print("   2. Data will be properly formatted with headers")
            print("   3. Use filters and pivot tables for analysis\n")

        except Exception as e:
            print(f"\n❌ Error during export: {e}")
            sys.exit(1)

    def export_audit_package(self, start_date: str, end_date: str):
        """Export complete audit package for a period"""
        print("╔" + "=" * 60 + "╗")
        print("║" + " " * 18 + "AUDIT PACKAGE EXPORT" + " " * 21 + "║")
        print("╚" + "=" * 60 + "╝")
        print(f"Period: {start_date} to {end_date}")
        print(f"Company ID: {self.company_id}")
        print(f"Output Directory: {self.output_dir}\n")

        files = []

        try:
            # 1. Opening balance (day before start)
            start = datetime.strptime(start_date, "%Y-%m-%d")
            opening_date = (start - timedelta(days=1)).strftime("%Y-%m-%d")

            print(f"📊 Report 1: Opening Balance (as of {opening_date})")
            files.append(self.export_valuation_at_date(opening_date))

            # 2. All movements in period
            print(f"📊 Report 2: All Inventory Movements ({start_date} to {end_date})")
            files.append(self.export_events_by_date_range(start_date, end_date))

            # 3. Purchases in period
            print(f"📊 Report 3: Purchases ({start_date} to {end_date})")
            files.append(
                self.export_events_by_date_range(start_date, end_date, "PURCHASE")
            )

            # 4. Adjustments in period
            print(f"📊 Report 4: Adjustments ({start_date} to {end_date})")
            files.append(
                self.export_events_by_date_range(start_date, end_date, "ADJUSTMENT")
            )

            # 5. Closing balance
            print(f"📊 Report 5: Closing Balance (as of {end_date})")
            files.append(self.export_valuation_at_date(end_date))

            # 6. NEW: Legal registers for all items (Article 142-A compliance)
            print(f"📊 Report 6: Legal Registers (Article 142-A) - One per Product")
            legal_registers = self.export_legal_registers_all_items(
                start_date, end_date
            )
            files.extend(legal_registers)

            # 7. Create manifest
            files.append(self.create_audit_manifest(start_date, end_date, files))

            print("=" * 63)
            print("✅ AUDIT PACKAGE EXPORTED SUCCESSFULLY")
            print("=" * 63 + "\n")

            print("📁 Files Generated:")
            for file in files:
                if file:
                    size = Path(file).stat().st_size
                    print(f"   {Path(file).name} ({size} bytes)")

            print(
                f"\n💡 This package contains everything an auditor needs for the period:"
            )
            print(f"   • Opening balance on {opening_date}")
            print(f"   • All movements from {start_date} to {end_date}")
            print(f"   • Closing balance on {end_date}")
            print(f"   • Legal registers (one per product) - Article 142-A compliant")
            print(f"   • Complete audit trail\n")

        except Exception as e:
            print(f"\n❌ Error during export: {e}")
            raise

    def create_current_manifest(self, files: List[str]) -> str:
        """Create manifest for current state export"""
        filename = f"export_manifest_{self.timestamp}.txt"
        filepath = self.output_dir / filename

        with open(filepath, "w") as f:
            f.write("INVENTORY COST REPORTS EXPORT\n")
            f.write("=" * 50 + "\n")
            f.write(f"Company ID: {self.company_id}\n")
            f.write(f"Export Date: {datetime.now()}\n")
            f.write(f"Output Directory: {self.output_dir}\n\n")

            f.write("FILES GENERATED:\n")
            for i, file in enumerate(files, 1):
                if file:
                    f.write(f"{i}. {Path(file).name}\n")

            f.write("\nUSAGE:\n")
            f.write("- Open CSV files in Excel or Google Sheets\n")
            f.write("- All monetary values are rounded to 2 decimal places\n")
            f.write("- Timestamps are in ISO 8601 format (UTC)\n")
            f.write("- Use filters and pivot tables for further analysis\n\n")

            f.write("NOTES:\n")
            f.write("- Product items (tipo_item = '1') include cost tracking\n")
            f.write("- Service items (tipo_item = '2') do not have inventory events\n")

        return str(filepath)

    def create_audit_manifest(
        self, start_date: str, end_date: str, files: List[str]
    ) -> str:
        """Create manifest for audit package"""
        filename = f"audit_manifest_{start_date}_to_{end_date}_{self.timestamp}.txt"
        filepath = self.output_dir / filename

        with open(filepath, "w") as f:
            f.write("AUDIT PACKAGE MANIFEST\n")
            f.write("=" * 50 + "\n")
            f.write(f"Company ID: {self.company_id}\n")
            f.write(f"Audit Period: {start_date} to {end_date}\n")
            f.write(f"Generated: {datetime.now()}\n")
            f.write(f"Output Directory: {self.output_dir}\n\n")

            f.write("CONTENTS:\n")
            for i, file in enumerate(files, 1):
                if file:
                    f.write(f"{i}. {Path(file).name}\n")

            f.write("\nPURPOSE:\n")
            f.write("This audit package provides complete documentation of all\n")
            f.write("inventory movements and valuations for the specified period.\n\n")

            f.write("AUDIT TRAIL:\n")
            f.write("All events are immutable and include:\n")
            f.write("- Unique event ID\n")
            f.write("- Timestamp\n")
            f.write("- Item details (SKU, name)\n")
            f.write("- Quantity changes\n")
            f.write("- Cost tracking (moving average)\n")
            f.write("- Balance after each transaction\n")
            f.write("- Notes/documentation\n\n")

            f.write("VALUATION METHOD:\n")
            f.write("Moving Average Cost (weighted average)\n\n")

            f.write("COMPLIANCE:\n")
            f.write("Reports suitable for:\n")
            f.write("- External audit\n")
            f.write("- Tax reporting\n")
            f.write("- Financial statement preparation\n")
            f.write("- Regulatory compliance (El Salvador)\n")

        return str(filepath)


def main():
    parser = argparse.ArgumentParser(
        description="Export inventory cost tracking reports to CSV files",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Export current state
  %(prog)s COMPANY_ID
  
  # Export audit package for 2024
  %(prog)s COMPANY_ID --mode audit --start-date 2024-01-01 --end-date 2024-12-31
  
  # Export Q1 2025 audit to specific directory
  %(prog)s COMPANY_ID --mode audit --start-date 2025-01-01 --end-date 2025-03-31 --output-dir ./audit_q1_2025
        """,
    )
    parser.add_argument("company_id", help="Company ID to export reports for")
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Base URL of the API (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--output-dir",
        default=".",
        help="Output directory for CSV files (default: current directory)",
    )
    parser.add_argument(
        "--mode",
        choices=["current", "audit"],
        default="current",
        help="Export mode: current (current state) or audit (period-specific)",
    )
    parser.add_argument("--start-date", help="Start date for audit mode (YYYY-MM-DD)")
    parser.add_argument("--end-date", help="End date for audit mode (YYYY-MM-DD)")

    args = parser.parse_args()

    # Validate audit mode requirements
    if args.mode == "audit":
        if not args.start_date or not args.end_date:
            print("❌ Error: --start-date and --end-date are required for audit mode")
            print("\nExample:")
            print(
                f"  {parser.prog} {args.company_id} --mode audit --start-date 2024-01-01 --end-date 2024-12-31"
            )
            sys.exit(1)

        # Validate date format
        try:
            datetime.strptime(args.start_date, "%Y-%m-%d")
            datetime.strptime(args.end_date, "%Y-%m-%d")
        except ValueError:
            print("❌ Error: Dates must be in YYYY-MM-DD format")
            sys.exit(1)

    exporter = InventoryReportExporter(
        base_url=args.base_url, company_id=args.company_id, output_dir=args.output_dir
    )

    if args.mode == "audit":
        exporter.export_audit_package(args.start_date, args.end_date)
    else:
        exporter.export_all()


if __name__ == "__main__":
    main()

"""
./export_inventory_cost_reports.py b46cbf8c-efd1-4799-9f33-2ae8049ae331 \
  --mode audit \
  --start-date 2025-01-01 \
  --end-date 2025-12-31 \
  --output-dir ./audit_2025

./export_inventory_cost_reports.py b46cbf8c-efd1-4799-9f33-2ae8049ae331 \
  --mode audit \
  --start-date 2025-10-01 \
  --end-date 2025-10-31 \
  --output-dir ./audit_october_2025

./export_inventory_cost_reports.py b46cbf8c-efd1-4799-9f33-2ae8049ae331 \
  --mode audit \
  --start-date 2025-10-01 \
  --end-date 2025-12-31 \
  --output-dir ./audit_q4_2025
"""
