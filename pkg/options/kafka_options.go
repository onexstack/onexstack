package options

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"k8s.io/klog/v2"

	stringsutil "github.com/onexstack/onexstack/pkg/util/strings"
)

var _ IOptions = (*KafkaOptions)(nil)

// franzLogger adapts klog to franz-go's logger interface.
type franzLogger struct{}

// Level implements kgo.Logger.
func (l franzLogger) Level() kgo.LogLevel {
	// 这里可以根据 klog 的全局设置动态返回，或者默认返回 Info
	// franz-go 会调用这个方法来决定是否调用 Log
	return kgo.LogLevelDebug
}

// Log implements kgo.Logger.
func (l franzLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	// Map franz-go levels to klog V-levels
	var vLevel klog.Level
	switch level {
	case kgo.LogLevelError:
		vLevel = 1
	case kgo.LogLevelWarn:
		vLevel = 2
	case kgo.LogLevelInfo:
		vLevel = 3
	case kgo.LogLevelDebug:
		vLevel = 4
	default:
		vLevel = 3
	}
	klog.V(vLevel).InfoS(msg, keyvals...)
}

type WriterOptions struct {
	// Limit on how many attempts will be made to deliver a message.
	// Default: 10
	MaxAttempts int `mapstructure:"max-attempts"`

	// Number of acknowledges from partition replicas required.
	// -1: AllISR (WaitForAll), 1: Leader, 0: NoResponse.
	// Default: -1
	RequiredAcks int `mapstructure:"required-acks"`

	// Setting this flag to true causes the produce to be asynchronous in the client usage.
	Async bool `mapstructure:"async"`

	// Maximum number of bytes to buffer before sending a batch.
	// Default: 1MB
	BatchBytes int `mapstructure:"batch-bytes"`

	// Max time to wait for a batch to fill.
	// Default: 1s
	BatchTimeout time.Duration `mapstructure:"batch-timeout"`
}

type ReaderOptions struct {
	// GroupID holds the optional consumer group id.
	GroupID string `mapstructure:"group-id"`

	// MinBytes indicates to the broker the minimum batch size that the consumer will accept.
	// Default: 1
	MinBytes int `mapstructure:"min-bytes"`

	// MaxBytes indicates to the broker the maximum batch size that the consumer will accept.
	// Default: 1MB
	MaxBytes int `mapstructure:"max-bytes"`

	// Maximum amount of time to wait for new data to come when fetching batches.
	// Default: 10s
	MaxWait time.Duration `mapstructure:"max-wait"`

	// HeartbeatInterval sets the frequency at which the reader sends the consumer group heartbeat.
	// Default: 3s
	HeartbeatInterval time.Duration `mapstructure:"heartbeat-interval"`

	// SessionTimeout sets the length of time the coordinator will wait for members to send a heartbeat.
	// Default: 10s
	SessionTimeout time.Duration `mapstructure:"session-timeout"`

	// RebalanceTimeout sets the length of time the coordinator will wait for members to join as part of a rebalance.
	// Default: 30s
	RebalanceTimeout time.Duration `mapstructure:"rebalance-timeout"`

	// StartOffset determines where to start if no committed offset is found.
	// -1: Newest (Latest), -2: Oldest (Earliest) - Adapting legacy kafka-go constants
	StartOffset int64 `mapstructure:"start-offset"`
}

