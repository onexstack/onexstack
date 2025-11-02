package gormslog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// SlogLogger implements gorm.io/gorm/logger.Interface using Go's log/slog package.
type SlogLogger struct {
	Logger                    *slog.Logger
	LogLevel                  logger.LogLevel // GORM log level
	SlowThreshold             time.Duration   // Slow query threshold
	IgnoreRecordNotFoundError bool            // Whether to ignore RecordNotFound errors
}

// New returns a new SlogLogger instance with sensible defaults.
func New(l *slog.Logger) *SlogLogger {
	return &SlogLogger{
		Logger:                    l,
		LogLevel:                  logger.Info,
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode changes the logger's log level and returns a new instance.
func (l *SlogLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs informational messages.
func (l *SlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.Logger.Log(ctx, slog.LevelInfo, fmt.Sprintf(msg, data...))
	}
}

// Warn logs warnings.
func (l *SlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.Logger.Log(ctx, slog.LevelWarn, fmt.Sprintf(msg, data...))
	}
}

// Error logs errors.
func (l *SlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.Logger.Log(ctx, slog.LevelError, fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL statements, duration, rows, and errors.
func (l *SlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []any{
		slog.String("sql", sql),
		slog.String("duration", elapsed.String()),
		slog.Int64("rows", rows),
	}

	switch {
	case err != nil && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		if l.LogLevel >= logger.Error {
			fields = append(fields, slog.String("error", err.Error()))
			l.Logger.Log(ctx, slog.LevelError, "SQL execution failed", fields...)
		}
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		fields = append(fields, slog.String("warning", "SLOW QUERY"))
		l.Logger.Log(ctx, slog.LevelWarn, "slow SQL query", fields...)
	case l.LogLevel == logger.Info:
		l.Logger.Log(ctx, slog.LevelInfo, "SQL", fields...)
	}
}
