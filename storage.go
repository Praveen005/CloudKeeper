package main

import (
	"context"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

// flushToDB function runs a ticker to periodically call persistData function and flush the metadata stored in-memory to the database for persistence (till the files get pushed to s3).
func flushToDB(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(FilesToUpdate) == 0 { // If there is no metadata stored in-memory, just continue
				continue
			}
			log.Println("[Info] Pushing data to DB")
			err := persistData()
			if err != nil {
				log.Println("[Error] error persisting the data: ", err)
				return
			}
			log.Println("[Info] Data pushed successfully, printing now...")
			printData()
			FilesToUpdate = make(map[string]FileChangeEvent) // Clear the map, since data has been persisted
		case <-ctx.Done():
			return
		}
	}
}

// persistData function stores the metadata to database
func persistData() error {
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

// printData function is a utility function to print the data present in db
func printData() {
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		log.Println("[ERROR] error printing: ", err)
		return
	}
	defer db.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		// Printing data from bucket
		bu := tx.Bucket([]byte("filesToUpdate"))
		if bu == nil {
			log.Println("[ERROR] bucket not found")
			return nil
		}
		err := bu.ForEach(func(k, v []byte) error {
			log.Printf("Filepath: %s, Action: %s\n", k, v)
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

// flushToS3 function calls the deleteFromS3 or uploadToS3 function as per value of the action field specified for a file path in the metadata
func flushToS3() error {
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
				err = uploadToS3(fileName, MetaCfg.s3Bucket, MetaCfg.s3Prefix)
			} else if action == "remove" {
				err = deleteFromS3(fileName)
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
