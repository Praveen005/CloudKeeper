package s3client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Praveen005/CloudKeeper/internal/customlog"
	"github.com/Praveen005/CloudKeeper/internal/fsconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// S3Client is an interface for the S3 client, to make it testable(creating mocks)
type S3Client interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// UploadToS3 walks through the local directory you specified, and backs it up to S3
func UploadToS3(localDir string, bucket, prefix string) error {
	customlog.Logger.Debug("starting file upload to s3")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
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
			return fmt.Errorf("failed to open file %s: %v", path, err)
		}
		defer file.Close()

		// Calculate the s3 key
		// relativePath, err := filepath.Rel(localDir, path)
		relativePath, err := filepath.Rel(fsconfig.MetaCfg.BackupDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path : %v", err)
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
			return fmt.Errorf("error uploading files: %v", err)
		}

		customlog.Logger.Debug("File uploaded to S3",
			zap.String("file", relativePath),
			zap.String("bucket", bucket),
			zap.String("s3Key", s3Key),
		)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error during upload process: %v", err)
	}

	return nil
}

// DeleteFromS3 function deletes objects from s3 bucket
func DeleteFromS3(fileToDelete string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("error loading AWS configuration: %v", err)
	}

	// create s3 client
	client := s3.NewFromConfig(cfg)

	/* Let's understand what's happening here:

	fsconfig.MetaCfg.BackupDir is the directory that you want to backup, say it looks like: '/home/praveen/fsnotifyTest'
	And from the file change event, you get the following file path which has the file you want to push to s3:
		'/home/praveen/fsnotifyTest/sample21/folder1/files34.txt'

	And in your s3 bucket you want to store it in say 's3folder', so say your s3 prefix is 's3folder/'
	So, you would want to store like: 's3folder/sample21/folder1/files34.txt'

	for that, you need to trim, '/home/praveen/fsnotifyTest' from '/home/praveen/fsnotifyTest/sample21/folder1/files34.txt'. And this is what 'filepath.Rel()' does.
	*/
	relativePath, err := filepath.Rel(fsconfig.MetaCfg.BackupDir, fileToDelete)
	if err != nil {
		return fmt.Errorf("error resolving relative path: %v", err)
	}
	s3Key := filepath.Join(fsconfig.MetaCfg.S3Prefix, relativePath)
	s3Key = strings.ReplaceAll(s3Key, "\\", "/") // Ensure forward slashes for S3 keys

	err = DeleteS3Directory(context.TODO(), client, s3Key)

	if err != nil {
		return fmt.Errorf("error deleting file(s): %v", err)
	}

	customlog.Logger.Info("All files successfully deleted from S3")
	return nil
}

// DeleteS3Directory function, traverses through all the files in a directory marked to be deleted and deletes them
func DeleteS3Directory(ctx context.Context, client S3Client, key string) error {
	customlog.Logger.Debug("Fetching file(s) to delete from s3 bucket",
		zap.String("s3key", key),
	)

	// specify the parameters for listing objects in the S3 bucket and filtering the results to only include objects whose keys start with a certain prefix.
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(fsconfig.MetaCfg.S3Bucket),
		Prefix: aws.String(key),
	}

	// Run it till there is no more object to fetch from the bucket
	for {
		output, err := client.ListObjectsV2(ctx, listInput) // gets you the objects from the bucket specified
		if err != nil {
			return fmt.Errorf("error listing objects from s3: %v", err)
		}

		// Deletes files sequentially by calling DeleteS3Object function
		for _, object := range output.Contents {
			if err := DeleteS3Object(ctx, client, *object.Key); err != nil {
				return fmt.Errorf("error deleting the file: %v", err)
			}
		}

		// Check if all of the results were returned
		if !*output.IsTruncated {
			break
		}

		// ContinuationToken is used for pagination of the list response, in one go all the objects are not listed, hence the infinite for loop :)
		listInput.ContinuationToken = output.ContinuationToken
	}
	return nil
}

// DeleteS3Object function deletes a single object from s3
func DeleteS3Object(ctx context.Context, client S3Client, key string) error {
	customlog.Logger.Debug("Deleting a file from s3 bucket",
		zap.String("file(s3key)", key),
	)
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fsconfig.MetaCfg.S3Bucket),
		Key:    aws.String(key),
	})
	return err
}
