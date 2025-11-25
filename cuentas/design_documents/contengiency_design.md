El Salvador DTE Contingency System - Technical Design Document
Version: 2.0 (Final)
Date: 2025-01-15

1. System Overview
1.1 Purpose
Handle DTE (electronic invoice) processing failures and implement Hacienda's mandated 3-step contingency recovery process.
1.2 Three Failure Scenarios

POS Offline - POS loses connectivity, creates invoices offline, syncs later
Firmador Failure - Cannot sign DTEs (service down)
Hacienda Failure - Cannot transmit DTEs to Hacienda (service down)

1.3 Recovery Pipeline
All failures converge into: Period → Event → Lotes → Poll Results

2. Database Schema
2.1 invoices Table (Modified)
sql-- Add columns to existing invoices table
ALTER TABLE invoices ADD COLUMN codigo_generacion VARCHAR(36) UNIQUE NOT NULL;
ALTER TABLE invoices ADD COLUMN contingency_period_id UUID REFERENCES contingency_periods(id);
ALTER TABLE invoices ADD COLUMN contingency_event_id UUID REFERENCES contingency_events(id);
ALTER TABLE invoices ADD COLUMN lote_id UUID REFERENCES lotes(id);

ALTER TABLE invoices ADD COLUMN dte_transmission_status VARCHAR(20) DEFAULT 'pending';
-- Status values: 'pending', 'pending_signature', 'contingency_queued', 'failed_retry', 'procesado', 'rechazado'

ALTER TABLE invoices ADD COLUMN dte_unsigned JSONB;
ALTER TABLE invoices ADD COLUMN dte_signed TEXT;
ALTER TABLE invoices ADD COLUMN dte_sello_recibido TEXT;
ALTER TABLE invoices ADD COLUMN hacienda_observaciones TEXT[];
ALTER TABLE invoices ADD COLUMN ambiente VARCHAR(2) NOT NULL DEFAULT '00';
ALTER TABLE invoices ADD COLUMN signature_retry_count INT DEFAULT 0;

-- Indexes
CREATE UNIQUE INDEX idx_invoices_codigo_generacion ON invoices(codigo_generacion);
CREATE INDEX idx_invoices_contingency_period ON invoices(contingency_period_id);
CREATE INDEX idx_invoices_contingency_event ON invoices(contingency_event_id);
CREATE INDEX idx_invoices_lote ON invoices(lote_id);
CREATE INDEX idx_invoices_status ON invoices(dte_transmission_status);
CREATE INDEX idx_invoices_pending_sig ON invoices(contingency_period_id) 
    WHERE dte_transmission_status = 'pending_signature';
2.2 contingency_periods Table
sqlCREATE TABLE contingency_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    f_inicio DATE NOT NULL,
    h_inicio TIME NOT NULL,
    f_fin DATE,
    h_fin TIME,
    
    tipo_contingencia INT NOT NULL CHECK (tipo_contingencia BETWEEN 1 AND 5),
    motivo_contingencia TEXT,
    
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    -- Status: 'active' (collecting), 'reporting' (events being created), 'completed' (all done)
    
    processing BOOLEAN DEFAULT false,  -- Concurrency control flag
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_active_periods ON contingency_periods(company_id, status) 
    WHERE status IN ('active', 'reporting');

CREATE INDEX idx_periods_claimable ON contingency_periods(status, processing) 
    WHERE status IN ('active', 'reporting') AND processing = false;

CREATE INDEX idx_periods_by_pos ON contingency_periods(
    company_id, establishment_id, point_of_sale_id
);

-- Enforce one active period per POS
CREATE UNIQUE INDEX idx_one_active_period_per_pos 
ON contingency_periods(company_id, establishment_id, point_of_sale_id)
WHERE status = 'active';

-- Documentation
COMMENT ON COLUMN contingency_periods.f_fin IS 
'End date - represents when period was closed (first event created), may be slightly after actual outage end';

COMMENT ON COLUMN contingency_periods.h_fin IS 
'End time - represents when period was closed (first event created), may be slightly after actual outage end';
Tipo Contingencia Catalog:

1 = Hacienda system down
2 = POS system failure
3 = Internet outage
4 = Power outage
5 = Other (requires motivo_contingencia text)

