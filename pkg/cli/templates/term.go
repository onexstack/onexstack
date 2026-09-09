// Copyright 2026 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack.

package templates

import "io"

const defaultWordWrapperLimit = 80

// newResponsiveWriter returns w unchanged. The upstream term.NewResponsiveWriter
// installs terminal-resize handling; a passthrough writer is used here to avoid
// pulling the k8s.io/kubectl dependency into this module.
func newResponsiveWriter(w io.Writer) io.Writer { return w }

// getWordWrapperLimit returns a fixed wrap limit in columns. The upstream term
// reads the terminal width dynamically; a fixed 80 keeps long flag help lines
// readable without terminal-size dependencies.
func getWordWrapperLimit() (uint, error) { return defaultWordWrapperLimit, nil }