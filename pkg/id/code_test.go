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

func TestNewCode(t *testing.T) {
	type args struct {
		id      uint64
		options []func(*CodeOptions)
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "default", args: args{id: 1}, want: "VHB4JX86"},
		{
			name: "with-options",
			args: args{
				id: 1,
				options: []func(*CodeOptions){
					WithCodeChars([]rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}),
					WithCodeN1(9),
					WithCodeN2(3),
					WithCodeL(5),
					WithCodeSalt(56789),
				},
			},
			want: "80773",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, NewCode(tt.args.id, tt.args.options...))
		})
	}
}
