User finalizes invoice
  ↓
InvoiceService.FinalizeInvoice()
  1. Update invoice status = 'finalized'
  2. Save payment info
  ↓
DTEService.ProcessInvoice() [BLOCKING/SYNC]
  ↓
  Step 1: Build DTE from invoice
  Step 2: Sign with Firmador (3 retries built-in)
  Step 3: Transmit to Hacienda (3 retries built-in)
  ↓
  SUCCESS PATH:
    - Update invoice with DTE info
    - Log transaction (status: 'accepted')
    - Return to user ✅
  ↓
  FAILURE PATH (after 3 retries):
    - Log transaction (status: 'failed_retry_queue')
    - Update invoice (dte_status: 'pending_retry')
    - Still return invoice to user ✅
    - Background worker will retry later
