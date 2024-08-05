package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	var err error
	if err := godotenv.Load(); err != nil {
		// log.Println("[WARN] Error loading .env file:", err)
		customlog.Logger.Error("Error loading .env file", zap.String("error", err.Error()))
		return
	}

	fsconfig.MetaCfg, err = fsconfig.ParseConfig()
	if err != nil {
		customlog.Logger.Error("parsing config", zap.String("error", err.Error()))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		watcher.Watch(ctx)
	}()

	go func() {
		defer wg.Done()
		db.FlushToDB(ctx)
	}()

	go func() {
		defer wg.Done()
		backup.Backup(ctx)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	customlog.Logger.Info("Received shutdown signal. Initiating graceful shutdown...")

	// Cancel the context to signal all goroutines to stop
	cancel()

	// Wait for all goroutines to finish with a timeout
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		customlog.Logger.Info("All goroutines have completed. Shutting down.")
	case <-time.After(30 * time.Second):
		customlog.Logger.Warn("Timeout waiting for goroutines to finish. Forcing shutdown.")
	case <-sigChan:
		customlog.Logger.Warn("Received second interrupt signal. Forcing immediate shutdown.")
		os.Exit(0)
	}

	customlog.Logger.Info("Shutdown complete.")
	os.Exit(0)
}
