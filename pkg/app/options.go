// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import (
	"github.com/spf13/pflag"
)

// OptionsValidator provides error-style validation. Implementations should
// return a single error aggregating any validation failures (e.g. via
// errors.Join), or nil when the options are valid.
type OptionsValidator interface {
	// Validate validates all the required options.
	Validate() error
}

// OptionsCompleter is implemented by options that need post-processing after flags
// are bound and unmarshaled but before validation and the RunFunc.
type OptionsCompleter interface {
	// Complete fills in derived or defaulted fields after flag binding.
	Complete() error
}

// NamedFlagSet represents a named group of flags.
type NamedFlagSet struct {
	// Name is the section name for the flag group.
	Name string
	*pflag.FlagSet
}

// NamedFlagSets is a collection of named flag sets, organized by section name.
// This is a self-defined replacement for k8s.io/component-base/cli/flag.NamedFlagSets.
type NamedFlagSets struct {
	FlagSets []NamedFlagSet
}

// NewNamedFlagSets creates a new NamedFlagSets collection.
func NewNamedFlagSets() *NamedFlagSets {
	return &NamedFlagSets{}
}

// FlagSet returns the flag set with the given name, creating it if needed.
func (n *NamedFlagSets) FlagSet(name string) *pflag.FlagSet {
	for _, f := range n.FlagSets {
		if f.Name == name {
			return f.FlagSet
		}
	}
	fs := pflag.NewFlagSet(name, pflag.ContinueOnError)
	fs.SetNormalizeFunc(pflag.CommandLine.GetNormalizeFunc())
	n.FlagSets = append(n.FlagSets, NamedFlagSet{Name: name, FlagSet: fs})
	return fs
}

// NamedFlagSetOptions provides access to named flag sets and embedding
// the OptionsValidator interface for validation.
type NamedFlagSetOptions interface {
	// Flags returns the named flag sets for this options instance.
	Flags() NamedFlagSets

	OptionsValidator
}

// FlagSetOptions defines an interface for command-line options that can
// add themselves to a flag set and perform validation.
type FlagSetOptions interface {
	// AddFlags adds command-specific flags to the provided flag set.
	AddFlags(fs *pflag.FlagSet)

	OptionsValidator
}
