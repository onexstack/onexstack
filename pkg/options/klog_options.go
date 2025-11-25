// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"time"

	"github.com/spf13/pflag"
	logsapi "k8s.io/component-base/logs/api/v1"
)

var _ IOptions = (*LogOptions)(nil)

// LogOptions contains configuration items related to log.
type LogOptions struct {
	// Format Flag specifies the structure of log messages.
	// default value of format is `text`
	Format string `json:"format,omitempty" mapstructure:"format"`
	// Maximum number of nanoseconds (i.e. 1s = 1000000000) between log
	// flushes. Ignored if the selected logging backend writes log
	// messages without buffering.
	FlushFrequency time.Duration `json:"flush-frequency" mapstructure:"flush-frequency"`
	// Verbosity is the threshold that determines which log messages are
	// logged. Default is zero which logs only the most important
	// messages. Higher values enable additional messages. Error messages
	// are always logged.
	Verbosity logsapi.VerbosityLevel `json:"verbosity" mapstructure:"verbosity"`
}

// NewLogOptions creates an Options object with default parameters.
func NewLogOptions() *LogOptions {
	c := logsapi.LoggingConfiguration{}
	logsapi.SetRecommendedLoggingConfiguration(&c)

	return &LogOptions{
		Format:         c.Format,
		FlushFrequency: c.FlushFrequency.Duration.Duration,
		Verbosity:      c.Verbosity,
	}
}

// Validate verifies flags passed to LogOptions.
func (o *LogOptions) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags adds command line flags for the configuration.
func (o *LogOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Format, fullPrefix+".format", o.Format, "Sets the log format. Permitted formats: json, text.")
	fs.DurationVar(&o.FlushFrequency, fullPrefix+".flush-frequency", o.FlushFrequency, "Maximum number of seconds between log flushes.")
	fs.VarP(logsapi.VerbosityLevelPflag(&o.Verbosity), fullPrefix+".verbosity", "", " Number for the log level verbosity.")
}

func (o *LogOptions) Native() *logsapi.LoggingConfiguration {
	c := logsapi.LoggingConfiguration{}
	logsapi.SetRecommendedLoggingConfiguration(&c)
	c.Format = o.Format
	if o.FlushFrequency != 0 {
		c.FlushFrequency.Duration.Duration = o.FlushFrequency
		c.FlushFrequency.SerializeAsString = true
	}
	c.Verbosity = o.Verbosity
	return &c
}
