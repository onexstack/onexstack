// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package id

import (
	"context"
	"testing"
	"time"

	"github.com/sony/sonyflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSonyflake(t *testing.T) {
	tests := []struct {
		name    string
		options []func(*SonyflakeOptions)
		wantErr bool
	}{
		{
			name: "default",
		},
		{
			name:    "with-machine-id",
			options: []func(*SonyflakeOptions){WithSonyflakeMachineID(123)},
		},
		{
			name:    "future-start-time",
			options: []func(*SonyflakeOptions){WithSonyflakeStartTime(time.Now().Add(time.Hour))},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sf, err := NewSonyflake(tt.options...)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, sf)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, sf)
		})
	}
}

func TestSonyflake_ID(t *testing.T) {
	t.Run("generates-unique-non-zero-ids", func(t *testing.T) {
		t.Parallel()

		sf, err := NewSonyflake(WithSonyflakeMachineID(7))
		require.NoError(t, err)

		seen := make(map[uint64]struct{}, 100)
		for i := 0; i < 100; i++ {
			id, err := sf.ID(context.Background())
			require.NoError(t, err)
			assert.NotZero(t, id)
			assert.Equal(t, uint64(7), sonyflake.MachineID(id))

			if _, ok := seen[id]; ok {
				assert.Failf(t, "expected unique ID", "duplicate ID %d", id)
			}
			seen[id] = struct{}{}
		}
	})
}
