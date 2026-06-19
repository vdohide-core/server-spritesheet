package uploader

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"server-spritesheet/internal/db/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	multipartThreshold = 100 * 1024 * 1024
	partSize           = 50 * 1024 * 1024
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
	return client, endpoint, nil
}

func fullObjectKey(storage *models.Storage, objectKey string) string {
	if storage.S3 == nil || storage.S3.Prefix == "" {
		return objectKey
	}
	if strings.HasPrefix(objectKey, storage.S3.Prefix) {
		return objectKey
	}
	return strings.TrimRight(storage.S3.Prefix, "/") + "/" + objectKey
}

// UploadToS3 uploads a local file to S3-compatible storage.
func UploadToS3(storage *models.Storage, localPath, objectKey, contentType string, onProgress func(uploaded, total int64)) error {
	client, endpoint, err := newS3Client(storage)
	if err != nil {
		return err
	}
	fullKey := fullObjectKey(storage, objectKey)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}
	totalSize := fileInfo.Size()

	log.Printf("📤 S3 Upload: endpoint=%s bucket=%s key=%s size=%.2fMB",
		endpoint, storage.S3.Bucket, fullKey, float64(totalSize)/1024/1024)

	if totalSize <= multipartThreshold {
		return uploadSinglePart(client, storage.S3.Bucket, fullKey, localPath, totalSize, contentType, onProgress)
	}
	return uploadMultipart(client, storage.S3.Bucket, fullKey, localPath, totalSize, contentType, onProgress)
}

func uploadSinglePart(client *s3.Client, bucket, key, localPath string, totalSize int64, contentType string, onProgress func(uploaded, total int64)) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          f,
		ContentLength: aws.Int64(totalSize),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("S3 PutObject: %w", err)
	}
	if onProgress != nil {
		onProgress(totalSize, totalSize)
	}
	return nil
}

func uploadMultipart(client *s3.Client, bucket, key, localPath string, totalSize int64, contentType string, onProgress func(uploaded, total int64)) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	createResp, err := client.CreateMultipartUpload(context.Background(), &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("S3 CreateMultipartUpload: %w", err)
	}
	uploadID := *createResp.UploadId

	var completedParts []types.CompletedPart
	var uploaded int64
	partNum := int32(1)
	buf := make([]byte, partSize)

	for {
		n, readErr := io.ReadFull(f, buf)
		if n == 0 && readErr != nil {
			break
		}

		partResp, err := client.UploadPart(context.Background(), &s3.UploadPartInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(key),
			UploadId:      aws.String(uploadID),
			PartNumber:    aws.Int32(partNum),
			Body:          strings.NewReader(string(buf[:n])),
			ContentLength: aws.Int64(int64(n)),
		})
		if err != nil {
			client.AbortMultipartUpload(context.Background(), &s3.AbortMultipartUploadInput{
				Bucket: aws.String(bucket), Key: aws.String(key), UploadId: aws.String(uploadID),
			})
			return fmt.Errorf("S3 UploadPart %d: %w", partNum, err)
		}

		completedParts = append(completedParts, types.CompletedPart{
			ETag: partResp.ETag, PartNumber: aws.Int32(partNum),
		})
		uploaded += int64(n)
		if onProgress != nil {
			onProgress(uploaded, totalSize)
		}
		partNum++
		if readErr != nil {
			break
		}
	}

	_, err = client.CompleteMultipartUpload(context.Background(), &s3.CompleteMultipartUploadInput{
		Bucket: aws.String(bucket), Key: aws.String(key), UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completedParts},
	})
	if err != nil {
		return fmt.Errorf("S3 CompleteMultipartUpload: %w", err)
	}
	return nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
