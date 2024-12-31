package filehandler

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Downloader struct {
	client     *s3.Client
	bucketName string
}

func NewS3Downloader(ctx context.Context, bucketName string) (*S3Downloader, error) {
	// Load AWS configuration using environment variables
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Downloader{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Downloads target file as an in memory file
func (d *S3Downloader) DownloadFileToMemory(ctx context.Context, key string) (*bytes.Buffer, error) {
	output, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("error downloading file from S3: %w", err)
	}
	defer output.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, output.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading file content into buffer: %w", err)
	}

	fmt.Printf("Successfully downloaded file from S3 bucket '%s' with key '%s' into memory\n", d.bucketName, key)
	return buf, nil
}
