// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

//go:build unix

package app

import (
	"os"
	"syscall"
)

// defaultSignals returns the default set of signals that trigger graceful
// shutdown. Override with WithSignals.
func defaultSignals() []os.Signal {
	return []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}
}
