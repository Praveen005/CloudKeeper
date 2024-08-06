package backup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Praveen005/CloudKeeper/internal/customlog"
	"github.com/Praveen005/CloudKeeper/internal/fsconfig"
	"github.com/Praveen005/CloudKeeper/internal/s3client"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// Backup function periodically calls the flushToS3 function to flush the data(files) to s3
func Backup(ctx context.Context) {
	ticker := time.NewTicker(fsconfig.MetaCfg.S3BackupInterval)
	customlog.Logger.Debug("Inside backup function",
		zap.String("Backup Interval", fsconfig.MetaCfg.S3BackupInterval.String()))

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			customlog.Logger.Debug("Ticker ticked: starting file(s) update to S3")

			if err := FlushToS3(); err != nil {
				customlog.Logger.Error("Flushing data to s3 failed",
					zap.String("error", err.Error()),
				)
				os.Exit(1)
			}
			customlog.Logger.Info("Success! all updates to s3 completed")
		case <-ctx.Done():
			customlog.Logger.Warn("[Inside Backup] Context cancellation signal received. Shutting down gracefully.")
			return
		}
	}
}

// FlushToS3 function calls the deleteFromS3 or uploadToS3 function as per value of the action field specified for a file path in the metadata
func FlushToS3() error {
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		return fmt.Errorf("failed to create/open database at the given path: %v", err)
	}
	defer db.Close()

	if err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("filesToUpdate"))

		err := b.ForEach(func(k, v []byte) error {
			fileName := string(k)
			action := string(v)
			var err error
			if action == "add" {
				err = s3client.UploadToS3(fileName, fsconfig.MetaCfg.S3Bucket, fsconfig.MetaCfg.S3Prefix)
			} else if action == "remove" {
				err = s3client.DeleteFromS3(fileName)
			}

			if err != nil {
				return fmt.Errorf("error processing file %s (action: %s): %v", fileName, action, err)
			}
			// if successfully uploaded, delete from db
			b.Delete(k)

			return nil
		})
		return err
	}); err != nil {
		return fmt.Errorf("error updating database: %v", err)
	}
	return nil
}
