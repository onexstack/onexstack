package options

import (
	"log/slog"
	"time"

	"github.com/spf13/pflag"
	"gorm.io/gorm"

	"github.com/onexstack/onexstack/pkg/db"
	gormlogger "github.com/onexstack/onexstack/pkg/logger/slog/gorm"
)

var _ IOptions = (*SQLiteOptions)(nil)

// SQLiteOptions defines options for the SQLite database.
type SQLiteOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`         // SQLite database file path
	Username              string        `json:"username,omitempty" mapstructure:"username"` // Reserved field for interface compatibility
	Password              string        `json:"-" mapstructure:"password"`                  // Reserved field
	Database              string        `json:"database" mapstructure:"database"`           // SQLite file name or path
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewSQLiteOptions creates a default SQLiteOptions instance.
func NewSQLiteOptions() *SQLiteOptions {
	return &SQLiteOptions{
		Addr:                  "./data/sqlite.db", // Can be a file path or ":memory:" (in-memory database)
		Username:              "",                 // Username/password not required, kept for config compatibility
		Password:              "",
		Database:              "", // Uses Addr when empty
		MaxIdleConnections:    2,
		MaxOpenConnections:    5,
		MaxConnectionLifeTime: time.Duration(30) * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate verifies flags passed to SQLiteOptions.
func (o *SQLiteOptions) Validate() []error {
	errs := []error{}
	return errs
}

// AddFlags injects SQLite-related configuration flags into the given FlagSet for an API server.
func (o *SQLiteOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.Addr, join(prefixes...)+"sqlite.path", o.Addr,
		"SQLite database file path (e.g. ./data/app.db or file::memory:?cache=shared).")
	fs.StringVar(&o.Username, join(prefixes...)+"sqlite.username", o.Username,
		"Username (reserved field for configuration consistency).")
	fs.StringVar(&o.Password, join(prefixes...)+"sqlite.password", o.Password,
		"Password (reserved field for configuration consistency).")
	fs.StringVar(&o.Database, join(prefixes...)+"sqlite.database", o.Database,
		"Database name or override path for SQLite.")
	fs.IntVar(&o.MaxIdleConnections, join(prefixes...)+"sqlite.max-idle-connections", o.MaxIdleConnections,
		"Maximum number of idle connections to SQLite.")
	fs.IntVar(&o.MaxOpenConnections, join(prefixes...)+"sqlite.max-open-connections", o.MaxOpenConnections,
		"Maximum number of open connections to SQLite.")
	fs.DurationVar(&o.MaxConnectionLifeTime, join(prefixes...)+"sqlite.max-connection-life-time", o.MaxConnectionLifeTime,
		"Maximum lifetime of a SQLite database connection.")
	fs.IntVar(&o.LogLevel, join(prefixes...)+"sqlite.log-mode", o.LogLevel,
		"Specify GORM log level.")
}

// DSN builds the SQLite DSN from options.
func (o *SQLiteOptions) DSN() string {
	if o.Database != "" {
		return o.Database
	}
	if o.Addr != "" {
		return o.Addr
	}
	// Default database file name
	return "./data/sqlite.db"
}

// NewDB creates a GORM SQLite database connection.
func (o *SQLiteOptions) NewDB() (*gorm.DB, error) {
	opts := &db.SQLiteOptions{
		Addr:                  o.Addr,
		Username:              o.Username, // Unused for SQLite but kept for interface compatibility
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		Logger:                gormlogger.New(slog.Default()),
	}

	return db.NewSQLite(opts)
}
