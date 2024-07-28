package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultBackupInterval = 15 * time.Minute
)

// MetaConfig contains the meta data needed to backup your files to s3
type MetaConfig struct {
	backupDir      string
	s3Bucket       string
	s3Prefix       string
	backupInterval time.Duration
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] Error loading .env file:", err)
	}

	cfg, err := parseConfig()
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	log.Printf("[INFO] Starting backup process for directory: %s", cfg.backupDir)
	if err := BackItUp(cfg.backupDir, int(cfg.backupInterval.Minutes()), cfg.s3Bucket, cfg.s3Prefix); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	return nil
}

func parseConfig() (MetaConfig, error) {
	var cfg MetaConfig
	var localDir, bucket, prefix string

	// Define flags for some meta informations you want to get though command line
	flag.StringVar(&localDir, "d", "", "local directory to backup")
	flag.StringVar(&bucket, "b", "", "bucket name")
	flag.StringVar(&prefix, "p", "", "Object prefix name")
	flag.Parse()

	// Get the directory which you want to backup.
	// read from env. variable or the flag variable if specified.
	cfg.backupDir = getConfigValue(localDir, "BACKUP_DIR")
	if cfg.backupDir == "" {
		return cfg, fmt.Errorf("no backup directory specified")
	}

	// Get the name of s3 bucket into which you want to backup.
	// read from env. variable or the flag variable if specified.
	cfg.s3Bucket = getConfigValue(bucket, "S3_BUCKET")
	if cfg.s3Bucket == "" {
		return cfg, fmt.Errorf("no s3 bucket specified")
	}

	// Get the filepath(or say prefix) from your s3 bucket which will be prefixed to your directory name.
	// read from env. variable or the flag variable if specified.
	cfg.s3Prefix = getConfigValue(prefix, "S3_BUCKET_PREFIX")
	if cfg.s3Prefix == "" {
		return cfg, fmt.Errorf("no s3 bucket prefix specified")
	}

	// We will be using this in a while, when we run this binary in background
	backupIntervalStr := os.Getenv("BACKUP_INTERVAL")
	if backupIntervalStr == "" {
		cfg.backupInterval = defaultBackupInterval
	} else {
		backupIntervalInt, err := strconv.Atoi(backupIntervalStr)
		if err != nil {
			return cfg, fmt.Errorf("invalid BACKUP_INTERVAL: %w", err)
		}
		cfg.backupInterval = time.Duration(backupIntervalInt) * time.Minute
	}

	return cfg, nil
}

// Gets you the metadata to populate MetaConfig
func getConfigValue(flagValue, envVar string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVar)
}
