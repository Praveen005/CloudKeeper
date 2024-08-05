package customlog

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global logger instance.
var Logger *zap.Logger

func init() {
	SetLogger()
}

// SetLogger sets up a logger with a specific configuration.
func SetLogger() {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:   "msg",
		LevelKey:     "level",
		TimeKey:      "time",
		CallerKey:    "caller",
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		EncodeTime:   zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeCaller: zapcore.ShortCallerEncoder,
	})

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)

	// AddCallerSkip is crucial for correct file and line number reporting.
	// When you wrap the zap logger in your own package, the caller information by default would point to your logging package rather than the actual calling site.
	// AddCallerSkip(1) tells the logger to skip one level up the call stack, correctly identifying where the log was called from in your main application.
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

// SyncLogger flushes any buffered log entries.
func SyncLogger() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
