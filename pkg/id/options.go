// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package id

import "time"

// CodeOptions holds the configurable parameters used by NewCode to encode a
// uint64 ID into a short, human-readable code.
type CodeOptions struct {
	chars                []rune
	diffusionCoefficient int
	confusionCoefficient int
	length               int
	salt                 uint64
}

// WithCodeChars sets the character set used by NewCode. An empty argument is
// ignored; otherwise the provided set replaces the built-in ambiguous-free
// default.
func WithCodeChars(arr []rune) func(*CodeOptions) {
	return func(options *CodeOptions) {
		if len(arr) > 0 {
			getCodeOptionsOrSetDefault(options).chars = arr
		}
	}
}

// WithCodeN1 sets the diffusion coefficient. It must be coprime with the
// length of the character set.
func WithCodeN1(n int) func(*CodeOptions) {
	return func(options *CodeOptions) {
		getCodeOptionsOrSetDefault(options).diffusionCoefficient = n
	}
}

// WithCodeN2 sets the confusion coefficient. It must be coprime with the
// output code length.
func WithCodeN2(n int) func(*CodeOptions) {
	return func(options *CodeOptions) {
		getCodeOptionsOrSetDefault(options).confusionCoefficient = n
	}
}

// WithCodeL sets the output code length.
func WithCodeL(l int) func(*CodeOptions) {
	return func(options *CodeOptions) {
		if l > 0 {
			getCodeOptionsOrSetDefault(options).length = l
		}
	}
}

// WithCodeSalt sets the salt used to scramble the input ID. Different salts
// produce entirely different codes for the same ID.
func WithCodeSalt(salt uint64) func(*CodeOptions) {
	return func(options *CodeOptions) {
		if salt > 0 {
			getCodeOptionsOrSetDefault(options).salt = salt
		}
	}
}

func getCodeOptionsOrSetDefault(options *CodeOptions) *CodeOptions {
	if options == nil {
		return &CodeOptions{
			// base string set, remove 0,1,I,O,U,Z
			chars: []rune{
				'2', '3', '4', '5', '6',
				'7', '8', '9', 'A', 'B',
				'C', 'D', 'E', 'F', 'G',
				'H', 'J', 'K', 'L', 'M',
				'N', 'P', 'Q', 'R', 'S',
				'T', 'V', 'W', 'X', 'Y',
			},
			// diffusionCoefficient / len(chars)=30 are coprime
			diffusionCoefficient: 17,
			// confusionCoefficient / length are coprime
			confusionCoefficient: 5,
			// output code length
			length: 8,
			// random number
			salt: 123567369,
		}
	}
	return options
}

// SonyflakeOptions holds the configurable parameters used by NewSonyflake.
type SonyflakeOptions struct {
	machineID uint16
	startTime time.Time
}

// WithSonyflakeMachineID sets the machine ID. Production deployments must
// assign a unique machine ID to each instance.
func WithSonyflakeMachineID(id uint16) func(*SonyflakeOptions) {
	return func(options *SonyflakeOptions) {
		if id > 0 {
			getSonyflakeOptionsOrSetDefault(options).machineID = id
		}
	}
}

// WithSonyflakeStartTime sets the epoch from which IDs are generated. Do not
// change this value in a running deployment, or previously generated IDs may
// collide.
func WithSonyflakeStartTime(startTime time.Time) func(*SonyflakeOptions) {
	return func(options *SonyflakeOptions) {
		if !startTime.IsZero() {
			getSonyflakeOptionsOrSetDefault(options).startTime = startTime
		}
	}
}

func getSonyflakeOptionsOrSetDefault(options *SonyflakeOptions) *SonyflakeOptions {
	if options == nil {
		return &SonyflakeOptions{
			machineID: 1,
			startTime: time.Date(2022, 10, 10, 0, 0, 0, 0, time.UTC),
		}
	}
	return options
}
