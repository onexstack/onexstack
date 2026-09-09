// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package id

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkNewCode(b *testing.B) {
	for b.Loop() {
		NewCode(1)
	}
}

func BenchmarkNewCodeTimeConsuming(b *testing.B) {
	id := NewCode(1)
	assert.Equal(b, "VHB4JX86", id)

	for b.Loop() {
		NewCode(1)
	}
}
