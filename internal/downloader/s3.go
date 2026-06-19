package downloader

import (
	"context"
	"fmt"
	"strings"

	"server-spritesheet/internal/db/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newS3Client(storage *models.Storage) (*s3.Client, string, error) {
	if storage.S3 == nil {
		return nil, "", fmt.Errorf("storage has no S3 config")
	}
	s3Cfg := storage.S3
	endpoint := strings.TrimRight(derefStr(s3Cfg.Endpoint), "/")
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	if strings.HasSuffix(endpoint, "/"+s3Cfg.Bucket) {
		endpoint = endpoint[:len(endpoint)-len(s3Cfg.Bucket)-1]
	}
	region := s3Cfg.Region
	if region == "" {
		region = "auto"
	}
	client := s3.New(s3.Options{
		Region:       region,
		BaseEndpoint: &endpoint,
		Credentials: credentials.NewStaticCredentialsProvider(
			s3Cfg.AccessKeyID,
			s3Cfg.SecretAccessKey,
			"",
		),
		UsePathStyle: s3Cfg.ForcePathStyle,
	})
	return client, s3Cfg.Bucket, nil
}

func objectKey(storage *models.Storage, key string) string {
	if storage.S3.Prefix != "" && !strings.HasPrefix(key, storage.S3.Prefix) {
		return strings.TrimRight(storage.S3.Prefix, "/") + "/" + key
	}
	return key
}

// ObjectExists checks whether an object exists on S3-compatible storage.
func ObjectExists(storage *models.Storage, key string) (bool, error) {
	client, bucket, err := newS3Client(storage)
	if err != nil {
		return false, err
	}
	_, err = client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey(storage, key)),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
