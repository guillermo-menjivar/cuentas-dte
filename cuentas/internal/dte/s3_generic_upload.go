package dte

import (
	"context"
	"cuentas/internal/hacienda"
	"fmt"
	"log"
	"time"
)

// uploadDTEToS3 is a GENERIC function that uploads any DTE type to S3
// Call this after successful Hacienda submission for any DTE type
func (s *DTEService) uploadDTEToS3(
	ctx context.Context,
	dteJSON []byte, // The full DTE JSON (unsigned)
	signedDTE string, // The signed DTE string
	response *hacienda.ReceptionResponse, // Hacienda's response
) error {
	// Skip if S3 is not configured
	if s.s3Uploader == nil {
		log.Println("[uploadDTEToS3] S3 uploader not configured, skipping upload")
		return nil
	}

	log.Printf("[uploadDTEToS3] Starting S3 upload for DTE: %s", response.CodigoGeneracion)

	// Extract metadata from DTE JSON
	extractor := NewDTEMetadataExtractor()
	uploadReq, err := extractor.ExtractFromJSON(dteJSON)
	if err != nil {
		return fmt.Errorf("failed to extract DTE metadata: %w", err)
	}

	// Add Hacienda response data
	uploadReq.SignedDTE = signedDTE
	uploadReq.Status = response.Estado

	if response.SelloRecibido != "" {
		uploadReq.ReceivedStamp = &response.SelloRecibido
	}

	if response.FhProcesamiento != "" {
		// Parse Hacienda's timestamp format: "DD/MM/YYYY HH:MM:SS"
		t, err := time.Parse("02/01/2006 15:04:05", response.FhProcesamiento)
		if err != nil {
			log.Printf("[uploadDTEToS3] Warning: failed to parse FhProcesamiento: %v", err)
		} else {
			uploadReq.ProcessingDate = &t
		}
	}

	// Upload to S3
	s3Result, err := s.s3Uploader.UploadDTE(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("S3 upload failed: %w", err)
	}

	log.Printf("[uploadDTEToS3] ✅ S3 upload successful")
	log.Printf("  Download path:  %s", s3Result.DownloadPath)
	log.Printf("  Analytics path: %s", s3Result.AnalyticsPath)
	log.Printf("  File size:      %d bytes", s3Result.FileSize)

	// Save to storage index
	record := s.buildStorageIndexRecord(uploadReq, s3Result)
	if err := s.saveDTEStorageIndex(ctx, record); err != nil {
		log.Printf("[uploadDTEToS3] ⚠️  Warning: failed to save storage index: %v", err)
		// Don't fail - DTE is already in S3
	} else {
		log.Printf("[uploadDTEToS3] ✅ Storage index record saved")
	}

	return nil
}

// uploadDTEToS3Async uploads to S3 in a goroutine (non-blocking)
// Use this when you don't want to wait for S3 upload
func (s *DTEService) uploadDTEToS3Async(
	ctx context.Context,
	dteJSON []byte,
	signedDTE string,
	response *hacienda.ReceptionResponse,
) {
	if s.s3Uploader == nil {
		return // S3 not configured
	}

	go func() {
		if err := s.uploadDTEToS3(ctx, dteJSON, signedDTE, response); err != nil {
			log.Printf("[uploadDTEToS3Async] Error: %v", err)
		}
	}()
}

