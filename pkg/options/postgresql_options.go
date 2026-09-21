// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/spf13/pflag"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/onexstack/onexstack/pkg/db"
	gormlogger "github.com/onexstack/onexstack/pkg/logger/slog/gorm"
)

var _ IOptions = (*PostgreSQLOptions)(nil)

// PostgreSQLOptions defines options for postgresql database.
type PostgreSQLOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
	// SSLMode is the libpq sslmode (disable/allow/prefer/require/verify-ca/verify-full).
	SSLMode string `json:"sslmode,omitempty" mapstructure:"sslmode"`
	// SearchPath is the schema search path, e.g. "iam,public". Services that
	// share one database but own one schema each set this so unqualified table
	// names resolve to their own schema. Empty keeps the server default.
	SearchPath string `json:"search-path,omitempty" mapstructure:"search-path"`
}

// NewPostgreSQLOptions create a `zero` value instance.
func NewPostgreSQLOptions() *PostgreSQLOptions {
	return &PostgreSQLOptions{
		Addr:                  "127.0.0.1:5432",
		Username:              "onex",
		Password:              "onex(#)666",
		Database:              "onex",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
		SSLMode:               "disable",
	}
}

// Validate verifies flags passed to PostgreSQLOptions.
func (o *PostgreSQLOptions) Validate() []error {
	errs := []error{}

	if o.SSLMode != "" && !validSSLModes[o.SSLMode] {
		errs = append(errs, fmt.Errorf("invalid postgresql sslmode %q: must be one of "+
			"disable, allow, prefer, require, verify-ca, verify-full", o.SSLMode))
	}

	// The search path is interpolated into the connection string, so reject
	// anything that is not a plain schema list. Otherwise a typo silently
	// appends stray connection parameters instead of failing.
	if o.SearchPath != "" && !validSearchPath.MatchString(o.SearchPath) {
		errs = append(errs, fmt.Errorf("invalid postgresql search-path %q: expected a "+
			"comma-separated list of schema names, e.g. \"iam,public\"", o.SearchPath))
	}

	return errs
}

// validSSLModes is the closed set of libpq sslmode values.
var validSSLModes = map[string]bool{
	"disable":     true,
	"allow":       true,
	"prefer":      true,
	"require":     true,
	"verify-ca":   true,
	"verify-full": true,
}

// validSearchPath matches a comma-separated list of unquoted schema names.
var validSearchPath = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*(\s*,\s*[A-Za-z_][A-Za-z0-9_$]*)*$`)

// AddFlags adds flags related to postgresql storage for a specific APIServer to the specified FlagSet.
func (o *PostgreSQLOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, ""+
		"PostgreSQL service address. If left blank, the following related postgresql options will be ignored.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username for access to postgresql service.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, ""+
		"Password for access to postgresql, should be used pair with password.")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, ""+
		"Database name for the server to use.")
	fs.IntVar(&o.MaxIdleConnections, fullPrefix+".max-idle-connections", o.MaxIdleConnections, ""+
		"Maximum idle connections allowed to connect to postgresql.")
	fs.IntVar(&o.MaxOpenConnections, fullPrefix+".max-open-connections", o.MaxOpenConnections, ""+
		"Maximum open connections allowed to connect to postgresql.")
	fs.DurationVar(&o.MaxConnectionLifeTime, fullPrefix+".max-connection-life-time", o.MaxConnectionLifeTime, ""+
		"Maximum connection life time allowed to connect to postgresql.")
	fs.IntVar(&o.LogLevel, fullPrefix+".log-level", o.LogLevel, ""+
		"Specify gorm log level.")
	fs.StringVar(&o.SSLMode, fullPrefix+".sslmode", o.SSLMode, ""+
		"PostgreSQL SSL mode: disable, allow, prefer, require, verify-ca or verify-full.")
	fs.StringVar(&o.SearchPath, fullPrefix+".search-path", o.SearchPath, ""+
		"PostgreSQL schema search path, e.g. \"iam,public\".")
}

// NewDB create postgresql store with the given config.
func (o *PostgreSQLOptions) NewDB() (*gorm.DB, error) {
	opts := &db.PostgreSQLOptions{
		Addr:                  o.Addr,
		Username:              o.Username,
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		SSLMode:               o.SSLMode,
		SearchPath:            o.SearchPath,
		Logger:                gormlogger.New(slog.Default(), gormlogger.WithLogLevel(logger.LogLevel(o.LogLevel))),
	}

	return db.NewPostgreSQL(opts)
}
