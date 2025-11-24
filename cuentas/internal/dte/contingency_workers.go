package dte

import (
	"context"
	"log"
	"time"
)

// StartContingencyWorkers starts all background workers for contingency processing
// These workers implement the 3-step process required by Ministerio de Hacienda:
// 1. Create and submit contingency events
// 2. Submit batches of DTEs for accepted events
// 3. Poll for batch processing results
func (s *ContingencyService) StartContingencyWorkers(ctx context.Context) {
	log.Println("[Contingency] 🚀 Starting background workers...")

	// Worker 1: Event Creator - runs every 10 minutes
	// Groups pending DTEs by company and creates contingency events
	go s.eventCreatorWorker(ctx)

	// Worker 2: Batch Submitter - runs every 5 minutes
	// Submits batches of DTEs for accepted events
	go s.batchSubmitterWorker(ctx)

	// Worker 3: Batch Poller - runs every 2 minutes
	// Polls Hacienda for batch processing results
	go s.batchPollerWorker(ctx)

	log.Println("[Contingency] ✅ All workers started successfully")
	log.Println("[Contingency] Event Creator: every 10 minutes")
	log.Println("[Contingency] Batch Submitter: every 5 minutes")
	log.Println("[Contingency] Batch Poller: every 2 minutes")
}

// ===========================
// WORKER 1: EVENT CREATOR
// ===========================

// eventCreatorWorker creates contingency events for pending DTEs
func (s *ContingencyService) eventCreatorWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	log.Println("[EventCreator] Worker started")

	// Run immediately on start
	s.processContingencyEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[EventCreator] Shutting down...")
			return
		case <-ticker.C:
			s.processContingencyEvents(ctx)
		}
	}
}

func (s *ContingencyService) processContingencyEvents(ctx context.Context) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("[EventCreator] 🔄 Processing contingency events...")

	// Get all companies with pending DTEs
	companies, err := s.GetCompaniesWithPendingDTEs(ctx)
	if err != nil {
		log.Printf("[EventCreator] ❌ Error getting companies: %v", err)
		return
	}

	if len(companies) == 0 {
		log.Println("[EventCreator] ℹ️  No pending DTEs found")
		return
	}

	log.Printf("[EventCreator] Found %d companies with pending DTEs", len(companies))

	successCount := 0
	errorCount := 0

	// Process each company
	for _, companyID := range companies {
		log.Printf("[EventCreator] Processing company: %s", companyID)

		// Get pending DTEs for this company
		dtes, err := s.GetPendingDTEsByCompany(ctx, companyID)
		if err != nil {
			log.Printf("[EventCreator] ❌ Error getting DTEs for company %s: %v", companyID, err)
			errorCount++
			continue
		}

		if len(dtes) == 0 {
			continue
		}

		log.Printf("[EventCreator] Found %d pending DTEs for company %s", len(dtes), companyID)

		// Create and submit contingency event (STEP 1)
		event, err := s.CreateAndSubmitContingencyEvent(ctx, companyID, dtes)
		if err != nil {
			log.Printf("[EventCreator] ❌ Failed to create event for company %s: %v", companyID, err)

			// Increment retry count for DTEs
			for _, dte := range dtes {
				if retryErr := s.incrementDTERetryCount(ctx, dte.ID); retryErr != nil {
					log.Printf("[EventCreator] ⚠️  Failed to increment retry for DTE %s: %v", dte.ID, retryErr)
				}
			}

			errorCount++
			continue
		}

		log.Printf("[EventCreator] ✅ Event created for company %s: %s", companyID, event.ID)
		successCount++
	}

	log.Printf("[EventCreator] ✅ Processing complete: %d succeeded, %d errors", successCount, errorCount)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ===========================
// WORKER 2: BATCH SUBMITTER
// ===========================

// batchSubmitterWorker submits batches for accepted events
func (s *ContingencyService) batchSubmitterWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("[BatchSubmitter] Worker started")

	// Run immediately on start (after 30 second delay to allow events to be created)
	time.Sleep(30 * time.Second)
	s.processBatchSubmissions(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[BatchSubmitter] Shutting down...")
			return
		case <-ticker.C:
			s.processBatchSubmissions(ctx)
		}
	}
}

func (s *ContingencyService) processBatchSubmissions(ctx context.Context) {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("[BatchSubmitter] 🔄 Processing batch submissions...")

	// Get all accepted events without batches
	events, err := s.getEventsReadyForBatch(ctx)
	if err != nil {
		log.Printf("[BatchSubmitter] ❌ Error getting events: %v", err)
		return
	}

	if len(events) == 0 {
		log.Println("[BatchSubmitter] ℹ️  No events ready for batch submission")
		return
	}

	log.Printf("[BatchSubmitter] Found %d events ready for batch", len(events))

	successCount := 0
	errorCount := 0

	for _, event := range events {
		log.Printf("[BatchSubmitter] Creating batch for event: %s", event.ID)

		// Create and submit batch (STEP 2)
		batch, err := s.CreateAndSubmitBatch(ctx, event.ID)
		if err != nil {
			log.Printf("[BatchSubmitter] ❌ Failed to submit batch for event %s: %v", event.ID, err)
			errorCount++
			continue
		}

		log.Printf("[BatchSubmitter] ✅ Batch submitted: %s (lote: %s)", batch.ID, batch.CodigoLote.String)
		successCount++
	}

	log.Printf("[BatchSubmitter] ✅ Processing complete: %d succeeded, %d errors", successCount, errorCount)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// ===========================
// WORKER 3: BATCH POLLER
// ===========================

// batchPollerWorker polls for batch results
func (s *ContingencyService) batchPollerWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	log.Println("[BatchPoller] Worker started")

	// Wait 2 minutes before first run (give batches time to process)
	// According to manual: 2-3 minutes for test, 1-3 minutes for production
	time.Sleep(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			log.Println("[BatchPoller] Shutting down...")
			return
		case <-ticker.C:
			log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			log.Println("[BatchPoller] 🔄 Polling batches...")

			err := s.PollBatches(ctx)
			if err != nil {
				log.Printf("[BatchPoller] ❌ Error: %v", err)
			}

			log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		}
	}
}
