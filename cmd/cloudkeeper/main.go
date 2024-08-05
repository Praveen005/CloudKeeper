package main

import (
	"context"

	"github.com/Praveen005/CloudKeeper/internal/backup"
	"github.com/Praveen005/CloudKeeper/internal/customlog"
	"github.com/Praveen005/CloudKeeper/internal/db"
	"github.com/Praveen005/CloudKeeper/internal/fsconfig"
	"github.com/Praveen005/CloudKeeper/internal/watcher"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	defer customlog.SyncLogger()
	run()
}

// run calls all the neccessary functions
func run() {
	var err error
	if err := godotenv.Load(); err != nil {
		// log.Println("[WARN] Error loading .env file:", err)
		customlog.Logger.Error("Error loading .env file:", zap.Error(err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fsconfig.MetaCfg, err = fsconfig.ParseConfig()
	if err != nil {
		customlog.Logger.Error("parsing config", zap.Error(err))
		return
	}
	go watcher.Watch(ctx)
	go db.FlushToDB(ctx)
	go backup.Backup(ctx)

	select {}
}
