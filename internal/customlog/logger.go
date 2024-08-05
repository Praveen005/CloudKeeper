package customlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global logger instance.
var Logger *zap.Logger

func init() {
	// Initialize the logger
	initLogger()
}

func initLogger() {
	// Define level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	// High-priority output should go to standard error, and low-priority
	// output should go to standard out.
	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	// Optimize the console output for human operators.
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	// Combine the outputs, encoders, and level-handling functions into zapcore.Cores.
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	// Construct a Logger from the zapcore.Core.
	Logger = zap.New(core)
}

// SetLogger sets up a logger with a specific configuration.
func SetLogger() {
	config := zap.Config{
		Encoding:    "console", // Use console encoding for human-readable logs
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		OutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "msg",
			LevelKey:     "level",
			TimeKey:      "time",
			CallerKey:    "caller",
			EncodeLevel:  zapcore.CapitalColorLevelEncoder,
			EncodeTime:   zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

// SyncLogger flushes any buffered log entries.
func SyncLogger() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