2.3 contingency_events Table
sqlCREATE TABLE contingency_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_period_id UUID NOT NULL REFERENCES contingency_periods(id),
    
    codigo_generacion VARCHAR(36) UNIQUE NOT NULL,
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    ambiente VARCHAR(2) NOT NULL,
    
    event_json JSONB NOT NULL,
    event_signed TEXT NOT NULL,
    
    estado VARCHAR(20),
    sello_recibido TEXT,
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ
);

CREATE INDEX idx_events_by_period ON contingency_events(contingency_period_id);
CREATE INDEX idx_events_by_status ON contingency_events(estado);
CREATE UNIQUE INDEX idx_events_codigo ON contingency_events(codigo_generacion);
2.4 lotes Table
sqlCREATE TABLE lotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contingency_event_id UUID NOT NULL REFERENCES contingency_events(id),
    
    codigo_lote VARCHAR(100),
    
    company_id UUID NOT NULL,
    establishment_id UUID NOT NULL,
    point_of_sale_id UUID NOT NULL,
    
    dte_count INT NOT NULL,
    
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- Status: 'pending', 'submitted', 'completed'
    
    processing BOOLEAN DEFAULT false,  -- Concurrency control flag
    
    hacienda_response JSONB,
    
    submitted_at TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_lotes_by_event ON lotes(contingency_event_id);
CREATE INDEX idx_pending_lotes ON lotes(status) 
    WHERE status IN ('pending', 'submitted');
CREATE INDEX idx_lotes_by_codigo ON lotes(codigo_lote);
CREATE INDEX idx_lotes_claimable ON lotes(status, processing) 
    WHERE status IN ('pending', 'submitted') AND processing = false;
2.5 Ambiente Consistency Enforcement
sqlCREATE OR REPLACE FUNCTION check_ambiente_consistency()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.contingency_period_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM contingency_periods 
            WHERE id = NEW.contingency_period_id 
            AND ambiente = NEW.ambiente
        ) THEN
            RAISE EXCEPTION 'Invoice ambiente (%) does not match period ambiente', NEW.ambiente;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_invoice_ambiente
BEFORE INSERT OR UPDATE ON invoices
FOR EACH ROW
WHEN (NEW.contingency_period_id IS NOT NULL)
EXECUTE FUNCTION check_ambiente_consistency();

3. Workflow: POS Offline Scenario
3.1 API Endpoint
POST /v1/invoices/sync
Request:
json{
  "invoices": [/* array of invoice objects */],
  "contingency_type": 3,
  "contingency_reason": "Internet outage"
}
3.2 Processing Logic
gofunc (s *InvoiceService) HandleInvoiceSync(ctx context.Context, req SyncRequest) error {
    // Validate
    if !codigos.IsValidContingencyType(req.ContingencyType) {
        return fmt.Errorf("invalid contingency_type: must be 1-5")
    }
    
    if req.ContingencyType == 5 && req.ContingencyReason == "" {
        return fmt.Errorf("contingency_reason required when type=5")
    }
    
    // Get time window
    earliestTime, _ := getTimeWindow(req.Invoices)
    
    // Find or create period
    period, err := s.findOrCreatePeriod(
        req.Invoices[0].CompanyID,
        req.Invoices[0].EstablishmentID,
        req.Invoices[0].PointOfSaleID,
        req.Invoices[0].Ambiente,
        req.ContingencyType,
        req.ContingencyReason,
        earliestTime,
    )
    
    // Process each invoice
    for _, invoiceData := range req.Invoices {
        invoice := createInvoiceFromData(invoiceData)
        
        dte, _ := s.dteBuilder.BuildFromInvoice(ctx, invoice)
        dteJSON, _ := json.Marshal(dte)
        
        // Try to sign
        signed, err := s.firmador.Sign(ctx, creds.NIT, creds.Password, dte)
        if err != nil {
            // Firmador failed - store unsigned
            invoice.ContingencyPeriodID = &period.ID
            invoice.DteTransmissionStatus = "pending_signature"
            invoice.DteUnsigned = dteJSON
            invoice.SignatureRetryCount = 0
        } else {
            // Successfully signed
            invoice.ContingencyPeriodID = &period.ID
            invoice.DteTransmissionStatus = "contingency_queued"
            invoice.DteUnsigned = dteJSON
            invoice.DteSigned = &signed
        }
        
        invoice.CodigoGeneracion = dte.Identificacion.CodigoGeneracion
        invoice.Ambiente = dte.Identificacion.Ambiente
        s.db.Save(invoice)
    }
    
    return nil
}
3.3 Period Creation
gofunc (s *InvoiceService) findOrCreatePeriod(
    companyID, establishmentID, posID, ambiente string,
    tipoContingencia int,
    motivoContingencia string,
    timeOfFirstInvoice time.Time,
) (*models.ContingencyPeriod, error) {
    
    loc, _ := time.LoadLocation("America/El_Salvador")
    if loc == nil {
        loc = time.FixedZone("CST", -6*60*60)
    }
    localTime := timeOfFirstInvoice.In(loc)
    
    // Try to find existing ACTIVE period for this POS
    var period models.ContingencyPeriod
    err := s.db.Where(`
        company_id = ? AND 
        establishment_id = ? AND 
        point_of_sale_id = ? AND 
        ambiente = ? AND
        status = 'active'
    `, companyID, establishmentID, posID, ambiente).First(&period).Error
    
    if err == nil {
        // Reuse existing period
        return &period, nil
    }
    
    // Create new period
    period = models.ContingencyPeriod{
        ID:                 uuid.New().String(),
        CompanyID:          companyID,
        EstablishmentID:    establishmentID,
        PointOfSaleID:      posID,
        Ambiente:           ambiente,
        FInicio:            localTime.Format("2006-01-02"),
        HInicio:            localTime.Format("15:04:05"),
        TipoContingencia:   tipoContingencia,
        MotivoContingencia: motivoContingencia,
        Status:             "active",
    }
    
    return &period, s.db.Create(&period).Error
}

