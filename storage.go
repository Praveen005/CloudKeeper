package main

import (
	"context"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)


func flushToDB(ctx context.Context){
	ticker := time.NewTicker(1 *time.Minute)
	defer ticker.Stop()
	for{
		select{
		case <- ticker.C:
			if len(FilesToUpdate) == 0{
				continue
			}
			log.Println("[Info] Pushing data to DB")
			err := persistData()
			if err != nil{
				log.Println("[Error] error persisting the data: ", err)
				return
			}
			log.Println("[Info] Data pushed successfully, printing now...")
			printData()
			FilesToUpdate = make(map[string]FileChangeEvent)
		case <-ctx.Done():
			return
		}
	}
}

func persistData() error {
	db, err := bolt.Open("filesToS3.db", 0666,  &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil{
		return err
	}
	defer db.Close()

	err = db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("filesToUpload"))
		if err != nil{
			return err
		}
		// Persist data
		for path, fileChangeEvent := range FilesToUpdate{
			exists := b.Get([]byte(path)) != nil
			if exists{
				continue
			}
			err := b.Put([]byte(path), []byte(fileChangeEvent.Action))
			if err != nil{
				return err
			}
		}
		return nil
	})
	if err != nil{
		return err
	}

	return nil
}

func printData(){
	db, err := bolt.Open("filesToS3.db", 0666,  &bolt.Options{Timeout: 2 * time.Minute})
	if err != nil{
		log.Println("[ERROR] error printing: ", err)
		return
	}
	defer db.Close()

	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("filesToUpload"))
		if b == nil {
			log.Println("[ERROR] bucket not found")
			return nil
		}
		err := b.ForEach(func(k, v []byte) error {
			log.Printf("Filepath: %s, Action: %s\n", k, v)
			return nil
		})
		return err
	}); err != nil {
		log.Fatal(err)
	}
}