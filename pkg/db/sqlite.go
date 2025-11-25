package db

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteOptions defines options for SQLite database.
type SQLiteOptions struct {
	Addr                  string // For SQLite, this will be the database file path
	Username              string // Not used in SQLite, kept for consistency
	Password              string // Not used in SQLite, kept for consistency
	Database              string // Database file name/path (primary field for SQLite)
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	// +optional
	Logger logger.Interface
}

// DSN return DSN from SQLiteOptions.
func (o *SQLiteOptions) DSN() string {
	// For SQLite, we use Database field as the primary file path
	// If Database is empty, fall back to Addr field
	dsn := o.Database
	if dsn == "" && o.Addr != "" {
		dsn = o.Addr
	}

	// If still empty, use default
	if dsn == "" {
		dsn = "app.db"
	}

	return dsn
}

// NewSQLite create a new gorm db instance with the given options.
func NewSQLite(opts *SQLiteOptions) (*gorm.DB, error) {
	// Set default values to ensure all fields in opts are available.
	setSQLiteDefaults(opts)

	db, err := gorm.Open(sqlite.Open(opts.DSN()), &gorm.Config{
		// PrepareStmt executes the given query in cached statement.
		// This can improve performance.
		PrepareStmt: true,
		Logger:      opts.Logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}

// setSQLiteDefaults set available default values for some fields.
func setSQLiteDefaults(opts *SQLiteOptions) {
	// For consistency with PostgreSQL, we check Addr first, then Database
	if opts.Addr == "" && opts.Database == "" {
		opts.Database = "app.db" // Default SQLite database file
	}

	// SQLite typically works better with fewer connections due to file locking
	if opts.MaxIdleConnections == 0 {
		opts.MaxIdleConnections = 2 // Conservative for SQLite
	}
	if opts.MaxOpenConnections == 0 {
		opts.MaxOpenConnections = 5 // Limit to prevent locking issues
	}
	if opts.MaxConnectionLifeTime == 0 {
		opts.MaxConnectionLifeTime = time.Duration(10) * time.Second // Same as PostgreSQL
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
}

// NewInMemorySQLite creates a new in-memory SQLite database instance.
// This is useful for testing or temporary data storage.
func NewInMemorySQLite(dbFile string) (*gorm.DB, error) {
	opts := &SQLiteOptions{
		// Configure the database using SQLite memory mode
		// ?cache=shared is used to set SQLite's cache mode to shared cache mode.
		// By default, each SQLite database connection has its own private cache. This mode is called private cache.
		// Using shared cache mode allows different connections to share the same in-memory database and cache.
		Database:              fmt.Sprintf("file:%s?cache=shared", dbFile),
		MaxIdleConnections:    1,
		MaxOpenConnections:    1,
		MaxConnectionLifeTime: time.Hour,
		Logger:                logger.Default,
	}

	return NewSQLite(opts)
}
