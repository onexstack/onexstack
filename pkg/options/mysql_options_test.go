// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestMySQLOptionsAddFlagsMaxIdle(t *testing.T) {
	opts := &MySQLOptions{
		MaxIdleConnections: 5,
		MaxOpenConnections: 10,
	}
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs, "mysql")

	got, err := fs.GetInt("mysql.max-idle-connections")
	if err != nil {
		t.Fatalf("flag not registered: %v", err)
	}
	if got != 5 {
		t.Fatalf("max-idle-connections default = %d, want 5", got)
	}
}

func TestMySQLOptionsAddFlagsAddrAndLogLevel(t *testing.T) {
	opts := NewMySQLOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs, "mysql")

	// flag 名应与 json/mapstructure tag 一致（此前 Addr 误用 .host）.
	if _, err := fs.GetString("mysql.addr"); err != nil {
		t.Fatalf("mysql.addr flag not registered: %v", err)
	}
	// flag 名应与 tag 一致（此前 LogLevel 误用 .log-mode）.
	if _, err := fs.GetInt("mysql.log-level"); err != nil {
		t.Fatalf("mysql.log-level flag not registered: %v", err)
	}
}
