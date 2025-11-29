#!/usr/bin/env python3
"""
Inventory Cost Reports Exporter
Exports inventory cost tracking data to CSV files for accounting review
"""

import argparse
import csv
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional
import requests


class InventoryReportExporter:
    """Exports inventory cost reports to CSV format"""

    def __init__(self, base_url: str, company_id: str, output_dir: str):
        self.base_url = base_url.rstrip("/")
        self.company_id = company_id
        self.output_dir = Path(output_dir)
        self.timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.headers = {"X-Company-ID": company_id}

        # Create output directory
        self.output_dir.mkdir(parents=True, exist_ok=True)

    def _get(self, endpoint: str, params: Optional[Dict] = None) -> Dict:
        """Make GET request to API"""
        url = f"{self.base_url}{endpoint}"
        try:
            response = requests.get(url, headers=self.headers, params=params or {})
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"❌ Error fetching {endpoint}: {e}")
            return {}

    def export_inventory_states(self) -> str:
        """Export current inventory states for all items"""
        print("📊 Exporting Report 1: Current Inventory States")

        filename = f"inventory_states_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        data = self._get("/v1/inventory/states")
        states = data.get("states", [])

        if not states:
            print("   ⚠️  No inventory states found")
            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(
                    [
                        "SKU",
                        "Item Name",
                        "Type",
                        "Quantity",
                        "Avg Cost",
                        "Total Value",
                        "Last Updated",
                    ]
                )
                writer.writerow(["No data available"])
        else:
            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(
                    [
                        "SKU",
                        "Item Name",
                        "Type",
                        "Quantity",
                        "Avg Cost",
                        "Total Value",
                        "Last Updated",
                    ]
                )

                for state in states:
                    item_type = (
                        "Product" if state.get("tipo_item") == "1" else "Service"
                    )
                    writer.writerow(
                        [
                            state.get("sku", ""),
                            state.get("item_name", ""),
                            item_type,
                            state.get("current_quantity", 0),
                            state.get("current_avg_cost", 0),
                            state.get("current_total_cost", 0),
                            state.get("updated_at", ""),
                        ]
                    )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_in_stock_items(self) -> str:
        """Export only items with stock > 0"""
        print("📊 Exporting Report 2: In-Stock Items")

        filename = f"inventory_in_stock_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        data = self._get("/v1/inventory/states", {"in_stock_only": "true"})
        states = data.get("states", [])

        if not states:
            print("   ⚠️  No in-stock items found")
            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(
                    [
                        "SKU",
                        "Item Name",
                        "Type",
                        "Quantity",
                        "Avg Cost",
                        "Total Value",
                        "Last Updated",
                    ]
                )
                writer.writerow(["No data available"])
        else:
            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(
                    [
                        "SKU",
                        "Item Name",
                        "Type",
                        "Quantity",
                        "Avg Cost",
                        "Total Value",
                        "Last Updated",
                    ]
                )

                for state in states:
                    item_type = (
                        "Product" if state.get("tipo_item") == "1" else "Service"
                    )
                    writer.writerow(
                        [
                            state.get("sku", ""),
                            state.get("item_name", ""),
                            item_type,
                            state.get("current_quantity", 0),
                            state.get("current_avg_cost", 0),
                            state.get("current_total_cost", 0),
                            state.get("updated_at", ""),
                        ]
                    )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_portfolio_summary(self) -> str:
        """Export portfolio summary metrics"""
        print("📊 Exporting Report 3: Portfolio Summary")

        filename = f"portfolio_summary_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        data = self._get("/v1/inventory/states")
        states = data.get("states", [])

        if not states:
            print("   ⚠️  No data for portfolio summary")
            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(["Metric", "Value"])
                writer.writerow(["Total Portfolio Value", 0])
                writer.writerow(["Total Items", 0])
        else:
            total_value = sum(s.get("current_total_cost", 0) for s in states)
            total_items = len(states)
            in_stock_items = sum(1 for s in states if s.get("current_quantity", 0) > 0)
            out_of_stock = total_items - in_stock_items

            with open(filepath, "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(["Metric", "Value"])
                writer.writerow(["Total Portfolio Value", f"{total_value:.2f}"])
                writer.writerow(["Total Items", total_items])
                writer.writerow(["Items In Stock", in_stock_items])
                writer.writerow(["Items Out of Stock", out_of_stock])

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def get_product_items(self) -> List[Dict]:
        """Get all product items (tipo_item = "1")"""
        data = self._get("/v1/inventory/items", {"active": "true"})
        items = data.get("items", [])
        return [item for item in items if item.get("tipo_item") == "1"]

    def export_cost_history(self) -> str:
        """Export complete cost history for all items"""
        print("📊 Exporting Report 4: Cost History (All Items)")

        filename = f"cost_history_all_items_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        items = self.get_product_items()

        with open(filepath, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(
                [
                    "Item ID",
                    "SKU",
                    "Item Name",
                    "Event Type",
                    "Timestamp",
                    "Quantity",
                    "Unit Cost",
                    "Total Cost",
                    "Avg Cost Before",
                    "Avg Cost After",
                    "Balance Qty After",
                    "Balance Value After",
                    "Notes",
                ]
            )

            if not items:
                print("   ⚠️  No product items found")
                writer.writerow(["No data available"])
            else:
                for item in items:
                    item_id = item.get("id")
                    sku = item.get("sku", "Unknown")
                    name = item.get("name", "Unknown")

                    # Use sort=asc for chronological order (oldest first)
                    history = self._get(
                        f"/v1/inventory/items/{item_id}/cost-history",
                        {"limit": "100", "sort": "asc"},
                    )
                    events = history.get("events", [])

                    for event in events:
                        writer.writerow(
                            [
                                item_id,
                                sku,
                                name,
                                event.get("event_type", ""),
                                event.get("event_timestamp", ""),
                                event.get("quantity", 0),
                                event.get("unit_cost", 0),
                                event.get("total_cost", 0),
                                event.get("moving_avg_cost_before", 0),
                                event.get("moving_avg_cost_after", 0),
                                event.get("balance_quantity_after", 0),
                                event.get("balance_total_cost_after", 0),
                                event.get("notes", ""),
                            ]
                        )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_purchase_events(self) -> str:
        """Export only purchase events"""
        print("📊 Exporting Report 5: Purchase Events")

        filename = f"purchase_events_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        items = self.get_product_items()

        with open(filepath, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(
                [
                    "Item ID",
                    "SKU",
                    "Item Name",
                    "Timestamp",
                    "Quantity",
                    "Unit Cost",
                    "Total Cost",
                    "Avg Cost After",
                    "Balance Qty After",
                    "Balance Value After",
                    "Notes",
                ]
            )

            if not items:
                print("   ⚠️  No product items found")
                writer.writerow(["No data available"])
            else:
                for item in items:
                    item_id = item.get("id")
                    sku = item.get("sku", "Unknown")
                    name = item.get("name", "Unknown")

                    history = self._get(
                        f"/v1/inventory/items/{item_id}/cost-history", {"limit": "100"}
                    )
                    events = history.get("events", [])

                    for event in events:
                        if event.get("event_type") == "PURCHASE":
                            writer.writerow(
                                [
                                    item_id,
                                    sku,
                                    name,
                                    event.get("event_timestamp", ""),
                                    event.get("quantity", 0),
                                    event.get("unit_cost", 0),
                                    event.get("total_cost", 0),
                                    event.get("moving_avg_cost_after", 0),
                                    event.get("balance_quantity_after", 0),
                                    event.get("balance_total_cost_after", 0),
                                    event.get("notes", ""),
                                ]
                            )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_adjustment_events(self) -> str:
        """Export only adjustment events"""
        print("📊 Exporting Report 6: Adjustment Events")

        filename = f"adjustment_events_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        items = self.get_product_items()

        with open(filepath, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(
                [
                    "Item ID",
                    "SKU",
                    "Item Name",
                    "Timestamp",
                    "Quantity",
                    "Unit Cost",
                    "Avg Cost Before",
                    "Avg Cost After",
                    "Cost Impact",
                    "Balance Qty After",
                    "Balance Value After",
                    "Reason",
                ]
            )

            if not items:
                print("   ⚠️  No product items found")
                writer.writerow(["No data available"])
            else:
                for item in items:
                    item_id = item.get("id")
                    sku = item.get("sku", "Unknown")
                    name = item.get("name", "Unknown")

                    history = self._get(
                        f"/v1/inventory/items/{item_id}/cost-history", {"limit": "100"}
                    )
                    events = history.get("events", [])

                    for event in events:
                        if event.get("event_type") == "ADJUSTMENT":
                            avg_before = event.get("moving_avg_cost_before", 0)
                            avg_after = event.get("moving_avg_cost_after", 0)
                            cost_impact = avg_after - avg_before

                            writer.writerow(
                                [
                                    item_id,
                                    sku,
                                    name,
                                    event.get("event_timestamp", ""),
                                    event.get("quantity", 0),
                                    event.get("unit_cost", 0),
                                    avg_before,
                                    avg_after,
                                    f"{cost_impact:.2f}",
                                    event.get("balance_quantity_after", 0),
                                    event.get("balance_total_cost_after", 0),
                                    event.get("notes", ""),
                                ]
                            )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_cost_analysis(self) -> str:
        """Export cost analysis by item"""
        print("📊 Exporting Report 7: Cost Analysis by Item")

        filename = f"cost_analysis_{self.timestamp}.csv"
        filepath = self.output_dir / filename

        items = self.get_product_items()

        with open(filepath, "w", newline="") as f:
            writer = csv.writer(f)
            writer.writerow(
                [
                    "SKU",
                    "Item Name",
                    "Total Purchases",
                    "Total Purchase Qty",
                    "Avg Purchase Cost",
                    "Total Adjustments",
                    "Total Adjustment Qty",
                    "Current Qty",
                    "Current Avg Cost",
                    "Current Total Value",
                ]
            )

            if not items:
                print("   ⚠️  No product items found")
                writer.writerow(["No data available"])
            else:
                for item in items:
                    item_id = item.get("id")
                    sku = item.get("sku", "Unknown")
                    name = item.get("name", "Unknown")

                    # Get cost history
                    history = self._get(
                        f"/v1/inventory/items/{item_id}/cost-history", {"limit": "100"}
                    )
                    events = history.get("events", [])

                    # Filter events
                    purchases = [e for e in events if e.get("event_type") == "PURCHASE"]
                    adjustments = [
                        e for e in events if e.get("event_type") == "ADJUSTMENT"
                    ]

                    # Calculate purchase metrics
                    total_purchases = len(purchases)
                    total_purchase_qty = sum(p.get("quantity", 0) for p in purchases)

                    if total_purchase_qty > 0:
                        weighted_cost = sum(
                            p.get("unit_cost", 0) * p.get("quantity", 0)
                            for p in purchases
                        )
                        avg_purchase_cost = weighted_cost / total_purchase_qty
                    else:
                        avg_purchase_cost = 0

                    # Calculate adjustment metrics
                    total_adjustments = len(adjustments)
                    total_adjustment_qty = sum(
                        a.get("quantity", 0) for a in adjustments
                    )

                    # Get current state
                    state = self._get(f"/v1/inventory/items/{item_id}/state")

                    writer.writerow(
                        [
                            sku,
                            name,
                            total_purchases,
                            total_purchase_qty,
                            f"{avg_purchase_cost:.2f}",
                            total_adjustments,
                            total_adjustment_qty,
                            state.get("current_quantity", 0),
                            state.get("current_avg_cost", 0),
                            state.get("current_total_cost", 0),
                        ]
                    )

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def create_manifest(self, files: List[str]) -> str:
        """Create a manifest file listing all exports"""
        print("📊 Creating Export Manifest")

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
                f.write(f"{i}. {Path(file).name}\n")

            f.write("\nUSAGE:\n")
            f.write("- Open CSV files in Excel or Google Sheets\n")
            f.write("- All monetary values are rounded to 2 decimal places\n")
            f.write("- Timestamps are in ISO 8601 format (UTC)\n")
            f.write("- Use filters and pivot tables for further analysis\n\n")

            f.write("NOTES:\n")
            f.write("- Product items (tipo_item = '1') include cost tracking\n")
            f.write("- Service items (tipo_item = '2') do not have inventory events\n")

        print(f"   ✅ Saved to: {filepath}\n")
        return str(filepath)

    def export_all(self):
        """Export all reports"""
        print("╔" + "=" * 60 + "╗")
        print("║" + " " * 15 + "INVENTORY COST REPORTS - CSV EXPORT" + " " * 10 + "║")
        print("╚" + "=" * 60 + "╝")
        print(f"Company ID: {self.company_id}")
        print(f"Output Directory: {self.output_dir}")
        print(f"Timestamp: {self.timestamp}\n")

        files = []

        try:
            files.append(self.export_inventory_states())
            files.append(self.export_in_stock_items())
            files.append(self.export_portfolio_summary())
            files.append(self.export_cost_history())
            files.append(self.export_purchase_events())
            files.append(self.export_adjustment_events())
            files.append(self.export_cost_analysis())
            files.append(self.create_manifest(files))

            print("=" * 63)
            print("✅ ALL REPORTS EXPORTED SUCCESSFULLY")
            print("=" * 63 + "\n")

            print("📁 Output Directory:", self.output_dir)
            print("📄 Files Generated:")
            for file in files:
                size = Path(file).stat().st_size
                print(f"   {Path(file).name} ({size} bytes)")

            print("\n💡 To view in Excel/Google Sheets:")
            print("   1. Open any CSV file")
            print("   2. Data will be properly formatted with headers")
            print("   3. Use filters and pivot tables for analysis\n")

        except Exception as e:
            print(f"\n❌ Error during export: {e}")
            sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Export inventory cost tracking reports to CSV files"
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

    args = parser.parse_args()

    exporter = InventoryReportExporter(
        base_url=args.base_url, company_id=args.company_id, output_dir=args.output_dir
    )

    exporter.export_all()


if __name__ == "__main__":
    main()
