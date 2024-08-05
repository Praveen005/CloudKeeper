package utils

import (
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

// PrintData function is a utility function to print the data present in db
func PrintData() {
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
