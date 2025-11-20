package dte

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Uploader handles uploading DTEs to S3
type S3Uploader struct {
	client *s3.Client
	bucket string
	region string
}

// NewS3Uploader creates a new S3 uploader
func NewS3Uploader(bucket, region string) (*S3Uploader, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Uploader{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		region: region,
	}, nil
}

// DTEUploadRequest contains all data needed to upload a DTE to S3
type DTEUploadRequest struct {
	// Required fields
	CompanyID      string
	GenerationCode string
	ControlNumber  string
	DocumentType   string
	IssueDate      time.Time

	// DTE content
	UnsignedJSON []byte // Full DTE JSON (unsigned)
	SignedDTE    string // Signed DTE string

	// Financial summary
	TotalAmount   float64
	TaxableAmount *float64
	TaxAmount     *float64

	// Parties
	SenderNIT    *string
	SenderName   *string
	ReceiverName *string
	ReceiverNIT  *string

	// Status
	Status string

	// Hacienda response (if available)
	ReceivedStamp  *string
	ProcessingDate *time.Time
}

// DTEUploadResult contains the S3 paths and metadata
type DTEUploadResult struct {
	DownloadPath   string
	AnalyticsPath  string
	FileSize       int64
	Checksum       string
	PartitionYear  int
	PartitionMonth int
	PartitionDay   int
}

// UploadDTE uploads a DTE to both S3 paths and returns metadata for database storage
func (u *S3Uploader) UploadDTE(ctx context.Context, req *DTEUploadRequest) (*DTEUploadResult, error) {
	log.Printf("[S3Uploader] Uploading DTE: generation_code=%s, document_type=%s",
		req.GenerationCode, req.DocumentType)

	// Calculate checksum
	checksum := u.calculateChecksum(req.UnsignedJSON)

	// Get partition values from issue date
	year, month, day := req.IssueDate.Date()

	// Build S3 paths
	downloadPath := u.buildDownloadPath(req.CompanyID, req.GenerationCode)
	analyticsPath := u.buildAnalyticsPath(
		req.CompanyID,
		req.DocumentType,
		year,
		int(month),
		day,
		req.GenerationCode,
	)

	log.Printf("[S3Uploader] Download path: %s", downloadPath)
	log.Printf("[S3Uploader] Analytics path: %s", analyticsPath)

	// Create metadata object with all DTE info
	metadata := u.createMetadata(req)
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Upload to downloads path (full document + metadata)
	if err := u.uploadDocument(ctx, downloadPath, req.UnsignedJSON); err != nil {
		return nil, fmt.Errorf("failed to upload to downloads path: %w", err)
	}

	// Upload metadata file
	metadataPath := u.buildDownloadPath(req.CompanyID, req.GenerationCode) + ".metadata.json"
	if err := u.uploadDocument(ctx, metadataPath, metadataJSON); err != nil {
		log.Printf("[S3Uploader] Warning: failed to upload metadata: %v", err)
		// Don't fail - metadata is optional
	}

	// Upload to analytics path (for Athena)
	if err := u.uploadDocument(ctx, analyticsPath, req.UnsignedJSON); err != nil {
		return nil, fmt.Errorf("failed to upload to analytics path: %w", err)
	}

	log.Printf("[S3Uploader] ✅ Successfully uploaded DTE to S3")

	return &DTEUploadResult{
		DownloadPath:   downloadPath,
		AnalyticsPath:  analyticsPath,
		FileSize:       int64(len(req.UnsignedJSON)),
		Checksum:       checksum,
		PartitionYear:  year,
		PartitionMonth: int(month),
		PartitionDay:   day,
	}, nil
}

// buildDownloadPath creates the S3 path for downloads
// Format: dtes/{company_id}/{generation_code}/document.json
func (u *S3Uploader) buildDownloadPath(companyID, generationCode string) string {
	return fmt.Sprintf("dtes/%s/%s/document.json", companyID, generationCode)
}

// buildAnalyticsPath creates the S3 path for analytics
// Format: analytics/company_id={id}/document_type={type}/year={yyyy}/month={mm}/day={dd}/{generation_code}.json
func (u *S3Uploader) buildAnalyticsPath(
	companyID, documentType string,
	year, month, day int,
	generationCode string,
) string {
	return fmt.Sprintf(
		"analytics/company_id=%s/document_type=%s/year=%d/month=%02d/day=%02d/%s.json",
		companyID,
		documentType,
		year,
		month,
		day,
		generationCode,
	)
}

// uploadDocument uploads a document to S3
func (u *S3Uploader) uploadDocument(ctx context.Context, key string, data []byte) error {
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(u.bucket),
		Key:                  aws.String(key),
		Body:                 bytes.NewReader(data),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: aws.String("AES256"),
	})

	if err != nil {
		return fmt.Errorf("S3 PutObject failed for key=%s: %w", key, err)
	}

	return nil
}

// calculateChecksum calculates SHA256 checksum of the data
func (u *S3Uploader) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// createMetadata creates a metadata object for quick lookups
func (u *S3Uploader) createMetadata(req *DTEUploadRequest) map[string]interface{} {
	metadata := map[string]interface{}{
		"generation_code": req.GenerationCode,
		"control_number":  req.ControlNumber,
		"document_type":   req.DocumentType,
		"issue_date":      req.IssueDate.Format(time.RFC3339),
		"total_amount":    req.TotalAmount,
		"status":          req.Status,
		"company_id":      req.CompanyID,
	}

	if req.SenderName != nil {
		metadata["sender_name"] = *req.SenderName
	}
	if req.SenderNIT != nil {
		metadata["sender_nit"] = *req.SenderNIT
	}
	if req.ReceiverName != nil {
		metadata["receiver_name"] = *req.ReceiverName
	}
	if req.ReceiverNIT != nil {
		metadata["receiver_nit"] = *req.ReceiverNIT
	}
	if req.TaxableAmount != nil {
		metadata["taxable_amount"] = *req.TaxableAmount
	}
	if req.TaxAmount != nil {
		metadata["tax_amount"] = *req.TaxAmount
	}
	if req.ReceivedStamp != nil {
		metadata["received_stamp"] = *req.ReceivedStamp
	}
	if req.ProcessingDate != nil {
		metadata["processing_date"] = req.ProcessingDate.Format(time.RFC3339)
	}

	return metadata
}

// DownloadDTE downloads a DTE from S3 by generation code
func (u *S3Uploader) DownloadDTE(ctx context.Context, companyID, generationCode string) ([]byte, error) {
	key := u.buildDownloadPath(companyID, generationCode)

	result, err := u.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return buf.Bytes(), nil
}
