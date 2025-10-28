#!/usr/bin/env python3
"""
Download DTE Reconciliation CSV Report
"""

import requests
import sys
from datetime import datetime, timedelta
import argparse

BASE_URL = "http://localhost:8080"


def download_csv(company_id, output_file, **filters):
    """Download CSV reconciliation report"""

    headers = {
        "X-Company-ID": company_id,
        "Accept": "text/csv",  # This tells the server we want CSV
    }

    # Build query parameters
    params = {}
    if filters.get("start_date"):
        params["start_date"] = filters["start_date"]
    if filters.get("end_date"):
        params["end_date"] = filters["end_date"]
    if filters.get("date"):
        params["date"] = filters["date"]
    if filters.get("month"):
        params["month"] = filters["month"]
    if filters.get("codigo_generacion"):
        params["codigo_generacion"] = filters["codigo_generacion"]
    if filters.get("include_matches") is not None:
        params["include_matches"] = str(filters["include_matches"]).lower()

    # Add format parameter as backup
    params["format"] = "csv"

    url = f"{BASE_URL}/v1/dte/reconciliation"

    print(f"📥 Downloading CSV...")
    print(f"🔗 URL: {url}")
    print(f"🎯 Company ID: {company_id}")
    print(f"📋 Parameters: {params}")
    print(f"📤 Headers: {headers}")
    print()

    try:
        response = requests.get(url, headers=headers, params=params, timeout=120)

        print(f"📥 Response Status: {response.status_code}")
        print(f"📄 Content-Type: {response.headers.get('Content-Type', 'N/A')}")
        print()

        if response.status_code == 200:
            # Check if we actually got CSV
            content_type = response.headers.get("Content-Type", "")

            if "text/csv" not in content_type and "application/json" in content_type:
                print("⚠️  WARNING: Server returned JSON instead of CSV!")
                print("This might be a server configuration issue.")
                print()
                print("Response preview:")
                print(response.text[:500])
                return False

            # Save file
            with open(output_file, "wb") as f:
                f.write(response.content)

            # Get file size
            import os

            size = os.path.getsize(output_file)

            # Verify it's actually CSV
            with open(output_file, "r", encoding="utf-8") as f:
                first_line = f.readline()
                if first_line.startswith("{") or first_line.startswith("["):
                    print("❌ Error: File contains JSON, not CSV!")
                    print(f"First line: {first_line[:100]}")
                    return False

            # Count lines
            with open(output_file, "r", encoding="utf-8") as f:
                lines = f.readlines()
                # Find where data starts (after summary section)
                data_start = 0
                for i, line in enumerate(lines):
                    if "Código Generación" in line:
                        data_start = i + 1
                        break
                record_count = len(lines) - data_start if data_start > 0 else len(lines)

            print(f"✅ CSV downloaded successfully!")
            print(f"📄 File: {output_file}")
            print(f"📊 Size: {size:,} bytes")
            print(f"📋 Records: ~{record_count}")

            # Show first few lines
            print("\n" + "=" * 70)
            print("Preview (first 15 lines):")
            print("=" * 70)
            with open(output_file, "r", encoding="utf-8") as f:
                for i, line in enumerate(f):
                    if i >= 15:
                        break
                    print(line.rstrip())

            return True
        else:
            print(f"❌ Error: HTTP {response.status_code}")
            print(response.text[:500])
            return False

    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(
        description="Download DTE Reconciliation CSV Report",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # All DTEs
  %(prog)s COMPANY_ID all_dtes.csv
  
  # Date range
  %(prog)s COMPANY_ID october.csv --start-date 2025-10-01 --end-date 2025-10-31
  
  # Specific month
  %(prog)s COMPANY_ID october.csv --month 2025-10
  
  # Only mismatches
  %(prog)s COMPANY_ID mismatches.csv --mismatches-only
  
  # Last 30 days
  %(prog)s COMPANY_ID last_30_days.csv --last-days 30
        """,
    )

    parser.add_argument("company_id", help="Company UUID")
    parser.add_argument(
        "output_file",
        nargs="?",
        default="reconciliation.csv",
        help="Output CSV filename (default: reconciliation.csv)",
    )

    # Date filters
    parser.add_argument("--start-date", help="Start date (YYYY-MM-DD)")
    parser.add_argument("--end-date", help="End date (YYYY-MM-DD)")
    parser.add_argument("--date", help="Specific date (YYYY-MM-DD)")
    parser.add_argument("--month", help="Specific month (YYYY-MM)")
    parser.add_argument("--last-days", type=int, help="Last N days")

    # Other filters
    parser.add_argument("--codigo-generacion", help="Specific DTE codigo")
    parser.add_argument(
        "--mismatches-only",
        action="store_true",
        help="Only include DTEs with discrepancies",
    )

    args = parser.parse_args()

    # Build filters
    filters = {}

    if args.last_days:
        today = datetime.now()
        start = (today - timedelta(days=args.last_days)).strftime("%Y-%m-%d")
        end = today.strftime("%Y-%m-%d")
        filters["start_date"] = start
        filters["end_date"] = end
    else:
        if args.start_date:
            filters["start_date"] = args.start_date
        if args.end_date:
            filters["end_date"] = args.end_date
        if args.date:
            filters["date"] = args.date
        if args.month:
            filters["month"] = args.month

    if args.codigo_generacion:
        filters["codigo_generacion"] = args.codigo_generacion

    if args.mismatches_only:
        filters["include_matches"] = False

    # Download
    success = download_csv(args.company_id, args.output_file, **filters)

    if success:
        print(f"\n✅ Done! Open with:")
        print(f"   open {args.output_file}")
        print(f"   # or")
        print(f"   libreoffice {args.output_file}")
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