4. Workflow: Real-Time Failures (Firmador/Hacienda)
4.1 Process Invoice
gofunc (s *DTEService) ProcessInvoice(ctx context.Context, invoice *models.Invoice) error {
    // Build DTE
    dte, _ := s.builder.BuildFromInvoice(ctx, invoice)
    dteJSON, _ := json.Marshal(dte)
    
    // Try to sign (3 retries)
    var signed string
    var signErr error
    
    for attempt := 1; attempt <= 3; attempt++ {
        signed, signErr = s.firmador.Sign(ctx, creds.NIT, creds.Password, dte)
        if signErr == nil {
            break
        }
        if attempt < 3 {
            time.Sleep(time.Duration(attempt*2) * time.Second)
        }
    }
    
    if signErr != nil {
        // FIRMADOR FAILED
        return s.queueForContingency(
            invoice,
            "firmador_failed",
            dteJSON,
            nil, // No signature
            dte.Identificacion.Ambiente,
        )
    }
    
    // Authenticate with Hacienda
    authResponse, err := s.haciendaService.AuthenticateCompany(ctx, invoice.CompanyID)
    if err != nil {
        return s.queueForContingency(invoice, "hacienda_auth_failed", dteJSON, &signed, ambiente)
    }
    
    // Try to submit to Hacienda (3 retries)
    var response *hacienda.ReceptionResponse
    var submitErr error
    
    for attempt := 1; attempt <= 3; attempt++ {
        response, submitErr = s.hacienda.SubmitDTE(ctx, authResponse.Body.Token, ambiente, 
            dte.Identificacion.TipoDte, codigoGeneracion, signed)
        
        if submitErr == nil && response != nil && response.Estado == "PROCESADO" {
            break
        }
        
        if isPermanentRejection(response) {
            invoice.DteTransmissionStatus = "rechazado"
            invoice.HaciendaObservaciones = response.Observaciones
            s.db.Save(invoice)
            return fmt.Errorf("DTE rejected: %s", response.DescripcionMsg)
        }
        
        if attempt < 3 {
            time.Sleep(time.Duration(attempt*2) * time.Second)
        }
    }
    
    if submitErr != nil || response == nil || response.Estado != "PROCESADO" {
        // HACIENDA FAILED
        return s.queueForContingency(invoice, "hacienda_timeout", dteJSON, &signed, ambiente)
    }
    
    // SUCCESS
    invoice.DteTransmissionStatus = "procesado"
    invoice.DteSelloRecibido = &response.SelloRecibido
    invoice.DteSigned = &signed
    s.db.Save(invoice)
    
    return nil
}
4.2 Queue for Contingency
gofunc (s *DTEService) queueForContingency(
    invoice *models.Invoice,
    failureType string,
    dteUnsigned []byte,
    dteSigned *string,
    ambiente string,
) error {
    
    // Determine tipo_contingencia
    var tipoContingencia int
    var motivoContingencia string
    
    switch failureType {
    case "firmador_failed":
        tipoContingencia = 5
        motivoContingencia = "Falla en servicio de firmador - no se pudo firmar el DTE"
    case "hacienda_auth_failed":
        tipoContingencia = 1
        motivoContingencia = "No fue posible autenticarse con el sistema del MH"
    case "hacienda_timeout":
        tipoContingencia = 1
        motivoContingencia = "No disponibilidad de sistema del MH - timeout después de 3 intentos"
    default:
        tipoContingencia = 5
        motivoContingencia = fmt.Sprintf("Error no clasificado: %s", failureType)
    }
    
    // Find or create period
    period, err := s.findOrCreatePeriod(
        invoice.CompanyID,
        invoice.EstablishmentID,
        invoice.PointOfSaleID,
        ambiente,
        tipoContingencia,
        motivoContingencia,
        time.Now(),
    )
    
    // Update invoice
    invoice.ContingencyPeriodID = &period.ID
    invoice.DteUnsigned = dteUnsigned
    invoice.Ambiente = ambiente
    
    if dteSigned != nil {
        invoice.DteTransmissionStatus = "failed_retry"
        invoice.DteSigned = dteSigned
    } else {
        invoice.DteTransmissionStatus = "pending_signature"
        invoice.DteSigned = nil
        invoice.SignatureRetryCount = 0
    }
    
    return s.db.Save(invoice).Error
}

