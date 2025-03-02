package filehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Uploader handles S3 operations
type S3Uploader struct {
	client     *s3.Client
	bucketName string
}

// NewS3Uploader creates a new instance of S3Uploader
func NewS3Uploader(ctx context.Context, bucketName string) (*S3Uploader, error) {
	// Load AWS configuration using environment variables
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS configuration: %w", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	return &S3Uploader{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadJSON uploads JSON data to S3 with a timestamped object key
func (u *S3Uploader) UploadJSON(ctx context.Context, data interface{}, objectKey string) (string, error) {
	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling data to JSON: %w", err)
	}

	// Upload the JSON data to S3
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(jsonData),
	})
	if err != nil {
		return "", fmt.Errorf("error uploading JSON to S3: %w", err)
	}

	fmt.Printf("Successfully uploaded JSON to S3 bucket '%s' with key '%s'\n", u.bucketName, objectKey)
	return objectKey, nil
}
