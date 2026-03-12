package options

import (
	"time"

	"github.com/spf13/pflag"
)

// Ensure the interface is implemented.
var _ IOptions = (*SecureServingOptions)(nil)

// SecureServingOptions defines the configuration for secure communication.
// It adopts a "Smart Mode": CA, Cert, and Key fields can be either file paths
// or the actual content (PEM string/Base64) itself.
type SecureServingOptions struct {
	// Addr is the address the server listens on (e.g. :443).
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout specifies the TLS timeout (e.g., for ReadHeaderTimeout).
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	TLSOptions `json:",inline" mapstructure:",squash"`
}

// NewSecureServingOptions creates a zero-value instance of SecureServingOptions with defaults.
func NewSecureServingOptions() *SecureServingOptions {
	return &SecureServingOptions{
		Addr:       ":443",           // Set default port to 443
		Timeout:    10 * time.Second, // Set default timeout to 10 seconds
		TLSOptions: NewTLSOptions(),
	}
}

// Validate verifies the flags passed to SecureServingOptions.
func (o *SecureServingOptions) Validate() []error {
	errs := []error{}
	errs = append(errs, o.TLSOptions.Validate()...)
	return errs
}

// AddFlags adds flags to the specified FlagSet.
func (o *SecureServingOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, "The address the secure server listens on (e.g. :443).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout for TLS connections.")

	o.TLSOptions.AddFlags(fs, fullPrefix)
}
