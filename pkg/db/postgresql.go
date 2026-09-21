// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgreSQLOptions defines options for PostgreSQL database.
type PostgreSQLOptions struct {
	Addr                  string
	Username              string
	Password              string
	Database              string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	// SSLMode is the libpq sslmode. Empty means "disable", which suits a local
	// server. Managed instances normally require "require" or stricter.
	SSLMode string
	// SearchPath is the schema search path, e.g. "iam,public". When set, an
	// unqualified table name resolves through it, which lets several services
	// share a single database while owning one schema each. Keep "public" at the
	// end so shared functions remain reachable. Empty keeps the server default.
	SearchPath string
	// +optional
	Logger logger.Interface
}

// DSN return DSN from PostgreSQLOptions.
func (o *PostgreSQLOptions) DSN() string {
	splited := strings.Split(o.Addr, ":")
	host, port := splited[0], "5432"
	if len(splited) > 1 {
		port = splited[1]
	}

	sslmode := o.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf(`user=%s password=%s host=%s port=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai`,
		quoteDSNValue(o.Username),
		quoteDSNValue(o.Password),
		host,
		port,
		quoteDSNValue(o.Database),
		sslmode,
	)

	if o.SearchPath != "" {
		dsn += " search_path=" + quoteDSNValue(o.SearchPath)
	}

	return dsn
}

// quoteDSNValue quotes a libpq keyword/value argument when it contains
// characters that would otherwise terminate the value or start the next
// argument. Without this a password containing a space silently truncates the
// DSN and the connection fails with a confusing parse error.
func quoteDSNValue(v string) string {
	if v != "" && !strings.ContainsAny(v, ` '\`) {
		return v
	}

	return `'` + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + `'`
}

// NewPostgreSQL create a new gorm db instance with the given options.
func NewPostgreSQL(opts *PostgreSQLOptions) (*gorm.DB, error) {
	// Set default values to ensure all fields in opts are available.
	setPostgreSQLDefaults(opts)

	db, err := gorm.Open(postgres.Open(opts.DSN()), &gorm.Config{
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

// setPostgreSQLDefaults set available default values for some fields.
func setPostgreSQLDefaults(opts *PostgreSQLOptions) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:5432"
	}
	if opts.MaxIdleConnections == 0 {
		opts.MaxIdleConnections = 100
	}
	if opts.MaxOpenConnections == 0 {
		opts.MaxOpenConnections = 100
	}
	if opts.MaxConnectionLifeTime == 0 {
		opts.MaxConnectionLifeTime = time.Duration(10) * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
}
