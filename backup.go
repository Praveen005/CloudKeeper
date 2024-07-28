package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// BackItUp walks through the local directory you specified, and backs it up to S3
func BackItUp(localDir string, backupInterval int, bucket, prefix string) error {
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
		relativePath, err := filepath.Rel(localDir, path)
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
