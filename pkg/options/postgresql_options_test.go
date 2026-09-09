// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestPostgreSQLOptionsAddFlagsMaxIdle(t *testing.T) {
	opts := &PostgreSQLOptions{
		MaxIdleConnections: 5,
		MaxOpenConnections: 10,
	}
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs, "postgresql")

	got, err := fs.GetInt("postgresql.max-idle-connections")
	if err != nil {
		t.Fatalf("flag not registered: %v", err)
	}
	if got != 5 {
		t.Fatalf("max-idle-connections default = %d, want 5", got)
	}
}
