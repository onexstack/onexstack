// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

//go:build windows

package app

import (
	"os"
)

// defaultSignals returns the default set of signals that trigger graceful
// shutdown. Windows only supports os.Interrupt (SIGTERM/SIGQUIT are not
// available). Override with WithSignals.
func defaultSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
