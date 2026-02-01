package options

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// Ensure the interface is implemented.
var _ IOptions = (*TLSOptions)(nil)

// TLSOptions defines the configuration for secure communication.
// It adopts a "Smart Mode": CA, Cert, and Key fields can be either file paths
// or the actual content (PEM string/Base64) itself.
type TLSOptions struct {
	// Enabled determines whether to enable TLS transport.
	Enabled bool `json:"enabled" mapstructure:"enabled"`

	// SkipVerify controls whether a client verifies the server's certificate chain and host name.
	SkipVerify bool `json:"skip-verify" mapstructure:"skip-verify"`

	// Smart Fields:
	// These can be absolute file paths (e.g., /etc/certs/ca.pem)
	// OR raw PEM data (e.g., -----BEGIN CERTIFICATE-----...)
	// OR Base64 encoded PEM data.
	CA   string `json:"ca" mapstructure:"ca"`
	Cert string `json:"cert" mapstructure:"cert"`
	Key  string `json:"key" mapstructure:"key"`
}

// NewTLSOptions creates a zero-value instance of TLSOptions.
func NewTLSOptions() *TLSOptions {
	return &TLSOptions{}
}

// Validate verifies the flags passed to TLSOptions.
func (o *TLSOptions) Validate() []error {
	errs := []error{}

	if !o.Enabled {
		return errs
	}

	// Both Cert and Key must be provided, or neither (for client-side usage without mTLS).
	if (o.Cert != "" && o.Key == "") || (o.Cert == "" && o.Key != "") {
		errs = append(errs, fmt.Errorf("both cert and key must be provided to enable mTLS"))
	}

	return errs
}

// AddFlags adds flags to the specified FlagSet.
func (o *TLSOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.BoolVar(&o.Enabled, fullPrefix+".enabled", o.Enabled, "Enable TLS transport.")
	fs.BoolVar(&o.SkipVerify, fullPrefix+".skip-verify", o.SkipVerify, "Insecurely skip TLS certificate verification.")

	// Simplified flags with descriptions indicating support for multiple formats.
	fs.StringVar(&o.CA, fullPrefix+".ca", o.CA, "Path to CA file OR raw PEM data.")
	fs.StringVar(&o.Cert, fullPrefix+".cert", o.Cert, "Path to Cert file OR raw PEM data.")
	fs.StringVar(&o.Key, fullPrefix+".key", o.Key, "Path to Key file OR raw PEM data.")
}

// MustTLSConfig returns a tls.Config or returns a default insecure config (or panics) on error.
func (o *TLSOptions) MustTLSConfig() *tls.Config {
	tlsConf, err := o.TLSConfig()
	if err != nil {
		// In production, you might want to panic or log the error.
		// Returning a default insecure config here is for fail-safe scenarios.
		return &tls.Config{InsecureSkipVerify: true}
	}
	return tlsConf
}

// TLSConfig generates the final tls.Config object.
func (o *TLSOptions) TLSConfig() (*tls.Config, error) {
	if !o.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: o.SkipVerify,
	}

	// 1. Load Client Certificate (Cert + Key)
	// Use the smart loader to handle file paths or raw data.
	certBytes, err := loadResource(o.Cert)
	if err != nil {
		return nil, fmt.Errorf("failed to load cert: %w", err)
	}
	keyBytes, err := loadResource(o.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load key: %w", err)
	}

	if len(certBytes) > 0 && len(keyBytes) > 0 {
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse x509 key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 2. Load CA Certificate
	caBytes, err := loadResource(o.CA)
	if err != nil {
		return nil, fmt.Errorf("failed to load ca: %w", err)
	}

	if len(caBytes) > 0 {
		capool := x509.NewCertPool()
		if !capool.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("failed to append ca certs from pem")
		}
		tlsConfig.RootCAs = capool
	}

	return tlsConfig, nil
}

// Scheme returns the URL scheme based on the TLS configuration.
func (o *TLSOptions) Scheme() string {
	if o.Enabled {
		return "https"
	}
	return "http"
}

// loadResource intelligently identifies whether the input is a "File Path",
// "Raw PEM String", or "Base64 Encoded String".
func loadResource(input string) ([]byte, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	// 1. High Priority: If it contains PEM header, return as is.
	// This avoids unnecessary disk I/O.
	if strings.Contains(input, "-----BEGIN") {
		return []byte(input), nil
	}

	// 2. Try to identify if it is a file path.
	// Logic: If no newlines and length is reasonable for a path, check file existence.
	if !strings.Contains(input, "\n") && len(input) < 1024 {
		_, err := os.Stat(input)
		if err == nil {
			return os.ReadFile(input)
		}
		// If file doesn't exist, do not error immediately; it might be a Base64 string.
	}

	// 3. Try Base64 decoding.
	// K8s Secrets often inject data as Base64 strings.
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err == nil {
		// If decoded successfully, verify if the content looks like a certificate/key.
		// This prevents misidentifying a random string (that happens to be valid Base64) as cert data.
		if strings.Contains(string(decoded), "-----BEGIN") {
			return decoded, nil
		}
	}

	// 4. Fallback: Neither a file, nor raw PEM, nor valid Base64 encoded PEM.
	// This usually means the user provided an invalid path or invalid data.
	return nil, fmt.Errorf("input is neither a valid file path, nor raw PEM data, nor base64 encoded PEM")
}
