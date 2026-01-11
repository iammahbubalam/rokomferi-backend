package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"rokomferi-backend/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client     *s3.Client
	bucketName string
	publicURL  string
}

func NewR2Storage(accountId, accessKey, secretKey, bucketName, publicURL string) (*R2Storage, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	return &R2Storage{
		client:     client,
		bucketName: bucketName,
		publicURL:  strings.TrimSuffix(publicURL, "/"),
	}, nil
}

func (s *R2Storage) UploadFile(file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("fast-shopping/%s%s", utils.GenerateUUID(), ext) // Organized in folder

	// Upload to R2
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        file,
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Return Public URL
	return fmt.Sprintf("%s/%s", s.publicURL, filename), nil
}
