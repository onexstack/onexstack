// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import "time"

// DefaultShutdownTimeout is the default duration for graceful shutdown hooks.
const DefaultShutdownTimeout = 10 * time.Second
