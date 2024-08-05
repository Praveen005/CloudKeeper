package db

import (
	"context"
	"log"
	"time"

	"github.com/Praveen005/CloudKeeper/internal/utils"
	bolt "go.etcd.io/bbolt"
)

// FileChangeEvent stores the action(add/remove) to be performed a given file path
type FileChangeEvent struct {
	Action string
}

// FilesToUpdate function stores the metadata(filepath and action to be performed on it) in-memory to be flushed later to DB
var FilesToUpdate map[string]FileChangeEvent

// Initializes the map at the start of the program
func init() {
	FilesToUpdate = make(map[string]FileChangeEvent)
}

// FlushToDB function runs a ticker to periodically call PersistData function and flush the metadata stored in-memory to the database for persistence (till the files get pushed to s3).
func FlushToDB(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(FilesToUpdate) == 0 { // If there is no metadata stored in-memory, just continue
				continue
			}
			log.Println("[Info] Pushing data to DB")
			err := PersistData()
			if err != nil {
				log.Println("[Error] error persisting the data: ", err)
				return
			}
			log.Println("[Info] Data pushed successfully, printing now...")
			utils.PrintData()
			FilesToUpdate = make(map[string]FileChangeEvent) // Clear the map, since data has been persisted
		case <-ctx.Done():
			return
		}
	}
}

// PersistData function stores the metadata to database
func PersistData() error {
	// creates and opens a database at the given path. If the file does not exist then it will be created automatically.
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("filesToUpdate")) // We don't ave tables here, we have buckets
		if err != nil {
			return err
		}
		// Persist data
		// files that are to be added to s3
		for path, fileChangeEvent := range FilesToUpdate {
			exists := b.Get([]byte(path)) != nil
			if exists {
				continue
			}
			err := b.Put([]byte(path), []byte(fileChangeEvent.Action))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
