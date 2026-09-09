// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package id

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

const (
	// initialRetryInterval is the delay before the first retry when NextID fails.
	initialRetryInterval = time.Millisecond
	// maxRetryInterval caps the exponential backoff so the retry loop never
	// overflows the interval and degrades into a busy loop.
	maxRetryInterval = time.Second
)

// Sonyflake wraps a sonyflake generator to produce distributed unique IDs.
type Sonyflake struct {
	ops SonyflakeOptions
	sf  *sonyflake.Sonyflake
}

// NewSonyflake returns a new Sonyflake ID generator configured by the given
// options, or an error if the generator cannot be created with those settings.
func NewSonyflake(options ...func(*SonyflakeOptions)) (*Sonyflake, error) {
	ops := getSonyflakeOptionsOrSetDefault(nil)
	for _, f := range options {
		f(ops)
	}

	st := sonyflake.Settings{
		StartTime: ops.startTime,
	}
	if ops.machineID > 0 {
		st.MachineID = func() (uint16, error) {
			return ops.machineID, nil
		}
	}

	ins := sonyflake.NewSonyflake(st)
	if ins == nil {
		return nil, errors.New("failed to create sonyflake")
	}
	if _, err := ins.NextID(); err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	return &Sonyflake{
		ops: *ops,
		sf:  ins,
	}, nil
}

// ID returns a new unique uint64 ID, retrying with exponential backoff until
// it succeeds or ctx is cancelled.
func (s *Sonyflake) ID(ctx context.Context) (uint64, error) {
	id, err := s.sf.NextID()
	if err == nil {
		return id, nil
	}

	interval := initialRetryInterval
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-timer.C:
		}

		id, err = s.sf.NextID()
		if err == nil {
			return id, nil
		}

		if interval < maxRetryInterval {
			interval *= 2
			if interval > maxRetryInterval {
				interval = maxRetryInterval
			}
		}
		timer.Reset(interval)
	}
}
