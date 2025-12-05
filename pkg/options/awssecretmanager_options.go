package options

import (
	"fmt"
	"regexp"

	"github.com/spf13/pflag"
)

// Ensure interface
var _ IOptions = (*AWSSecretManagerOptions)(nil)

// AWSSecretManagerOptions contains minimal configuration to read a secret
// from AWS Secrets Manager.
type AWSSecretManagerOptions struct {
	// Region is the AWS region where the secret resides, e.g. "ap-southeast-1".
	Region string `json:"region" mapstructure:"region"`

	// SecretName is the identifier of the secret in AWS Secrets Manager,
	// e.g. "sre-redis-platform/-/ro".
	SecretName string `json:"secret-name" mapstructure:"secret-name"`
}

// NewAWSSecretManagerOptions creates an options instance with sensible defaults.
func NewAWSSecretManagerOptions() *AWSSecretManagerOptions {
	return &AWSSecretManagerOptions{
		Region:     "ap-southeast-1",
		SecretName: "",
	}
}

// Validate checks required fields and basic constraints.
func (o *AWSSecretManagerOptions) Validate() []error {
	if o == nil {
		return nil
	}
	var errs []error

	if o.Region == "" {
		errs = append(errs, fmt.Errorf("awssm.region is required"))
	} else {
		// loose sanity check, accepts standard patterns like us-east-1, ap-southeast-1, eu-west-3
		re := regexp.MustCompile(`^[a-z]{2}-[a-z]+-\d$`)
		if !re.MatchString(o.Region) {
			// don't block non-standard partitions; just warn via error text
			errs = append(errs, fmt.Errorf("awssm.region %q looks unusual; expected like 'ap-southeast-1' or 'us-east-1'", o.Region))
		}
	}

	if o.SecretName == "" {
		errs = append(errs, fmt.Errorf("awssm.secret-name is required"))
	}

	return errs
}

// AddFlags registers AWS Secrets Manager related flags to the specified FlagSet,
// using fullPrefix as the complete prefix for flag names.
func (o *AWSSecretManagerOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Region, fullPrefix+".region", o.Region,
		"AWS region where the secret resides, e.g. ap-southeast-1.")
	fs.StringVar(&o.SecretName, fullPrefix+".secret-name", o.SecretName,
		"Secret identifier in AWS Secrets Manager, e.g. sre-redis-platform/-/ro.")
}