// KafkaOptions defines options for kafka cluster.
type KafkaOptions struct {
	Brokers       []string      `mapstructure:"brokers"`
	Topic         string        `mapstructure:"topic"`
	ClientID      string        `mapstructure:"client-id"`
	Timeout       time.Duration `mapstructure:"timeout"`
	TLSOptions    *TLSOptions   `mapstructure:"tls"`
	SASLMechanism string        `mapstructure:"mechanism"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	Algorithm     string        `mapstructure:"algorithm"`
	Compressed    bool          `mapstructure:"compressed"`

	WriterOptions WriterOptions `mapstructure:"writer"`
	ReaderOptions ReaderOptions `mapstructure:"reader"`
}

// NewKafkaOptions create a `zero` value instance.
func NewKafkaOptions() *KafkaOptions {
	return &KafkaOptions{
		TLSOptions: NewTLSOptions(),
		Timeout:    3 * time.Second,
		WriterOptions: WriterOptions{
			RequiredAcks: -1, // -1 in franz-go means AllISR
			MaxAttempts:  10,
			Async:        true,
			BatchTimeout: 1 * time.Second,
			BatchBytes:   1 * MiB,
		},
		ReaderOptions: ReaderOptions{
			MinBytes:          1,
			MaxBytes:          1 * MiB,
			MaxWait:           5 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			SessionTimeout:    10 * time.Second,
			RebalanceTimeout:  30 * time.Second,
			StartOffset:       -2, // Default to Earliest (kafka-go FirstOffset=-2)
		},
	}
}

// Validate verifies flags passed to KafkaOptions.
func (o *KafkaOptions) Validate() []error {
	errs := []error{}

	if len(o.Brokers) == 0 {
		errs = append(errs, fmt.Errorf("kafka broker can not be empty"))
	}

	if !o.TLSOptions.Enabled && o.SASLMechanism != "" {
		errs = append(errs, fmt.Errorf("SASL-Mechanism is setted but use_ssl is false"))
	}

	if !stringsutil.StringIn(strings.ToLower(o.SASLMechanism), []string{"plain", "scram", ""}) {
		errs = append(errs, fmt.Errorf("doesn't support '%s' SASL mechanism", o.SASLMechanism))
	}

	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--kafka.timeout cannot be negative"))
	}

	if o.WriterOptions.BatchTimeout <= 0 {
		errs = append(errs, fmt.Errorf("--kafka.writer.batch-timeout cannot be negative"))
	}

	errs = append(errs, o.TLSOptions.Validate()...)

	return errs
}

// AddFlags adds flags related to redis storage for a specific APIServer to the specified FlagSet.
func (o *KafkaOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	o.TLSOptions.AddFlags(fs, fullPrefix+".tls")

	fs.StringSliceVar(&o.Brokers, fullPrefix+".brokers", o.Brokers, "The list of brokers used to discover the partitions available on the kafka cluster.")
	fs.StringVar(&o.Topic, fullPrefix+".topic", o.Topic, "The topic that the writer/reader will produce/consume messages to.")
	fs.StringVar(&o.ClientID, fullPrefix+".client-id", o.ClientID, " Unique identifier for client connections established.")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout for network requests.")
	fs.StringVar(&o.SASLMechanism, fullPrefix+".mechanism", o.SASLMechanism, "Configures the Dialer to use SASL authentication.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username of the kafka cluster.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "Password of the kafka cluster.")
	fs.StringVar(&o.Algorithm, fullPrefix+".algorithm", o.Algorithm, "Algorithm used to create sasl.Mechanism (scram-sha-256 or scram-sha-512).")
	fs.BoolVar(&o.Compressed, fullPrefix+".compressed", o.Compressed, "compressed is used to specify whether compress Kafka messages (Snappy).")

	fs.IntVar(&o.WriterOptions.RequiredAcks, fullPrefix+".required-acks", o.WriterOptions.RequiredAcks, "Number of acknowledges required: -1=All, 1=Leader, 0=None.")
	fs.IntVar(&o.WriterOptions.MaxAttempts, fullPrefix+".writer.max-attempts", o.WriterOptions.MaxAttempts, "Limit on how many attempts will be made to deliver a message.")
	fs.BoolVar(&o.WriterOptions.Async, fullPrefix+".writer.async", o.WriterOptions.Async, "Whether to produce messages asynchronously.")
	fs.DurationVar(&o.WriterOptions.BatchTimeout, fullPrefix+".writer.batch-timeout", o.WriterOptions.BatchTimeout, "Time limit on how often incomplete message batches will be flushed to kafka.")
	fs.IntVar(&o.WriterOptions.BatchBytes, fullPrefix+".writer.batch-bytes", o.WriterOptions.BatchBytes, "Limit the maximum size of a request in bytes before being sent to a partition.")

	fs.StringVar(&o.ReaderOptions.GroupID, fullPrefix+".reader.group-id", o.ReaderOptions.GroupID, "GroupID holds the optional consumer group id.")
	fs.IntVar(&o.ReaderOptions.MinBytes, fullPrefix+".reader.min-bytes", o.ReaderOptions.MinBytes, "MinBytes indicates the minimum batch size.")
	fs.IntVar(&o.ReaderOptions.MaxBytes, fullPrefix+".reader.max-bytes", o.ReaderOptions.MaxBytes, "MaxBytes indicates the maximum batch size.")
	fs.DurationVar(&o.ReaderOptions.MaxWait, fullPrefix+".reader.max-wait", o.ReaderOptions.MaxWait, "Maximum amount of time to wait for new data.")
	fs.DurationVar(&o.ReaderOptions.HeartbeatInterval, fullPrefix+".reader.heartbeat-interval", o.ReaderOptions.HeartbeatInterval, "Frequency of consumer group heartbeats.")
	fs.DurationVar(&o.ReaderOptions.SessionTimeout, fullPrefix+".reader.session-timeout", o.ReaderOptions.SessionTimeout, "Coordinator session timeout.")
	fs.DurationVar(&o.ReaderOptions.RebalanceTimeout, fullPrefix+".reader.rebalance-timeout", o.ReaderOptions.RebalanceTimeout, "Time to wait for members to join rebalance.")
	fs.Int64Var(&o.ReaderOptions.StartOffset, fullPrefix+".reader.start-offset", o.ReaderOptions.StartOffset, "StartOffset: -1 for Newest, -2 for Oldest (kafka-go legacy constants).")
}

// GetMechanism returns the franz-go SASL mechanism.
func (o *KafkaOptions) GetMechanism() (sasl.Mechanism, error) {
	if o.SASLMechanism == "" {
		return nil, nil
	}

	switch strings.ToLower(o.SASLMechanism) {
	case "plain":
		// 修复 1: 使用 plain.Auth 而不是 plain.Mechanism
		return plain.Auth{
			User: o.Username,
			Pass: o.Password,
		}.AsMechanism(), nil
	case "scram":
		if strings.EqualFold(o.Algorithm, "sha-512") {
			return scram.Auth{
				User: o.Username,
				Pass: o.Password,
			}.AsSha512Mechanism(), nil
		}
		// Default to sha-256
		return scram.Auth{
			User: o.Username,
			Pass: o.Password,
		}.AsSha256Mechanism(), nil
	default:
		return nil, fmt.Errorf("unsupported mechanism: %s", o.SASLMechanism)
	}
}

// ClientOptions returns a slice of kgo.Opt to configure the franz-go client.
// This replaces the old Dialer() and Writer() methods as franz-go uses a single Client for both.
func (o *KafkaOptions) ClientOptions() ([]kgo.Opt, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(o.Brokers...),
		kgo.WithLogger(franzLogger{}), // Use our custom adapter for klog
	}

	// 1. Client Identity
	if o.ClientID != "" {
		opts = append(opts, kgo.ClientID(o.ClientID))
	}

	// 2. Network / Timeouts
	opts = append(opts, kgo.DialTimeout(o.Timeout))
	// kgo doesn't have a direct "ReadTimeout" or "WriteTimeout" per se,
	// it handles request timeouts dynamically, but we can set request timeout overhead.
	opts = append(opts, kgo.RequestTimeoutOverhead(o.Timeout))

	// 3. TLS
	tlsConfig, err := o.TLSOptions.TLSConfig()
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}

	// 4. SASL
	mechanism, err := o.GetMechanism()
	if err != nil {
		return nil, err
	}
	if mechanism != nil {
		opts = append(opts, kgo.SASL(mechanism))
	}

	// 5. Compression
	if o.Compressed {
		// Enable Snappy compression (and others if desired) for producing
		opts = append(opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	}

	// 6. Producer Options (Writer)
	// Required Acks - 修复 3: 使用新的 API
	/*
		var acks kgo.Acks
		switch o.WriterOptions.RequiredAcks {
		case 0:
			acks = kgo.NoAck()
		case 1:
			acks = kgo.LeaderAck()
			case -1:
			acks = kgo.AllISRAck()
			default:
			acks = kgo.AllISRAck() // Default to safest
		}
		opts = append(opts, kgo.ProducerAcks(acks))
	*/

	// Retries
	opts = append(opts, kgo.RecordRetries(o.WriterOptions.MaxAttempts))

	// Batching
	opts = append(opts, kgo.ProducerBatchMaxBytes(int32(o.WriterOptions.BatchBytes)))
	opts = append(opts, kgo.ProducerLinger(o.WriterOptions.BatchTimeout))

	// 7. Consumer Options (Reader)
	if o.ReaderOptions.GroupID != "" {
		opts = append(opts, kgo.ConsumerGroup(o.ReaderOptions.GroupID))

		// Heartbeats & Timeouts
		opts = append(opts, kgo.HeartbeatInterval(o.ReaderOptions.HeartbeatInterval))
		opts = append(opts, kgo.SessionTimeout(o.ReaderOptions.SessionTimeout))
		opts = append(opts, kgo.RebalanceTimeout(o.ReaderOptions.RebalanceTimeout))

		// Reset Offsets logic
		// Mapping legacy kafka-go constants: -1 (Latest), -2 (Oldest)
		// franz-go defaults to StopOnDataLoss, so we should configure a reset policy.
		if o.ReaderOptions.StartOffset == -2 { // FirstOffset
			opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
		} else {
			opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
		}
	}

	// Fetching limits
	opts = append(opts, kgo.FetchMinBytes(int32(o.ReaderOptions.MinBytes)))
	opts = append(opts, kgo.FetchMaxBytes(int32(o.ReaderOptions.MaxBytes)))
	opts = append(opts, kgo.FetchMaxWait(o.ReaderOptions.MaxWait))

	// Default Topics
	if o.Topic != "" {
		opts = append(opts, kgo.ConsumeTopics(o.Topic))
		opts = append(opts, kgo.DefaultProduceTopic(o.Topic))
	}

	return opts, nil
}

// NewClient creates a new franz-go client based on the options.
func (o *KafkaOptions) NewClient() (*kgo.Client, error) {
	opts, err := o.ClientOptions()
	if err != nil {
		return nil, err
	}
	return kgo.NewClient(opts...)
}