// buildStorageIndexRecord creates a storage index record from upload request and result
func (s *DTEService) buildStorageIndexRecord(
	req *DTEUploadRequest,
	result *DTEUploadResult,
) *DTEStorageIndexRecord {
	return &DTEStorageIndexRecord{
		CompanyID:              req.CompanyID,
		GenerationCode:         req.GenerationCode,
		ControlNumber:          req.ControlNumber,
		DocumentType:           req.DocumentType,
		IssueDate:              req.IssueDate,
		ProcessingDate:         req.ProcessingDate,
		SenderNIT:              req.SenderNIT,
		SenderName:             req.SenderName,
		ReceiverNIT:            req.ReceiverNIT,
		ReceiverName:           req.ReceiverName,
		ReceiverDocumentType:   req.ReceiverDocumentType,
		ReceiverDocumentNumber: req.ReceiverDocumentNumber,
		TotalAmount:            req.TotalAmount,
		TaxableAmount:          req.TaxableAmount,
		ExemptAmount:           req.ExemptAmount,
		TaxAmount:              req.TaxAmount,
		Currency:               req.Currency,
		Status:                 req.Status,
		ReceivedStamp:          req.ReceivedStamp,
		Observations:           req.Observations,
		S3DownloadPath:         result.DownloadPath,
		S3AnalyticsPath:        result.AnalyticsPath,
		S3Bucket:               s.s3Uploader.bucket,
		S3Region:               s.s3Uploader.region,
		PartitionYear:          result.PartitionYear,
		PartitionMonth:         result.PartitionMonth,
		PartitionDay:           result.PartitionDay,
		FileSizeBytes:          result.FileSize,
		FileChecksum:           result.Checksum,
	}
}

// saveDTEStorageIndex saves a record to the dte_storage_index table
func (s *DTEService) saveDTEStorageIndex(ctx context.Context, record *DTEStorageIndexRecord) error {
	query := `
		INSERT INTO dte_storage_index (
			company_id,
			generation_code,
			control_number,
			document_type,
			issue_date,
			processing_date,
			sender_nit,
			sender_nrc,
			sender_name,
			receiver_nit,
			receiver_nrc,
			receiver_name,
			receiver_document_type,
			receiver_document_number,
			total_amount,
			taxable_amount,
			exempt_amount,
			tax_amount,
			currency,
			status,
			received_stamp,
			observations,
			s3_download_path,
			s3_analytics_path,
			s3_bucket,
			s3_region,
			partition_year,
			partition_month,
			partition_day,
			file_size_bytes,
			file_checksum
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31
		)
		ON CONFLICT (generation_code) 
		DO UPDATE SET
			status = EXCLUDED.status,
			processing_date = EXCLUDED.processing_date,
			received_stamp = EXCLUDED.received_stamp,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		record.CompanyID,              // $1
		record.GenerationCode,         // $2
		record.ControlNumber,          // $3
		record.DocumentType,           // $4
		record.IssueDate,              // $5
		record.ProcessingDate,         // $6
		record.SenderNIT,              // $7
		record.SenderNRC,              // $8
		record.SenderName,             // $9
		record.ReceiverNIT,            // $10
		record.ReceiverNRC,            // $11
		record.ReceiverName,           // $12
		record.ReceiverDocumentType,   // $13
		record.ReceiverDocumentNumber, // $14
		record.TotalAmount,            // $15
		record.TaxableAmount,          // $16
		record.ExemptAmount,           // $17
		record.TaxAmount,              // $18
		record.Currency,               // $19
		record.Status,                 // $20
		record.ReceivedStamp,          // $21
		record.Observations,           // $22
		record.S3DownloadPath,         // $23
		record.S3AnalyticsPath,        // $24
		record.S3Bucket,               // $25
		record.S3Region,               // $26
		record.PartitionYear,          // $27
		record.PartitionMonth,         // $28
		record.PartitionDay,           // $29
		record.FileSizeBytes,          // $30
		record.FileChecksum,           // $31
	)

	return err
}

// DTEStorageIndexRecord represents a record in the storage index
type DTEStorageIndexRecord struct {
	CompanyID              string
	GenerationCode         string
	ControlNumber          string
	DocumentType           string
	IssueDate              time.Time
	ProcessingDate         *time.Time
	SenderNIT              *string
	SenderNRC              *string
	SenderName             *string
	ReceiverNIT            *string
	ReceiverNRC            *string
	ReceiverName           *string
	ReceiverDocumentType   *string
	ReceiverDocumentNumber *string
	TotalAmount            float64
	TaxableAmount          *float64
	ExemptAmount           *float64
	TaxAmount              *float64
	Currency               string
	Status                 string
	ReceivedStamp          *string
	Observations           *string
	S3DownloadPath         string
	S3AnalyticsPath        string
	S3Bucket               string
	S3Region               string
	PartitionYear          int
	PartitionMonth         int
	PartitionDay           int
	FileSizeBytes          int64
	FileChecksum           string
}
