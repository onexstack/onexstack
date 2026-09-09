// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestKafkaOptionsAddFlagsRequiredAcks(t *testing.T) {
	opts := NewKafkaOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs, "kafka")

	if fs.Lookup("kafka.writer.required-acks") == nil {
		t.Fatal("expected flag kafka.writer.required-acks to be registered")
	}
	if fs.Lookup("kafka.required-acks") != nil {
		t.Fatal("unexpected flag kafka.required-acks (should be under the writer prefix)")
	}
}
