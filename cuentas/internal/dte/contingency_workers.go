package dte

import (
	"context"
	"log"
	"time"
)

// StartContingencyWorkers starts all background workers for contingency processing
func (s *ContingencyService) StartContingencyWorkers(ctx context.Context) {
	log.Println("[Contingency] Starting background workers...")

	// Worker 1: Event Creator - runs every 10 minutes
	// Groups pending DTEs and creates contingency events
	go s.eventCreatorWorker(ctx)

	// Worker 2: Batch Submitter - runs every 5 minutes
	// Submits batches for accepted events
	go s.batchSubmitterWorker(ctx)

	// Worker 3: Batch Poller - runs every 2 minutes
	// Polls Hacienda for batch results
	go s.batchPollerWorker(ctx)

	log.Println("[Contingency] ✅ All workers started")
}

// eventCreatorWorker - Creates contingency events for pending DTEs
func (s *ContingencyService) eventCreatorWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

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
	log.Println("[EventCreator] Processing contingency events...")

	// Get all companies with pending DTEs
	companies, err := s.GetCompaniesWithPendingDTEs(ctx)
	if err != nil {
		log.Printf("[EventCreator] ❌ Error getting companies: %v", err)
		return
	}

	if len(companies) == 0 {
		log.Println("[EventCreator] No pending DTEs found")
		return
	}

	log.Printf("[EventCreator] Found %d companies with pending DTEs", len(companies))

	// Process each company
	for _, companyID := range companies {
		log.Printf("[EventCreator] Processing company: %s", companyID)

		// Get pending DTEs for this company
		dtes, err := s.GetPendingDTEsByCompany(ctx, companyID)
		if err != nil {
			log.Printf("[EventCreator] ❌ Error getting DTEs for company %s: %v", companyID, err)
			continue
		}

		if len(dtes) == 0 {
			continue
		}

		log.Printf("[EventCreator] Found %d pending DTEs for company %s", len(dtes), companyID)

		// Create and submit contingency event
		event, err := s.CreateAndSubmitContingencyEvent(ctx, companyID, dtes)
		if err != nil {
			log.Printf("[EventCreator] ❌ Failed to create event for company %s: %v", companyID, err)

			// Increment retry count for DTEs
			for _, dte := range dtes {
				s.incrementDTERetryCount(ctx, dte.ID)
			}

			continue
		}

		log.Printf("[EventCreator] ✅ Event created for company %s: %s", companyID, event.ID)
	}

	log.Println("[EventCreator] ✅ Processing complete")
}

// batchSubmitterWorker - Submits batches for accepted events
func (s *ContingencyService) batchSubmitterWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
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
	log.Println("[BatchSubmitter] Processing batch submissions...")

	// Get all accepted events without batches
	events, err := s.getEventsReadyForBatch(ctx)
	if err != nil {
		log.Printf("[BatchSubmitter] ❌ Error getting events: %v", err)
		return
	}

	if len(events) == 0 {
		log.Println("[BatchSubmitter] No events ready for batch submission")
		return
	}

	log.Printf("[BatchSubmitter] Found %d events ready for batch", len(events))

	for _, event := range events {
		log.Printf("[BatchSubmitter] Creating batch for event: %s", event.ID)

		batch, err := s.CreateAndSubmitBatch(ctx, event.ID)
		if err != nil {
			log.Printf("[BatchSubmitter] ❌ Failed to submit batch for event %s: %v", event.ID, err)
			continue
		}

		log.Printf("[BatchSubmitter] ✅ Batch submitted: %s (lote: %s)", batch.ID, batch.CodigoLote.String)
	}

	log.Println("[BatchSubmitter] ✅ Processing complete")
}

// batchPollerWorker - Polls for batch results
func (s *ContingencyService) batchPollerWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	// Wait 2 minutes before first run (give batches time to process)
	time.Sleep(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			log.Println("[BatchPoller] Shutting down...")
			return
		case <-ticker.C:
			err := s.PollBatches(ctx)
			if err != nil {
				log.Printf("[BatchPoller] ❌ Error: %v", err)
			}
		}
	}
}

// Helper methods

func (s *ContingencyService) incrementDTERetryCount(ctx context.Context, dteID string) error {
	query := `
        UPDATE dte_contingency_queue
        SET retry_count = retry_count + 1,
            updated_at = NOW()
        WHERE id = $1
    `

	_, err := s.db.ExecContext(ctx, query, dteID)
	return err
}

func (s *ContingencyService) getEventsReadyForBatch(ctx context.Context) ([]*EventInfo, error) {
	query := `
        SELECT e.id, e.codigo_generacion, e.company_id, e.ambiente
        FROM dte_contingency_events e
        WHERE e.status = 'accepted'
        AND NOT EXISTS (
            SELECT 1 FROM dte_contingency_batches b
            WHERE b.contingency_event_id = e.id
        )
        ORDER BY e.accepted_at ASC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventInfo
	for rows.Next() {
		var event EventInfo
		err := rows.Scan(
			&event.ID,
			&event.CodigoGeneracion,
			&event.CompanyID,
			&event.Ambiente,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	return events, nil
}

type EventInfo struct {
	ID               string
	CodigoGeneracion string
	CompanyID        string
	Ambiente         string
}
