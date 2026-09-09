// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package id

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithCodeChars(t *testing.T) {
	tests := []struct {
		name string
		arr  []rune
		want []rune
	}{
		{name: "custom", arr: []rune{'a', 'b', 'c'}, want: []rune{'a', 'b', 'c'}},
		{name: "nil-is-ignored", arr: nil, want: nil},
		{name: "empty-is-ignored", arr: []rune{}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &CodeOptions{}
			WithCodeChars(tt.arr)(opts)
			assert.Equal(t, tt.want, opts.chars)
		})
	}
}

func TestWithCodeN1(t *testing.T) {
	t.Parallel()

	opts := &CodeOptions{}
	WithCodeN1(9)(opts)
	assert.Equal(t, 9, opts.diffusionCoefficient)
}

func TestWithCodeN2(t *testing.T) {
	t.Parallel()

	opts := &CodeOptions{}
	WithCodeN2(3)(opts)
	assert.Equal(t, 3, opts.confusionCoefficient)
}

func TestWithCodeL(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int
	}{
		{name: "positive", length: 5, want: 5},
		{name: "zero-is-ignored", length: 0, want: 0},
		{name: "negative-is-ignored", length: -1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &CodeOptions{}
			WithCodeL(tt.length)(opts)
			assert.Equal(t, tt.want, opts.length)
		})
	}
}

func TestWithCodeSalt(t *testing.T) {
	tests := []struct {
		name string
		salt uint64
		want uint64
	}{
		{name: "positive", salt: 56789, want: 56789},
		{name: "zero-is-ignored", salt: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &CodeOptions{}
			WithCodeSalt(tt.salt)(opts)
			assert.Equal(t, tt.want, opts.salt)
		})
	}
}

func TestWithSonyflakeMachineID(t *testing.T) {
	tests := []struct {
		name string
		id   uint16
		want uint16
	}{
		{name: "positive", id: 123, want: 123},
		{name: "zero-is-ignored", id: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &SonyflakeOptions{}
			WithSonyflakeMachineID(tt.id)(opts)
			assert.Equal(t, tt.want, opts.machineID)
		})
	}
}

func TestWithSonyflakeStartTime(t *testing.T) {
	startTime := time.Date(2022, 10, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		startTime time.Time
		want      time.Time
	}{
		{name: "set", startTime: startTime, want: startTime},
		{name: "zero-is-ignored", startTime: time.Time{}, want: time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := &SonyflakeOptions{}
			WithSonyflakeStartTime(tt.startTime)(opts)
			assert.Equal(t, tt.want, opts.startTime)
		})
	}
}
