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
            print("📊 SUMMARY:")
            print(f"  Total Records:          {summary.get('total_records', 0)}")
            print(f"  ✅ Matched:             {summary.get('matched_records', 0)}")
            print(f"  ❌ Mismatched:          {summary.get('mismatched_records', 0)}")
            print(f"  📅 Date Mismatches:     {summary.get('date_mismatches', 0)}")
            print(
                f"  🔍 Not Found (H):       {summary.get('not_found_in_hacienda', 0)}"
            )
            print(f"  ⚠️  Query Errors:        {summary.get('query_errors', 0)}")
            print()

        # Print results
        if "results" in data:
            results = data["results"]
            print(f"📋 RESULTS: {len(results)} DTEs")
            print()

            for i, dte in enumerate(results[:5], 1):  # Show first 5
                codigo = dte.get("codigo_generacion", "N/A")
                print(f"[{i}] {codigo[:13] if len(codigo) > 13 else codigo}...")
                print(f"    📄 Número Control:      {dte.get('numero_control', 'N/A')}")
                print(f"    🧾 Factura:             {dte.get('invoice_number', 'N/A')}")
                print(f"    📅 Fecha Emisión:       {dte.get('fecha_emision', 'N/A')}")
                print(f"    💰 Monto:               ${dte.get('total_amount', 0):.2f}")

                # Estado interno (nullable)
                estado_interno = dte.get("internal_estado")
                print(
                    f"    📌 Estado Interno:      {estado_interno if estado_interno else 'N/A'}"
                )

                # Estado Hacienda (may be empty string)
                estado_hacienda = dte.get("hacienda_estado", "")
                print(
                    f"    🏛️  Estado Hacienda:     {estado_hacienda if estado_hacienda else 'N/A'}"
                )

                # Match status
                matches = dte.get("matches", False)
                fecha_matches = dte.get("fecha_emision_matches", False)
                print(f"    {'✅' if matches else '❌'} Coincide:            {matches}")
                print(
                    f"    {'✅' if fecha_matches else '❌'} Fecha Coincide:      {fecha_matches}"
                )

                # Query status
                query_status = dte.get("hacienda_query_status", "unknown")
                status_icon = {"success": "✅", "not_found": "🔍", "error": "⚠️"}.get(
                    query_status, "❓"
                )
                print(f"    {status_icon} Estado Consulta:     {query_status}")

                # Discrepancies (may not exist if omitempty and empty)
                discrepancies = dte.get("discrepancies")
                if discrepancies:  # Only show if exists and not empty
                    print(f"    ⚠️  Discrepancias:")
                    for disc in discrepancies:
                        print(f"        • {disc}")

                # Error message (may not exist if omitempty and empty)
                error_msg = dte.get("error_message")
                if error_msg:  # Only show if exists and not empty
                    print(f"    🚨 Error: {error_msg}")

                print()

            if len(results) > 5:
                print(f"... and {len(results) - 5} more DTEs")
                print()

        elif isinstance(data, dict) and "codigo_generacion" in data:
            # Single DTE result
            codigo = data.get("codigo_generacion", "N/A")
            print("📋 SINGLE DTE RESULT:")
            print(f"  Código Generación:   {codigo}")
            print(f"  📄 Número Control:      {data.get('numero_control', 'N/A')}")
            print(f"  🧾 Factura:             {data.get('invoice_number', 'N/A')}")
            print(f"  📅 Fecha Emisión:       {data.get('fecha_emision', 'N/A')}")
            print(f"  💰 Monto:               ${data.get('total_amount', 0):.2f}")

            estado_interno = data.get("internal_estado")
            print(
                f"  📌 Estado Interno:      {estado_interno if estado_interno else 'N/A'}"
            )

            estado_hacienda = data.get("hacienda_estado", "")
            print(
                f"  🏛️  Estado Hacienda:     {estado_hacienda if estado_hacienda else 'N/A'}"
            )

            matches = data.get("matches", False)
            fecha_matches = data.get("fecha_emision_matches", False)
            print(f"  {'✅' if matches else '❌'} Coincide:            {matches}")
            print(
                f"  {'✅' if fecha_matches else '❌'} Fecha Coincide:      {fecha_matches}"
            )

            query_status = data.get("hacienda_query_status", "unknown")
            status_icon = {"success": "✅", "not_found": "🔍", "error": "⚠️"}.get(
                query_status, "❓"
            )
            print(f"  {status_icon} Estado Consulta:     {query_status}")

            discrepancies = data.get("discrepancies")
            if discrepancies:
                print(f"  ⚠️  Discrepancias:")
                for disc in discrepancies:
                    print(f"      • {disc}")

            error_msg = data.get("error_message")
            if error_msg:
                print(f"  🚨 Error: {error_msg}")
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
