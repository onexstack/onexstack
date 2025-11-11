package resty

import (
	"fmt"
	"log/slog"
)

// Logger is a Breeze logger implementation using log/slog.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new logger using the default slog instance.
func NewLogger() *Logger {
	return &Logger{logger: slog.Default()}
}

// Errorf 输出 Error 级日志
func (l *Logger) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

// Warnf 输出 Warn 级日志
func (l *Logger) Warnf(format string, v ...any) {}

// Debugf 输出 Debug 级日志
func (l *Logger) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}
