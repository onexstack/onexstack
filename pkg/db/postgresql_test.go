// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package db

import (
	"strings"
	"testing"
)

func TestPostgreSQLOptionsDSN(t *testing.T) {
	tests := []struct {
		name       string
		opts       PostgreSQLOptions
		wantParts  []string
		absentFrom []string
	}{
		{
			name: "defaults keep sslmode disable and omit search_path",
			opts: PostgreSQLOptions{Addr: "127.0.0.1:5432", Username: "onex", Password: "p", Database: "onex"},
			wantParts: []string{
				"host=127.0.0.1", "port=5432", "dbname=onex", "sslmode=disable",
			},
			// Unset SearchPath must not appear at all: an empty value would
			// otherwise inject `search_path=` and blank out the server default.
			absentFrom: []string{"search_path"},
		},
		{
			name: "explicit sslmode wins over the default",
			opts: PostgreSQLOptions{Addr: "db:5432", Username: "u", Password: "p", Database: "d", SSLMode: "require"},
			wantParts: []string{"sslmode=require"},
		},
		{
			name: "search_path is appended for schema-per-service",
			opts: PostgreSQLOptions{Addr: "db:5432", Username: "u", Password: "p", Database: "d", SearchPath: "iam,public"},
			wantParts: []string{"search_path=iam,public"},
		},
		{
			name: "port defaults to 5432 when addr has none",
			opts: PostgreSQLOptions{Addr: "db", Username: "u", Password: "p", Database: "d"},
			wantParts: []string{"host=db", "port=5432"},
		},
		{
			// A password with a space would otherwise terminate the value and
			// turn the rest into stray connection parameters.
			name:      "password with spaces is quoted",
			opts:      PostgreSQLOptions{Addr: "db:5432", Username: "u", Password: "pa ss", Database: "d"},
			wantParts: []string{`password='pa ss'`},
		},
		{
			name:      "password with a quote is escaped",
			opts:      PostgreSQLOptions{Addr: "db:5432", Username: "u", Password: `p'\x`, Database: "d"},
			wantParts: []string{`password='p\'\\x'`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			got := opts.DSN()

			for _, want := range tt.wantParts {
				if !strings.Contains(got, want) {
					t.Errorf("DSN() = %q, missing %q", got, want)
				}
			}
			for _, absent := range tt.absentFrom {
				if strings.Contains(got, absent) {
					t.Errorf("DSN() = %q, must not contain %q", got, absent)
				}
			}
		})
	}
}

func TestQuoteDSNValue(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "''"},
		{"plain", "plain"},
		{"pa ss", "'pa ss'"},
		{`a'b`, `'a\'b'`},
		{`a\b`, `'a\\b'`},
	}

	for _, tt := range tests {
		if got := quoteDSNValue(tt.in); got != tt.want {
			t.Errorf("quoteDSNValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
