El Salvador DTE Contingency System - Design Summary
What This System Does
When we can't send electronic invoices (DTEs) to Hacienda in real-time due to failures, this system tracks them, queues them, and automatically recovers them once services return. It follows Hacienda's official 3-step recovery process.

Three Failure Scenarios
1. POS Goes Offline

POS loses internet or power
Continues creating invoices locally with timestamps
When connectivity returns, syncs all invoices to our API
API detects old timestamps (>1 hour) and queues them for contingency

2. Firmador Service Down

POS connects to API successfully
API tries to sign the DTE but firmador is unavailable
Invoice stored unsigned, will retry signing later

3. Hacienda System Down

POS connects, invoice signed successfully
API tries to send to Hacienda but times out after 8 seconds
Retries 3 times with delays (2s, 5s)
If all fail, queues for contingency


Core Concept: The Contingency Period
When any failure happens, we create a contingency period for that specific POS. This period has:

Start time (when first failure occurred)
End time (when we closed it - set by worker)
Type of failure (1-5: Hacienda down, POS failure, internet, power, other)
Reason text
Status (active → reporting → completed)

Important: Only ONE active period per POS at a time. If firmador fails, then internet fails for the same POS, they share the same period.

Data Storage
Four tables work together:

invoices (modified existing table)

Links to period, event, and lote
Stores both unsigned and signed DTE
Tracks status: pending → pending_signature → contingency_queued → procesado/rechazado
Counts signature retry attempts


contingency_periods

One record per outage episode per POS
Stores time window and failure reason
Tracks if period is active, reporting, or completed


contingency_events

The report we send to Hacienda listing affected DTEs
One period can have multiple events (if >1000 invoices)
Stores Hacienda's seal number when accepted


lotes

Batches of up to 100 DTEs
One event creates multiple lotes (chunks of 100)
Tracks submission and polling status



Relationships: Period → Events → Lotes → Invoices

How Recovery Works
Step 1: Create Contingency Event
Worker runs every 10 minutes

Finds all active/reporting periods
For each period, gets up to 1000 invoices not yet in an event
If invoices are unsigned: Tries to sign them first (retry firmador)
Builds the Evento de Contingencia JSON with list of all DTEs
Signs the event and submits to Hacienda
Hacienda returns a seal number if accepted
Creates lotes (chunks of 100 invoices each)

Key behavior: If firmador is still down, unsigned invoices stay in period. Worker will try again in 10 minutes. Only signed invoices go into events.
Step 2: Submit Lotes
Worker runs every 5 minutes

Finds pending lotes
For each lote, builds JSON with the 100 signed DTEs
Submits to Hacienda
Hacienda returns a lote code and says "RECIBIDO" (received)
Marks lote as "submitted"

Step 3: Poll for Results
Same worker, every 5 minutes

Finds submitted lotes (not polled in last 3 minutes)
Asks Hacienda: "What's the status of lote XYZ?"
Hacienda returns results:

procesados: Invoices accepted (gives seal numbers)
rechazados: Invoices rejected (gives error reasons)


Updates each invoice status based on results
When all invoices in lote are finalized, marks lote complete
When all lotes in event are complete, checks if period should be completed


Concurrency Protection
Problem: Multiple worker instances could process the same period/lote at the same time.
Solution: Row-level locking with processing flag

Each table has a processing boolean column
Worker uses FOR UPDATE SKIP LOCKED to claim rows atomically
Sets processing = true before work
Always sets processing = false when done (even if error)
Other workers skip locked rows and pick different ones

Result: Zero duplicate processing, workers never step on each other.

Status Flow (Invoice Lifecycle)

