package fsconfig

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultS3BackupInterval      = 24 * time.Hour
	defaultDBPersistenceInterval = 10 * time.Minute
)

// MetaConfig holds the configuration settings needed to back up files to S3.
type MetaConfig struct {
	BackupDir             string
	S3Bucket              string
	S3Prefix              string
	S3BackupInterval      time.Duration
	DBPersistenceInterval time.Duration
}

// MetaCfg is a MetaConfig instance
var MetaCfg MetaConfig

// ParseConfig retrieves the required data and stores in MetaCfg
func ParseConfig() (MetaConfig, error) {

	var localDir, bucket, prefix string

	// Define flags for some meta informations you want to get though command line
	flag.StringVar(&localDir, "d", "", "local directory to backup")
	flag.StringVar(&bucket, "b", "", "bucket name")
	flag.StringVar(&prefix, "p", "", "Object prefix name")
	flag.Parse()

	// Get the directory which you want to backup.
	// read from env. variable or the flag variable if specified.
	MetaCfg.BackupDir = getConfigValue(localDir, "BACKUP_DIR")
	if MetaCfg.BackupDir == "" {
		return MetaCfg, fmt.Errorf("no backup directory specified")
	}

	// Get the name of s3 bucket into which you want to backup.
	// read from env. variable or the flag variable if specified.
	MetaCfg.S3Bucket = getConfigValue(bucket, "S3_BUCKET")
	if MetaCfg.S3Bucket == "" {
		return MetaCfg, fmt.Errorf("no s3 bucket specified")
	}

	// Get the filepath(or say prefix) from your s3 bucket which will be prefixed to your directory name.
	// read from env. variable or the flag variable if specified.
	MetaCfg.S3Prefix = getConfigValue(prefix, "S3_BUCKET_PREFIX")
	if MetaCfg.S3Prefix == "" {
		return MetaCfg, fmt.Errorf("no s3 bucket prefix specified")
	}

	// After every `S3BackupInterval`, files will be updated to s3
	S3BackupIntervalStr := os.Getenv("S3_BACKUP_INTERVAL")
	if S3BackupIntervalStr == "" {
		MetaCfg.S3BackupInterval = defaultS3BackupInterval
	} else {
		S3BackupIntervalInt, err := strconv.Atoi(S3BackupIntervalStr)
		if err != nil {
			return MetaCfg, fmt.Errorf("invalid BACKUP_INTERVAL: %w", err)
		}
		MetaCfg.S3BackupInterval = time.Duration(S3BackupIntervalInt) * time.Minute
	}

	// After every `DBPersistenceInterval`, files will be updated to s3
	DBPersistenceIntervalStr := os.Getenv("S3_BACKUP_INTERVAL")
	if DBPersistenceIntervalStr == "" {
		MetaCfg.DBPersistenceInterval = defaultDBPersistenceInterval
	} else {
		DBPersistenceIntervalInt, err := strconv.Atoi(DBPersistenceIntervalStr)
		if err != nil {
			return MetaCfg, fmt.Errorf("invalid BACKUP_INTERVAL: %w", err)
		}
		MetaCfg.DBPersistenceInterval = time.Duration(DBPersistenceIntervalInt) * time.Minute
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
