package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultBackupInterval = 65 * time.Second
)

// MetaConfig contains the meta data needed to backup your files to s3
type MetaConfig struct {
	backupDir      string
	s3Bucket       string
	s3Prefix       string
	backupInterval time.Duration
}

var MetaCfg MetaConfig

func main() {
	run()
}

func run() {
	var err error
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Error loading .env file:", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	MetaCfg, err = parseConfig()
	if err != nil {
		log.Printf("parsing config: %v", err)
		return
	}

	go watch(ctx)
	go flushToDB(ctx)
	go backup(ctx)

	select {}
}

func parseConfig() (MetaConfig, error) {

	var localDir, bucket, prefix string

	// Define flags for some meta informations you want to get though command line
	flag.StringVar(&localDir, "d", "", "local directory to backup")
	flag.StringVar(&bucket, "b", "", "bucket name")
	flag.StringVar(&prefix, "p", "", "Object prefix name")
	flag.Parse()

	// Get the directory which you want to backup.
	// read from env. variable or the flag variable if specified.
	MetaCfg.backupDir = getConfigValue(localDir, "BACKUP_DIR")
	if MetaCfg.backupDir == "" {
		return MetaCfg, fmt.Errorf("no backup directory specified")
	}

	// Get the name of s3 bucket into which you want to backup.
	// read from env. variable or the flag variable if specified.
	MetaCfg.s3Bucket = getConfigValue(bucket, "S3_BUCKET")
	if MetaCfg.s3Bucket == "" {
		return MetaCfg, fmt.Errorf("no s3 bucket specified")
	}

	// Get the filepath(or say prefix) from your s3 bucket which will be prefixed to your directory name.
	// read from env. variable or the flag variable if specified.
	MetaCfg.s3Prefix = getConfigValue(prefix, "S3_BUCKET_PREFIX")
	if MetaCfg.s3Prefix == "" {
		return MetaCfg, fmt.Errorf("no s3 bucket prefix specified")
	}

	// We will be using this in a while, when we run this binary in background
	backupIntervalStr := os.Getenv("BACKUP_INTERVAL")
	if backupIntervalStr == "" {
		MetaCfg.backupInterval = defaultBackupInterval
	} else {
		backupIntervalInt, err := strconv.Atoi(backupIntervalStr)
		if err != nil {
			return MetaCfg, fmt.Errorf("invalid BACKUP_INTERVAL: %w", err)
		}
		MetaCfg.backupInterval = time.Duration(backupIntervalInt) * time.Minute
	}

	return MetaCfg, nil
}

// Gets you the metadata to populate MetaConfig
func getConfigValue(flagValue, envVar string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVar)
}
