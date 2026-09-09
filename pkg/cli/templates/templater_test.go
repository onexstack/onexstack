// Copyright 2026 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onexstack.

package templates

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/cobra"
)

// TestInheritedFlagsShown verifies that a subcommand's usage output lists the
// persistent flags inherited from its parent command (the behavior change added
// on top of the upstream kubectl templates).
func TestInheritedFlagsShown(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("config", "", "config file")

	child := &cobra.Command{Use: "child", Run: func(*cobra.Command, []string) {}}
	child.Flags().String("local", "", "local flag")
	root.AddCommand(child)

	tmpl := &templater{
		RootCmd:       root,
		UsageTemplate: MainUsageTemplate(),
		HelpTemplate:  MainHelpTemplate(),
	}

	tpl := template.New("usage")
	tpl.Funcs(tmpl.templateFuncs())
	template.Must(tpl.Parse(tmpl.UsageTemplate))

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, child); err != nil {
		t.Fatalf("execute usage template: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"--config", "--local"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in rendered usage, got:\n%s", want, out)
		}
	}
}