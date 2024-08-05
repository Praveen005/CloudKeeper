package main

import (
	"context"
	"log"

	"github.com/Praveen005/CloudKeeper/fsconfig"
	"github.com/Praveen005/CloudKeeper/internal/backup"
	"github.com/Praveen005/CloudKeeper/internal/db"
	"github.com/Praveen005/CloudKeeper/internal/watcher"
	"github.com/joho/godotenv"
)

func main() {
	run()
}

// run calls all the neccessary functions
func run() {
	var err error
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Error loading .env file:", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsconfig.MetaCfg, err = fsconfig.ParseConfig()
	if err != nil {
		log.Printf("parsing config: %v", err)
		return
	}

	go watcher.Watch(ctx)
	go db.FlushToDB(ctx)
	go backup.Backup(ctx)

	select {}
}
