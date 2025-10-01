package store

import (
	krtlog "github.com/go-kratos/kratos/v2/log"
	"k8s.io/klog/v2"
)

// Logger is a logger that implements the Logger interface.
// It uses the log package to log error messages with additional context.
type Logger struct{}

// NewLogger creates and returns a new instance of Logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Error logs an error message with the provided context using the log package.
// func (l *Logger) Log(ctx context.Context, err error, msg string, kvs ...any) {
func (l *Logger) Log(level krtlog.Level, keyvals ...any) error {
	switch level {
	case krtlog.LevelDebug:
		klog.V(2).InfoS("", keyvals...)
	case krtlog.LevelInfo:
		klog.InfoS("", keyvals...)
	case krtlog.LevelWarn:
		klog.V(1).InfoS("", keyvals...)
	case krtlog.LevelError:
		klog.InfoS("", keyvals...)
	case krtlog.LevelFatal:
		klog.Fatalln(keyvals...)
	}

	return nil
}
