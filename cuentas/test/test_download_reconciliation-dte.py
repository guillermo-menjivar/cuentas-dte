#!/usr/bin/env python3
"""
Download DTE Reconciliation CSV Report
"""

import requests
import sys
from datetime import datetime

BASE_URL = "http://localhost:8080"


def download_csv(company_id, filename="dte_reconciliation.csv", **params):
    """Download CSV report"""

    headers = {"X-Company-ID": company_id, "Accept": "text/csv"}

    # Build query parameters
    query_params = []
    for key, value in params.items():
        if value:
            query_params.append(f"{key}={value}")

    query_string = "&".join(query_params)
    url = f"{BASE_URL}/v1/dte/reconciliation"
    if query_string:
        url += f"?{query_string}"

    print(f"📥 Downloading CSV from: {url}")

    try:
        response = requests.get(url, headers=headers, timeout=60)

        if response.status_code == 200:
            with open(filename, "wb") as f:
                f.write(response.content)
            print(f"✅ CSV saved to: {filename}")

            # Show file size
            size = len(response.content)
            print(f"📊 File size: {size:,} bytes")

            # Count lines
            lines = response.text.count("\n")
            print(f"📋 Total lines: {lines:,}")

            return True
        else:
            print(f"❌ Error: HTTP {response.status_code}")
            print(response.text)
            return False

    except Exception as e:
        print(f"❌ Failed to download: {e}")
        return False


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage:")
        print("  ./download_csv.py <company_id>")
        print(
            "  ./download_csv.py <company_id> --start-date 2025-10-01 --end-date 2025-10-31"
        )
        print("  ./download_csv.py <company_id> --month 2025-10")
        print("  ./download_csv.py <company_id> --mismatches-only")
        sys.exit(1)

    company_id = sys.argv[1]

    # Parse additional arguments
    params = {}
    filename = f"dte_reconciliation_{datetime.now().strftime('%Y%m%d_%H%M%S')}.csv"

    args = sys.argv[2:]
    i = 0
    while i < len(args):
        if args[i] == "--start-date" and i + 1 < len(args):
            params["start_date"] = args[i + 1]
            i += 2
        elif args[i] == "--end-date" and i + 1 < len(args):
            params["end_date"] = args[i + 1]
            i += 2
        elif args[i] == "--month" and i + 1 < len(args):
            params["month"] = args[i + 1]
            filename = f"dte_reconciliation_{args[i + 1]}.csv"
            i += 2
        elif args[i] == "--date" and i + 1 < len(args):
            params["date"] = args[i + 1]
            filename = f"dte_reconciliation_{args[i + 1]}.csv"
            i += 2
        elif args[i] == "--mismatches-only":
            params["include_matches"] = "false"
            filename = (
                f"dte_reconciliation_mismatches_{datetime.now().strftime('%Y%m%d')}.csv"
            )
            i += 1
        else:
            i += 1

    # Update filename if date range specified
    if "start_date" in params and "end_date" in params:
        filename = (
            f"dte_reconciliation_{params['start_date']}_to_{params['end_date']}.csv"
        )

    download_csv(company_id, filename, **params)
