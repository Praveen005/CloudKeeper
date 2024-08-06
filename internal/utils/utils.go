package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/Praveen005/CloudKeeper/internal/customlog"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// PrintData function is a utility function to print the data present in db
func PrintData() {
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		customlog.Logger.Error("failed to create/open database at the given path",
			zap.String("path", "filesToS3.db"),
			zap.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		// Printing data from bucket
		bu := tx.Bucket([]byte("filesToUpdate"))
		if bu == nil {
			customlog.Logger.Error("bucket not found",
				zap.String("bucket", "filesToUpdate"),
			)
			os.Exit(1)
		}
		err := bu.ForEach(func(k, v []byte) error {
			fmt.Printf("Filepath: %s, Action: %s\n", k, v)
			return nil
		})
		if err != nil {
			return fmt.Errorf("error reading data from database: %v", err)
		}
		return nil
	}); err != nil {
		customlog.Logger.Error("reading from database failed")
		os.Exit(1)
	}
}