5. Worker 1: Event Creator (Every 10 Minutes)
5.1 Main Loop
gofunc (w *ContingencyWorker) EventCreatorWorker(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()
    
    // Run immediately
    w.processContingencyEvents(ctx)
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processContingencyEvents(ctx)
        }
    }
}
5.2 Process Events (With Concurrency Control)
gofunc (w *ContingencyWorker) processContingencyEvents(ctx context.Context) {
    // Claim periods atomically
    rows, _ := w.db.Raw(`
        UPDATE contingency_periods
        SET processing = true
        WHERE id IN (
            SELECT id FROM contingency_periods
            WHERE status IN ('active', 'reporting')
            AND processing = false
            ORDER BY created_at ASC
            LIMIT 10
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, company_id, establishment_id, point_of_sale_id,
                  ambiente, tipo_contingencia, motivo_contingencia,
                  status, f_inicio, h_inicio, f_fin, h_fin
    `).Rows()
    
    defer rows.Close()
    
    var periods []models.ContingencyPeriod
    for rows.Next() {
        var period models.ContingencyPeriod
        rows.Scan(&period.ID, &period.CompanyID, /* ... */)
        periods = append(periods, period)
    }
    
    for _, period := range periods {
        err := w.processPeriod(ctx, &period)
        
        // ALWAYS release lock
        w.db.Model(&models.ContingencyPeriod{}).
            Where("id = ?", period.ID).
            Update("processing", false)
        
        if err != nil {
            log.Printf("Error processing period %s: %v", period.ID, err)
        }
    }
}
5.3 Process Single Period
gofunc (w *ContingencyWorker) processPeriod(ctx context.Context, period *models.ContingencyPeriod) error {
    // Get unreported invoices
    var invoices []models.Invoice
    w.db.Where(`
        contingency_period_id = ? AND
        contingency_event_id IS NULL AND
        dte_transmission_status IN (?, ?, ?)
    `, period.ID, "pending_signature", "failed_retry", "contingency_queued").
        Order("finalized_at ASC").
        Limit(1000).
        Find(&invoices)
    
    if len(invoices) == 0 {
        if period.Status == "reporting" {
            if w.checkPeriodCompletion(ctx, period) {
                period.Status = "completed"
                w.db.Save(period)
            }
        }
        return nil
    }
    
    // Retry signatures
    signedInvoices := w.retrySignatures(ctx, invoices)
    
    if len(signedInvoices) == 0 {
        return nil
    }
    
    // Close period if first event
    if period.Status == "active" {
        loc, _ := time.LoadLocation("America/El_Salvador")
        now := time.Now().In(loc)
        
        // NOTE: f_fin/h_fin = when period was closed, not last invoice time
        period.FFin = ptrString(now.Format("2006-01-02"))
        period.HFin = ptrString(now.Format("15:04:05"))
        period.Status = "reporting"
        w.db.Save(period)
    }
    
    // Build event
    eventJSON, codigoGeneracion, _ := w.buildContingencyEvent(ctx, period, signedInvoices)
    
    // Sign event
    var eventForSigning interface{}
    json.Unmarshal(eventJSON, &eventForSigning)
    signedEvent, _ := w.firmador.Sign(ctx, creds.NIT, creds.Password, eventForSigning)
    
    // Submit to Hacienda
    authResponse, _ := w.haciendaService.AuthenticateCompany(ctx, period.CompanyID)
    response, _ := w.hacienda.SubmitContingencyEvent(ctx, authResponse.Body.Token, creds.NIT, signedEvent)
    
    if response.Estado != "RECIBIDO" {
        return fmt.Errorf("event rejected: %s", response.Mensaje)
    }
    
    // Store event
    eventID := uuid.New().String()
    eventRecord := models.ContingencyEvent{
        ID:                 eventID,
        ContingencyPeriodID: period.ID,
        CodigoGeneracion:   codigoGeneracion,
        /* ... other fields ... */
    }
    w.db.Create(&eventRecord)
    
    // Link invoices to event
    invoiceIDs := extractIDs(signedInvoices)
    w.db.Model(&models.Invoice{}).
        Where("id IN ?", invoiceIDs).
        Update("contingency_event_id", eventID)
    
    // Create lotes
    w.createLotes(ctx, &eventRecord, signedInvoices)
    
    return nil
}
5.4 Retry Signatures
gofunc (w *ContingencyWorker) retrySignatures(ctx context.Context, invoices []models.Invoice) []models.Invoice {
    var signedInvoices []models.Invoice
    
    for i := range invoices {
        invoice := &invoices[i]
        
        if invoice.DteSigned != nil && *invoice.DteSigned != "" {
            signedInvoices = append(signedInvoices, *invoice)
            continue
        }
        
        // Try to sign
        var dte interface{}
        json.Unmarshal(invoice.DteUnsigned, &dte)
        
        signed, err := w.firmador.Sign(ctx, creds.NIT, creds.Password, dte)
        if err != nil {
            invoice.SignatureRetryCount++
            w.db.Save(invoice)
            
            if invoice.SignatureRetryCount >= 10 {
                w.alertAdmin(fmt.Sprintf("Firmador unavailable: Invoice %s failed %d times", 
                    invoice.ID, invoice.SignatureRetryCount))
            }
            continue
        }
        
        // Success
        invoice.DteSigned = &signed
        invoice.DteTransmissionStatus = "contingency_queued"
        invoice.SignatureRetryCount = 0
        w.db.Save(invoice)
        
        signedInvoices = append(signedInvoices, *invoice)
    }
    
    return signedInvoices
}
5.5 Build Contingency Event
gofunc (w *ContingencyWorker) buildContingencyEvent(
    ctx context.Context,
    period *models.ContingencyPeriod,
    invoices []models.Invoice,
) ([]byte, string, error) {
    
    codigoGeneracion := strings.ToUpper(uuid.New().String())
    
    // Build detalleDTE
    detalleDTE := make([]map[string]interface{}, len(invoices))
    for i, inv := range invoices {
        detalleDTE[i] = map[string]interface{}{
            "noItem":           i + 1,
            "codigoGeneracion": inv.CodigoGeneracion,
            "tipoDoc":          inv.TipoDte,
        }
    }
    
    now := time.Now().In(loc)
    
    event := map[string]interface{}{
        "identificacion": map[string]interface{}{
            "version":          3,
            "ambiente":         period.Ambiente,
            "codigoGeneracion": codigoGeneracion,
            "fTransmision":     now.Format("2006-01-02"),
            "hTransmision":     now.Format("15:04:05"),
        },
        "emisor": map[string]interface{}{
            "nit":    company.NIT,
            "nombre": company.LegalName,
            /* ... */
        },
        "detalleDTE": detalleDTE,
        "motivo": map[string]interface{}{
            "fInicio":            period.FInicio,
            "fFin":               period.FFin,
            "hInicio":            period.HInicio,
            "hFin":               period.HFin,
            "tipoContingencia":   period.TipoContingencia,
            "motivoContingencia": period.MotivoContingencia,
        },
    }
    
    eventJSON, _ := json.Marshal(event)
    return eventJSON, codigoGeneracion, nil
}
5.6 Create Lotes
gofunc (w *ContingencyWorker) createLotes(
    ctx context.Context,
    event *models.ContingencyEvent,
    invoices []models.Invoice,
) error {
    batchSize := 100
    totalBatches := (len(invoices) + batchSize - 1) / batchSize
    
    for i := 0; i < totalBatches; i++ {
        start := i * batchSize
        end := start + batchSize
        if end > len(invoices) {
            end = len(invoices)
        }
        
        batchInvoices := invoices[start:end]
        
        loteID := uuid.New().String()
        lote := models.Lote{
            ID:                 loteID,
            ContingencyEventID: event.ID,
            CompanyID:          event.CompanyID,
            EstablishmentID:    event.EstablishmentID,
            PointOfSaleID:      event.PointOfSaleID,
            DTECount:           len(batchInvoices),
            Status:             "pending",
        }
        w.db.Create(&lote)
        
        // Link invoices to lote
        invoiceIDs := extractIDs(batchInvoices)
        w.db.Model(&models.Invoice{}).
            Where("id IN ?", invoiceIDs).
            Update("lote_id", loteID)
    }
    
    return nil
}

