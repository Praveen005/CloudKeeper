package backup

import (
	"context"
	"log"
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
	// fmt.Println("Inside backup function, backup interval: ", fsconfig.MetaCfg.BackupInterval)
	customlog.Logger.Info("Inside backup function",
		zap.String("backup_interval", fsconfig.MetaCfg.S3BackupInterval.String()))

	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// log.Println("[INFO] starting files updation in S3")
			customlog.Logger.Debug("starting files updation in S3")

			if err := FlushToS3(); err != nil {
				log.Fatalf("backup failed: %v", err)
			}
			log.Printf("[Info] Success! files updated in s3.")
		case <-ctx.Done():
			return
		}
	}
}

// FlushToS3 function calls the deleteFromS3 or uploadToS3 function as per value of the action field specified for a file path in the metadata
func FlushToS3() error {
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		log.Println("[ERROR] error printing: ", err)
		return err
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
				log.Printf("[ERROR] Error processing file %s (action: %s): %v", fileName, action, err)
				return err
			}
			// if successfully uploaded, delete from db
			b.Delete(k)

			return nil
		})
		return err
	}); err != nil {
		return err
	}
	log.Println("All updates to s3 completed successfully!")
	return nil
}
