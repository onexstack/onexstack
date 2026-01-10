package empty

import (
	"context"
	"time"

	"gorm.io/gorm/logger"
)

// SilentLogger is a custom silent logger implementation.
type SilentLogger struct{}

// NewSilentLogger is the constructor.
func NewSilentLogger() logger.Interface {
	return &SilentLogger{}
}

// LogMode implements the interface method.
// Regardless of the log level set, it returns the current SilentLogger to remain silent.
func (l *SilentLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info is a no-op implementation.
func (l *SilentLogger) Info(ctx context.Context, s string, args ...interface{}) {
	// Do nothing
}

// Warn is a no-op implementation.
func (l *SilentLogger) Warn(ctx context.Context, s string, args ...interface{}) {
	// Do nothing
}

// Error is a no-op implementation.
func (l *SilentLogger) Error(ctx context.Context, s string, args ...interface{}) {
	// Do nothing
}

// Trace is a no-op implementation.
// This is the most critical part. Not calling fc() avoids the expensive overhead of SQL string formatting.
func (l *SilentLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// Do nothing
}