6. Worker 2: Lote Processor (Every 5 Minutes)
6.1 Main Loop
gofunc (w *ContingencyWorker) LoteProcessorWorker(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    time.Sleep(30 * time.Second) // Initial delay
    w.processLotes(ctx)
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.processLotes(ctx)
        }
    }
}

func (w *ContingencyWorker) processLotes(ctx context.Context) {
    w.submitPendingLotes(ctx)
    w.pollSubmittedLotes(ctx)
}
6.2 Submit Pending Lotes
gofunc (w *ContingencyWorker) submitPendingLotes(ctx context.Context) {
    // Claim lotes
    rows, _ := w.db.Raw(`
        UPDATE lotes
        SET processing = true
        WHERE id IN (
            SELECT id FROM lotes
            WHERE status = 'pending'
            AND processing = false
            ORDER BY created_at ASC
            LIMIT 10
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, contingency_event_id, company_id, dte_count
    `).Rows()
    defer rows.Close()
    
    var lotes []models.Lote
    for rows.Next() {
        var lote models.Lote
        rows.Scan(&lote.ID, &lote.ContingencyEventID, &lote.CompanyID, &lote.DTECount)
        lotes = append(lotes, lote)
    }
    
    for _, lote := range lotes {
        w.submitLote(ctx, &lote)
        
        // Release lock
        w.db.Model(&models.Lote{}).
            Where("id = ?", lote.ID).
            Update("processing", false)
    }
}

