#!/usr/bin/env python3
"""
Test DTE Reconciliation API
"""

import requests
import sys
import json
from datetime import datetime, timedelta

BASE_URL = "http://localhost:8080"


def test_reconciliation(company_id, test_type="all", **kwargs):
    """Test different reconciliation scenarios"""

    headers = {"X-Company-ID": company_id}

    print("=" * 70)
    print(f"Testing DTE Reconciliation: {test_type}")
    print("=" * 70)

    if test_type == "all":
        # Get all DTEs
        url = f"{BASE_URL}/v1/dte/reconciliation"
        print(f"GET {url}")

    elif test_type == "date_range":
        # Get DTEs in date range
        start_date = kwargs.get("start_date")
        end_date = kwargs.get("end_date")
        url = f"{BASE_URL}/v1/dte/reconciliation?start_date={start_date}&end_date={end_date}"
        print(f"GET {url}")

    elif test_type == "single_date":
        # Get DTEs for specific date
        date = kwargs.get("date")
        url = f"{BASE_URL}/v1/dte/reconciliation?date={date}"
        print(f"GET {url}")

    elif test_type == "month":
        # Get DTEs for specific month
        month = kwargs.get("month")
        url = f"{BASE_URL}/v1/dte/reconciliation?month={month}"
        print(f"GET {url}")

    elif test_type == "single":
        # Get single DTE
        codigo = kwargs.get("codigo_generacion")
        url = f"{BASE_URL}/v1/dte/reconciliation/{codigo}"
        print(f"GET {url}")

    elif test_type == "mismatches_only":
        # Get only mismatches
        url = f"{BASE_URL}/v1/dte/reconciliation?include_matches=false"
        print(f"GET {url}")

    else:
        print(f"Unknown test type: {test_type}")
        return

    # Make request
    try:
        response = requests.get(url, headers=headers, timeout=30)
    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
        print()
        return

    print(f"Status: {response.status_code}")
    print()

    if response.status_code == 200:
        data = response.json()

        # Print summary if present
        if "summary" in data:
            summary = data["summary"]
            print("SUMMARY:")
            print(f"  Total Records:          {summary['total_records']}")
            print(f"  Matched:                {summary['matched_records']}")
            print(f"  Mismatched:             {summary['mismatched_records']}")
            print(f"  Date Mismatches:        {summary['date_mismatches']}")
            print(f"  Not Found in Hacienda:  {summary['not_found_in_hacienda']}")
            print(f"  Query Errors:           {summary['query_errors']}")
            print()

        # Print results
        if "results" in data:
            results = data["results"]
            print(f"RESULTS: {len(results)} DTEs")
            print()

            for i, dte in enumerate(results[:5], 1):  # Show first 5
                print(f"[{i}] {dte['codigo_generacion'][:8]}...")
                print(f"    Número Control:      {dte['numero_control']}")
                print(f"    Factura:             {dte['invoice_number']}")
                print(f"    Fecha Emisión:       {dte['fecha_emision']}")
                print(f"    Estado Interno:      {dte.get('internal_estado', 'N/A')}")
                print(f"    Estado Hacienda:     {dte.get('hacienda_estado', 'N/A')}")
                print(f"    Coincide:            {'✅' if dte['matches'] else '❌'}")
                print(
                    f"    Fecha Coincide:      {'✅' if dte['fecha_emision_matches'] else '❌'}"
                )
                print(f"    Estado Consulta:     {dte['hacienda_query_status']}")

                if dte["discrepancies"]:
                    print(f"    Discrepancias:")
                    for disc in dte["discrepancies"]:
                        print(f"      - {disc}")

                if dte.get("error_message"):
                    print(f"    Error: {dte['error_message']}")

                print()

            if len(results) > 5:
                print(f"... and {len(results) - 5} more DTEs")

        elif isinstance(data, dict) and "codigo_generacion" in data:
            # Single DTE result
            print("SINGLE DTE RESULT:")
            print(f"  Código Generación:   {data['codigo_generacion']}")
            print(f"  Número Control:      {data['numero_control']}")
            print(f"  Factura:             {data['invoice_number']}")
            print(f"  Fecha Emisión:       {data['fecha_emision']}")
            print(f"  Estado Interno:      {data.get('internal_estado', 'N/A')}")
            print(f"  Estado Hacienda:     {data.get('hacienda_estado', 'N/A')}")
            print(f"  Coincide:            {'✅' if data['matches'] else '❌'}")
            print(
                f"  Fecha Coincide:      {'✅' if data['fecha_emision_matches'] else '❌'}"
            )
            print(f"  Estado Consulta:     {data['hacienda_query_status']}")

            if data["discrepancies"]:
                print(f"  Discrepancias:")
                for disc in data["discrepancies"]:
                    print(f"    - {disc}")

            if data.get("error_message"):
                print(f"  Error: {data['error_message']}")
    else:
        print(f"❌ ERROR: {response.text}")

    print()


