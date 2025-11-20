package dte

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// UploadDTEToS3 uploads a DTE to S3 - SIMPLE, GENERIC, WORKS FOR ALL TYPES
//
// Parameters:
//   - dteBytes: The DTE content (JSON or signed JWT)
//   - signed: true if this is signed JWT, false if unsigned JSON
//   - tipoDte: "01", "03", "04", "05", "06", "11", "14"
//   - companyID: UUID of the company
//   - codigoGeneracion: UUID of the DTE
func UploadDTEToS3(
	ctx context.Context,
	dteBytes []byte,
	signed bool,
	tipoDte string,
	companyID string,
	codigoGeneracion string,
) error {
	// Get bucket from env
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "cuentas-dtes-prod"
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	// Build S3 path
	now := time.Now()
	var filename string
	if signed {
		filename = "signed.jwt"
	} else {
		filename = "unsigned.json"
	}

	s3Key := fmt.Sprintf(
		"dtes/%s/%s/%s/%04d/%02d/%02d/%s",
		companyID,
		tipoDte,
		codigoGeneracion,
		now.Year(),
		now.Month(),
		now.Day(),
		filename,
	)

	// Calculate checksum
	hash := sha256.Sum256(dteBytes)
	checksum := hex.EncodeToString(hash[:])

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Upload
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(bucket),
		Key:                  aws.String(s3Key),
		Body:                 bytes.NewReader(dteBytes),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
		Metadata: map[string]string{
			"company-id":        companyID,
			"codigo-generacion": codigoGeneracion,
			"tipo-dte":          tipoDte,
			"signed":            fmt.Sprintf("%v", signed),
			"checksum-sha256":   checksum,
			"uploaded-at":       now.Format(time.RFC3339),
		},
	})

	if err != nil {
		return fmt.Errorf("S3 upload failed: %w", err)
	}

	log.Printf("[S3] ✅ Uploaded %s to s3://%s/%s (%d bytes)",
		filename, bucket, s3Key, len(dteBytes))

	return nil
}

// UploadDTEToS3Async - non-blocking version
func UploadDTEToS3Async(
	dteBytes []byte,
	signed bool,
	tipoDte string,
	companyID string,
	codigoGeneracion string,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := UploadDTEToS3(ctx, dteBytes, signed, tipoDte, companyID, codigoGeneracion)
		if err != nil {
			log.Printf("[S3] ⚠️  Upload failed: %v", err)
		}
	}()
}