func (w *ContingencyWorker) submitLote(ctx context.Context, lote *models.Lote) error {
    // Get invoices
    var invoices []models.Invoice
    w.db.Where("lote_id = ?", lote.ID).Order("finalized_at ASC").Find(&invoices)
    
    // Build lote JSON
    documentos := make([]string, len(invoices))
    for i, inv := range invoices {
        documentos[i] = *inv.DteSigned
    }
    
    loteJSON := map[string]interface{}{
        "identificacion": map[string]interface{}{
            "version":  1,
            "ambiente": invoices[0].Ambiente,
            "idEnvio":  strings.ToUpper(uuid.New().String()),
        },
        "nit":        creds.NIT,
        "documentos": documentos,
    }
    
    // Submit
    authResponse, _ := w.haciendaService.AuthenticateCompany(ctx, lote.CompanyID)
    response, _ := w.hacienda.SubmitLote(ctx, authResponse.Body.Token, loteJSON)
    
    if response.Estado != "RECIBIDO" {
        return fmt.Errorf("lote rejected: %s", response.DescripcionMsg)
    }
    
    // Update lote
    lote.CodigoLote = &response.CodigoLote
    lote.Status = "submitted"
    lote.SubmittedAt = ptrTime(time.Now())
    w.db.Save(lote)
    
    return nil
}
6.3 Poll Submitted Lotes
gofunc (w *ContingencyWorker) pollSubmittedLotes(ctx context.Context) {
    threeMinutesAgo := time.Now().Add(-3 * time.Minute)
    
    rows, _ := w.db.Raw(`
        UPDATE lotes
        SET processing = true, last_polled_at = NOW()
        WHERE id IN (
            SELECT id FROM lotes
            WHERE status = 'submitted'
            AND processing = false
            AND (last_polled_at IS NULL OR last_polled_at < ?)
            ORDER BY submitted_at ASC
            LIMIT 10
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, codigo_lote, company_id, contingency_event_id
    `, threeMinutesAgo).Rows()
    defer rows.Close()
    
    var lotes []models.Lote
    for rows.Next() {
        var lote models.Lote
        rows.Scan(&lote.ID, &lote.CodigoLote, &lote.CompanyID, &lote.ContingencyEventID)
        lotes = append(lotes, lote)
    }
    
    for _, lote := range lotes {
        w.pollLote(ctx, &lote)
        
        // Release lock
        w.db.Model(&models.Lote{}).
            Where("id = ?", lote.ID).
            Update("processing", false)
    }
}

