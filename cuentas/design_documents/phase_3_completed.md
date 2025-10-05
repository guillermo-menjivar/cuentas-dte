Phase 2 Invoice System - Current Status Documentation
What We've Built
A complete, production-ready invoice transaction system for El Salvador with immutable snapshots, full audit trails, and准备 for DTE (government electronic invoicing) integration.

Database Schema (Migrations 0008-0012)
0008: Client Credit Tracking
Added to clients table:

credit_limit - Maximum credit allowed per client
current_balance - Running balance of what client owes
credit_status - Account standing (good_standing, over_limit, suspended)

0009: Invoices Table
Core invoice header with:

Identification: invoice_number (INV-YYYY-#####), invoice_type (sale/credit_note/debit_note)
Client Snapshot: Complete client data captured at transaction time (name, NIT, NCR, DUI, address, tipo_contribuyente, tipo_persona)
Financial Totals: subtotal, total_discount, total_taxes, total (all rounded to 2 decimals)
Payment Tracking: payment_terms, payment_status, amount_paid, balance_due, due_date
Status Flow: draft → finalized → void
DTE Fields: dte_codigo_generacion, dte_numero_control, dte_status, dte_hacienda_response
Audit Trail: created_at, finalized_at, voided_at, created_by, voided_by

Key Constraint: Finalized invoices MUST have DTE codigo (enforced by database)
0010: Invoice Line Items
Detailed line-level data:

Item Snapshot: SKU, name, description, tipo_item, unit_of_measure, unit_price (at transaction time)
Calculations: quantity, line_subtotal, discount_percentage, discount_amount
Tax Base: taxable_amount, total_taxes, line_total
Reference: item_id (for tracking only, NOT for pricing)

0011: Invoice Line Item Taxes
Tax detail per line item:

Tax Snapshot: tributo_code, tributo_name, tax_rate (at transaction time)
Calculations: taxable_base, tax_amount
Data comes from Go codigos package, NOT from database

0012: Invoice Payments
Payment tracking:

Multiple payments per invoice supported
payment_method, amount, reference_number, payment_date
Updates invoice.amount_paid and invoice.balance_due

0013: Consumidor Final Auto-Creation

Automatic "Consumidor Final" client created for every company
Trigger creates it when company is registered
NIT: 9999999999999, NCR: 999999, tipo_persona: "1"


Models (internal/models/)
Invoice Models

Invoice - Complete invoice with all fields
CreateInvoiceRequest - API request with client_id, line_items[], payment_terms, contact info
InvoiceLineItem - Line item with item snapshot and calculations
InvoiceLineItemTax - Tax detail with rate snapshot
InvoicePayment - Payment record

Key Features:

Validation on all requests
Proper Go types (pointers for nullable fields)
JSON tags for API responses
Contact fields (contact_email, contact_whatsapp) for overriding client data


Service Layer (internal/services/invoice.go)
Core Functions
CreateInvoice - Creates draft invoice with complete snapshots:

Validates request
Snapshots client data (name, tax IDs, address at that moment)
Generates sequential invoice_number (INV-YYYY-#####)
For each line item:

Snapshots inventory item (price, name, description at that moment)
Snapshots tax rates from codigos package
Calculates all amounts with proper rounding


Saves atomically (invoice → line_items → taxes)
Returns complete invoice with all relationships

GetInvoice - Retrieves complete invoice with line items and taxes
ListInvoices - Lists invoices with filters (status, client_id, payment_status)
DeleteDraftInvoice - Deletes only draft invoices (finalized cannot be deleted)
Helper Functions
snapshotClient - Captures all client fields at transaction time
snapshotInventoryItem - Captures item price and details at transaction time
snapshotItemTaxes - Gets tax codes from inventory_item_taxes, looks up names/rates in Go codigos package, calculates tax amounts
generateInvoiceNumber - Sequential per company, format: INV-YYYY-#####
round() - Rounds all monetary values to 2 decimals (fixes floating-point precision)

API Endpoints (internal/handlers/invoice_handlers.go)
POST   /v1/invoices          Create draft invoice
GET    /v1/invoices          List invoices (filters: status, client_id, payment_status)
GET    /v1/invoices/:id      Get invoice with full details
DELETE /v1/invoices/:id      Delete draft invoice only
All endpoints require X-Company-ID header (set by middleware)

Key Design Decisions
1. Complete Snapshots
Every invoice captures:

Client data (can change over time)
Item prices (can change over time)
Tax rates (can change over time)
Tax names (can change over time)

Why: Historical invoices remain accurate forever, regardless of future changes
2. Immutability

Draft invoices: Can be edited (delete + recreate) or deleted
Finalized invoices: Cannot be edited or deleted
Corrections: Void original + create new invoice
Full audit trail preserved

3. No Database for Tax Rates
Tax codes stored in inventory_item_taxes table, but names and rates come from Go codigos package

Simpler (no need to sync database with government changes)
Single source of truth in code
Easy to version control

4. Rounding
All monetary calculations use math.Round(val*100)/100 to prevent floating-point precision issues

Before: 3.2205000000000004
After: 3.22

5. Consumidor Final Support

Special client auto-created per company
Optional contact_email/contact_whatsapp on invoice for delivery
Supports retail/cash transactions without full client registration


Data Flow Example
Creating an Invoice:
User Request → Validate
            → Snapshot Client (from clients table)
            → Generate Invoice Number
            → For Each Line Item:
              → Snapshot Item (price from inventory_items)
              → Get Tax Codes (from inventory_item_taxes)
              → Get Tax Names/Rates (from codigos package)
              → Calculate: subtotal, discount, taxes, total
              → Round all amounts
            → Insert Invoice Header
            → Insert Line Items
            → Insert Taxes
            → Commit Transaction
            → Return Complete Invoice
What's Stored:

Invoice header with client snapshot and totals
Line items with item snapshot and calculations
Taxes with code, name, rate snapshot and amounts
Everything immutable once finalized


Testing Scripts (scripts/)

test_create_invoice.sh - Create invoice with items
test_get_invoice.sh - Retrieve invoice by ID
test_list_invoices.sh - List with filters
test_delete_invoice.sh - Delete draft
test_list_clients.sh - Find clients
test_list_inventory.sh - Find inventory items
test_find_consumidor_final.sh - Get consumidor final client ID
setup_test_data.sh - Create complete test scenario (client, item, tax, invoice)


What's Working

✅ Draft invoice creation with complete snapshots
✅ Sequential invoice numbering per company
✅ Client data snapshot (including tipo_persona validation)
✅ Item price snapshot
✅ Tax rate snapshot from codigos package
✅ Line-level discounts
✅ Multi-item invoices
✅ Proper monetary rounding (2 decimals)
✅ Consumidor final auto-creation
✅ Optional contact override (email/whatsapp)
✅ Payment terms (cash, net_30, net_60, cuenta)
✅ Full API with filters
✅ Complete audit trail


What's NOT Built Yet
Invoice Operations:

FinalizeInvoice (lock invoice, generate DTE, deduct stock)
VoidInvoice (create credit note, reverse balances)
RecordPayment (update balances, payment status)

DTE Integration (Phase 3):

Generate DTE JSON from invoice
Submit to Ministerio de Hacienda API
Handle contingency mode
Store DTE responses
Track DTE status changes
Generate signed PDF

Reporting:

Sales reports
Tax reports (IVA collected)
Aging reports
Client statements
Export formats (CSV, Excel, PDF)

Credit Management:

Check credit limit before finalizing
Update client.current_balance on finalize
Overdue invoice detection


Next Steps for DTE Integration

Create dte_submissions table for detailed DTE tracking
Build DTE JSON generator from invoice data
Implement FinalizeInvoice function
Add DTE API client (submit to Hacienda)
Handle DTE responses and status updates
Build contingency mode support
Generate PDF from DTE


Database State
Tables: clients (with credit tracking), invoices, invoice_line_items, invoice_line_item_taxes, invoice_payments
Indexes: Optimized for common queries (company_id, status, dates, invoice_number)
Constraints: Enforce data integrity (finalized must have DTE, NIT requires NCR, etc.)
Triggers: Auto-create consumidor final client on company registration

This system is production-ready for draft invoices and fully prepared for DTE integration. The snapshot architecture ensures historical accuracy, and the immutable design guarantees audit compliance.