def test_csv_export(company_id, start_date, end_date):
    """Test CSV export"""
    print("=" * 70)
    print("Testing CSV Export")
    print("=" * 70)

    headers = {"X-Company-ID": company_id, "Accept": "text/csv"}

    url = (
        f"{BASE_URL}/v1/dte/reconciliation?start_date={start_date}&end_date={end_date}"
    )
    print(f"GET {url}")

    try:
        response = requests.get(url, headers=headers, timeout=30)
    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
        print()
        return

    print(f"Status: {response.status_code}")

    if response.status_code == 200:
        filename = f"dte_reconciliation_{start_date}_to_{end_date}.csv"
        with open(filename, "wb") as f:
            f.write(response.content)
        print(f"✅ CSV saved to: {filename}")

        # Show first few lines
        with open(filename, "r", encoding="utf-8") as f:
            lines = f.readlines()[:10]
            print("\nFirst 10 lines of CSV:")
            print("=" * 70)
            for line in lines:
                print(line.rstrip())
    else:
        print(f"❌ ERROR: {response.text}")

    print()


def get_sample_codigo(company_id):
    """Get a sample codigo_generacion from the database"""
    headers = {"X-Company-ID": company_id}
    url = f"{BASE_URL}/v1/dte/commit-log"

    try:
        response = requests.get(url, headers=headers, timeout=30)
        if response.status_code == 200:
            data = response.json()
            if data and len(data) > 0:
                return data[0].get("codigo_generacion")
    except:
        pass

    return None


def main():
    if len(sys.argv) < 2:
        print("Usage: ./test_dte_reconciliation.py <company_id>")
        print("\nExample:")
        print("  ./test_dte_reconciliation.py $(cat .company_id)")
        sys.exit(1)

    company_id = sys.argv[1]

    print("\n")
    print("🔍 DTE RECONCILIATION TEST SUITE")
    print("=" * 70)
    print(f"Company ID: {company_id}")
    print("=" * 70)
    print()

    # Test 1: Get all DTEs
    print("TEST 1: Get all DTEs")
    test_reconciliation(company_id, "all")

    # Test 2: Date range (last 30 days)
    print("TEST 2: Date range (last 30 days)")
    today = datetime.now()
    start = (today - timedelta(days=30)).strftime("%Y-%m-%d")
    end = today.strftime("%Y-%m-%d")
    test_reconciliation(company_id, "date_range", start_date=start, end_date=end)

    # Test 3: Specific date (today)
    print("TEST 3: Specific date (today)")
    test_reconciliation(company_id, "single_date", date=today.strftime("%Y-%m-%d"))

    # Test 4: Specific month (current month)
    print("TEST 4: Specific month (current month)")
    test_reconciliation(company_id, "month", month=today.strftime("%Y-%m"))

    # Test 5: Mismatches only
    print("TEST 5: Mismatches only")
    test_reconciliation(company_id, "mismatches_only")

    # Test 6: Single DTE (if available)
    print("TEST 6: Single DTE")
    sample_codigo = get_sample_codigo(company_id)
    if sample_codigo:
        test_reconciliation(company_id, "single", codigo_generacion=sample_codigo)
    else:
        print("⚠️  No DTEs found to test single DTE reconciliation")
        print()

    # Test 7: CSV export
    print("TEST 7: CSV export")
    test_csv_export(company_id, start, end)

    print()
    print("=" * 70)
    print("✅ All tests completed!")
    print("=" * 70)
    print()


if __name__ == "__main__":
    main()