func (w *ContingencyWorker) pollLote(ctx context.Context, lote *models.Lote) error {
    // Query Hacienda
    authResponse, _ := w.haciendaService.AuthenticateCompany(ctx, lote.CompanyID)
    response, _ := w.hacienda.QueryLoteStatus(ctx, authResponse.Body.Token, *lote.CodigoLote)
    
    // Process results
    for _, resultado := range response.Procesados {
        w.db.Model(&models.Invoice{}).
            Where("codigo_generacion = ?", resultado.CodigoGeneracion).
            Updates(map[string]interface{}{
                "dte_transmission_status": "procesado",
                "dte_sello_recibido":      resultado.SelloRecibido,
                "hacienda_observaciones":  pq.Array(resultado.Observaciones),
            })
    }
    
    for _, resultado := range response.Rechazados {
        w.db.Model(&models.Invoice{}).
            Where("codigo_generacion = ?", resultado.CodigoGeneracion).
            Updates(map[string]interface{}{
                "dte_transmission_status": "rechazado",
                "hacienda_observaciones":  pq.Array(resultado.Observaciones),
            })
    }
    
    // Check if lote complete
    var remainingCount int64
    w.db.Model(&models.Invoice{}).
        Where("lote_id = ? AND dte_transmission_status NOT IN (?, ?)", 
            lote.ID, "procesado", "rechazado").
        Count(&remainingCount)
    
    if remainingCount == 0 {
        lote.Status = "completed"
        lote.CompletedAt = ptrTime(time.Now())
        w.db.Save(lote)
        
        w.checkEventCompletion(ctx, lote.ContingencyEventID)
    }
    
    return nil
}
6.4 Check Completion
gofunc (w *ContingencyWorker) checkEventCompletion(ctx context.Context, eventID string) {
    // Check if all lotes complete
    var remainingLotes int64
    w.db.Model(&models.Lote{}).
        Where("contingency_event_id = ? AND status != ?", eventID, "completed").
        Count(&remainingLotes)
    
    if remainingLotes == 0 {
        var event models.ContingencyEvent
        w.db.First(&event, "id = ?", eventID)
        
        // Check if all invoices in period complete
        var remainingInvoices int64
        w.db.Model(&models.Invoice{}).
            Where("contingency_period_id = ? AND dte_transmission_status NOT IN (?, ?)", 
                event.ContingencyPeriodID, "procesado", "rechazado").
            Count(&remainingInvoices)
        
        if remainingInvoices == 0 {
            w.db.Model(&models.ContingencyPeriod{}).
                Where("id = ?", event.ContingencyPeriodID).
                Update("status", "completed")
        }
    }
}

7. Configuration
yamlcontingency:
  old_invoice_threshold: 1h
  
  workers:
    event_creator_interval: 10m
    lote_processor_interval: 5m
    poll_interval: 3m
  
  limits:
    max_dtes_per_event: 1000
    max_dtes_per_lote: 100
  
  retry:
    timeout: 8s
    max_attempts: 3
    backoff: [2s, 5s]
  
  alerts:
    signature_retry_threshold: 10
    admin_email: "admin@company.com"
