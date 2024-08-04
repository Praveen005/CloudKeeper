package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func backup(ctx context.Context) {
	ticker := time.NewTicker(MetaCfg.backupInterval)
	fmt.Println("Inside backup function, backup interval: ", MetaCfg.backupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Println("[INFO] starting files updation in S3")

			if err := flushToS3(); err != nil {
				log.Fatalf("backup failed: %v", err)
			}
			log.Printf("[Info] Success! files updated in s3.")
		case <-ctx.Done():
			return
		}
	}
}

// BackItUp walks through the local directory you specified, and backs it up to S3
func uploadToS3(localDir string, bucket, prefix string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("[ERROR] failed to load configuration: %v", err)
	}
	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	// Walk through the directory
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("[ERROR] failed to open file %s: %v", path, err)
		}
		defer file.Close()

		// Calculate the s3 key
		// relativePath, err := filepath.Rel(localDir, path)
		relativePath, err := filepath.Rel(MetaCfg.backupDir, path)
		if err != nil {
			return fmt.Errorf("[ERROR] failed to get relative path : %v", err)
		}
		s3Key := filepath.Join(prefix, relativePath)

		// Now Upload the file to s3
		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &bucket,
			Key:    &s3Key,
			Body:   file,
		})

		// Above the name of loacalDir would be trimmed, if you don't want that, can do:
		// for more: https://pkg.go.dev/path/filepath#Rel
		// err = uploadDirectory(s3Client, localDir, bucket, prefix)

		if err != nil {
			return fmt.Errorf("[ERROR] error uploading files: %v", err)
		}

		fmt.Printf("uploaded %s to s3://%s/%s\n", path, bucket, s3Key)
		return nil
	})

	if err != nil {
		log.Fatalf("Error during upload process: %v", err)
	}

	fmt.Println("\033[38;5;{51}mupload completed successfully!\033[0m")
	return nil
}

// S3Client is an interface for the S3 client, to make it testable(creating mocks)
type S3Client interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

func deleteFromS3(fileToDelete string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	// create s3 client
	client := s3.NewFromConfig(cfg)

	relativePath, err := filepath.Rel(MetaCfg.backupDir, fileToDelete)
	if err != nil {
		return err
	}
	s3Key := filepath.Join(MetaCfg.s3Prefix, relativePath)
	// s3Key := filepath.Join(MetaCfg.s3Prefix , fileToDelete)
	s3Key = strings.ReplaceAll(s3Key, "\\", "/") // Ensure forward slashes for S3 keys

	err = deleteS3Directory(context.TODO(), client, s3Key)

	if err != nil {
		return err
	}

	log.Println("Successfully deleted from S3: ", fileToDelete)
	return nil
}

func deleteS3Directory(ctx context.Context, client S3Client, key string) error {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(MetaCfg.s3Bucket),
		Prefix: aws.String(key),
	}

	for {
		output, err := client.ListObjectsV2(ctx, listInput)
		if err != nil {
			return err
		}

		for _, object := range output.Contents {
			if err := DeleteS3Object(ctx, client, *object.Key); err != nil {
				return err
			}
		}

		if !*output.IsTruncated {
			break
		}

		listInput.ContinuationToken = output.ContinuationToken
	}
	return nil
}

// Delete a single object from s3
func DeleteS3Object(ctx context.Context, client S3Client, key string) error {
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(MetaCfg.s3Bucket),
		Key:    aws.String(key),
	})
	return err
}
