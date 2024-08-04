package main

import (
	"context"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

func flushToDB(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// if len(FilesToAdd) == 0 && len(FilesToRemove) == 0 {
			// 	continue
			// }
			if len(FilesToUpdate) == 0 {
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
			// FilesToAdd = make(map[string]FileChangeEvent)
			// FilesToRemove = make(map[string]FileChangeEvent)
			FilesToUpdate = make(map[string]FileChangeEvent)
		case <-ctx.Done():
			return
		}
	}
}

func persistData() error {
	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("filesToUpdate"))
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

	// err = db.Batch(func(tx *bolt.Tx) error {
	// 	b, err := tx.CreateBucketIfNotExists([]byte("filesToDelete"))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// files that are to be removed from s3
	// 	for path, fileChangeEvent := range FilesToRemove {
	// 		exists := b.Get([]byte(path)) != nil
	// 		if exists {
	// 			continue
	// 		}
	// 		err := b.Put([]byte(path), []byte(fileChangeEvent.Action))
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	return err
	// }

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

		// // Printing data from delete bucket
		// bd := tx.Bucket([]byte("filesToDelete"))
		// if bd == nil {
		// 	log.Println("[ERROR] bucket not found")
		// 	return nil
		// }
		// err = bd.ForEach(func(k, v []byte) error {
		// 	log.Printf("Filepath: %s, Action: %s\n", k, v)
		// 	return nil
		// })
		// if err != nil{
		// 	return err
		// }
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

// func manageS3Update()error{
// 	err := flushFromS3()
// 	if err != nil{
// 		return err
// 	}
// 	err = flushToS3()
// 	if err != nil{
// 		return err
// 	}
// 	return nil
// }

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

// func flushFromS3() error {
// 	db, err := bolt.Open("filesToS3.db", 0666, &bolt.Options{Timeout: 2 * time.Minute})
// 	if err != nil {
// 		log.Println("[ERROR] error printing: ", err)
// 		return err
// 	}
// 	defer db.Close()

// 	if err := db.Update(func(tx *bolt.Tx) error {
// 		b := tx.Bucket([]byte("filesToDelete"))

// 		err := b.ForEach(func(k, v []byte) error {
// 			fileName := string(k)
// 			action := string(v)
// 			// var err error

// 			err := deleteFromS3(fileName)

// 			if err != nil {
// 				log.Printf("[ERROR] Error processing file %s (action: %s): %v", fileName, action, err)
// 				return err
// 			}
// 			// if deleted sucessfully from S3, delete from db
// 			b.Delete(k)

// 			return nil
// 		})
// 		return err
// 	}); err != nil {
// 		return err
// 	}

// 	return nil
// }
