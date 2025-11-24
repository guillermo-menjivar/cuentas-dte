package dte

import (
	"context"
	"log"
	"time"
)

// StartContingencyWorkers starts all background workers for contingency processing
func (s *ContingencyService) StartContingencyWorkers(ctx context.Context) {
	log.Println("[Contingency] 🚀 Starting background workers...")

	go s.eventCreatorWorker(ctx)
	go s.batchSubmitterWorker(ctx)
	go s.batchPollerWorker(ctx)

	log.Println("[Contingency] ✅ All workers started")
}

// eventCreatorWorker - creates contingency events for pending DTEs
func (s *ContingencyService) eventCreatorWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

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

	companies, err := s.GetCompaniesWithPendingDTEs(ctx)
	if err != nil {
		log.Printf("[EventCreator] ❌ Error getting companies: %v", err)
		return
	}

	if len(companies) == 0 {
		log.Println("[EventCreator] No pending DTEs found")
		return
	}

	for _, companyID := range companies {
		dtes, err := s.GetPendingDTEsByCompany(ctx, companyID)
		if err != nil {
			log.Printf("[EventCreator] ❌ Error getting DTEs for company %s: %v", companyID, err)
			continue
		}

		if len(dtes) == 0 {
			continue
		}

		event, err := s.CreateAndSubmitContingencyEvent(ctx, companyID, dtes)
		if err != nil {
			log.Printf("[EventCreator] ❌ Failed to create event for company %s: %v", companyID, err)
			for _, dte := range dtes {
				s.incrementDTERetryCount(ctx, dte.ID)
			}
			continue
		}

		log.Printf("[EventCreator] ✅ Event created: %s", event.ID)
	}
}

// batchSubmitterWorker - submits batches for accepted events
func (s *ContingencyService) batchSubmitterWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

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
	log.Println("[BatchSubmitter] Processing batch submissions...")

	events, err := s.getEventsReadyForBatch(ctx)
	if err != nil {
		log.Printf("[BatchSubmitter] ❌ Error getting events: %v", err)
		return
	}

	if len(events) == 0 {
		log.Println("[BatchSubmitter] No events ready for batch")
		return
	}

	for _, event := range events {
		batch, err := s.CreateAndSubmitBatch(ctx, event.ID)
		if err != nil {
			log.Printf("[BatchSubmitter] ❌ Failed batch for event %s: %v", event.ID, err)
			continue
		}

		log.Printf("[BatchSubmitter] ✅ Batch submitted: %s", batch.ID)
	}
}

// batchPollerWorker - polls for batch results
func (s *ContingencyService) batchPollerWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	time.Sleep(2 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			log.Println("[BatchPoller] Shutting down...")
			return
		case <-ticker.C:
			s.PollBatches(ctx)
		}
	}
}
