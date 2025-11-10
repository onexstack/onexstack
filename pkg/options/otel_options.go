package options

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/onexstack/onexstack/pkg/otelslog"
)

// Ensure interface
var _ IOptions = (*OTelOptions)(nil)

// OutputMode represents the output mode for OpenTelemetry data
type OutputMode string

const (
	OutputModeOTLP    OutputMode = "otel"    // Send to OpenTelemetry Collector
	OutputModeFile    OutputMode = "file"    // Output to files
	OutputModeConsole OutputMode = "console" // Output to standard output
)

// String implements the Stringer interface
func (o OutputMode) String() string {
	return string(o)
}

// IsValid validates the output mode
func (o OutputMode) IsValid() bool {
	switch o {
	case OutputModeConsole, OutputModeFile, OutputModeOTLP:
		return true
	default:
		return false
	}
}

// outputModeFlag implements pflag.Value interface for OutputMode
type outputModeFlag OutputMode

func (f *outputModeFlag) String() string { return string(*f) }
func (f *outputModeFlag) Type() string   { return "string" }
func (f *outputModeFlag) Set(s string) error {
	mode := OutputMode(s)
	if !mode.IsValid() {
		return fmt.Errorf("invalid output mode: %s, valid options: otlp, console, file, slog", s)
	}
	*f = outputModeFlag(mode)
	return nil
}

// Provider wraps OpenTelemetry providers with shutdown capability
type Provider interface {
	Shutdown(context.Context) error
}

// OTelProviders holds all OpenTelemetry providers
type OTelProviders struct {
	tracer *trace.TracerProvider
	meter  *metric.MeterProvider
	logger *log.LoggerProvider
}

// OTelOptions OpenTelemetry configuration
type OTelOptions struct {
	// Connection settings
	Endpoint string `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Insecure bool   `json:"insecure,omitempty" mapstructure:"insecure"`

	// Service identification
	ServiceName       string `json:"service-name,omitempty" mapstructure:"service-name"`
	ServiceVersion    string `json:"service-version,omitempty" mapstructure:"service-version"`
	ServiceInstanceID string `json:"service-instance-id,omitempty" mapstructure:"service-instance-id"`
	Environment       string `json:"environment,omitempty" mapstructure:"environment"`

	// Behavior settings
	SamplingRatio float64 `json:"sampling-ratio,omitempty" mapstructure:"sampling-ratio"`
	WithResource  bool    `json:"with-resource,omitempty" mapstructure:"with-resource"`

	// Output configuration
	OutputMode OutputMode `json:"output-mode,omitempty" mapstructure:"output-mode"`
	OutputDir  string     `json:"output-dir,omitempty" mapstructure:"output-dir"`

	// Prometheus integration
	UsePrometheusEndpoint bool `json:"use-prometheus-endpoint,omitempty" mapstructure:"use-prometheus-endpoint"`
	UserNativeSlog        bool `json:"user-native-slog,omitempty" mapstructure:"user-native-slog"`

	// Logging configuration
	Level     string `json:"level,omitempty" mapstructure:"level"`
	AddSource bool   `json:"add-source,omitempty" mapstructure:"add-source"`

	Slog *SlogOptions `json:"slog,omitempty" mapstructure:"slog"`

	// Internal state
	mu        sync.RWMutex
	providers *OTelProviders
	files     []io.Closer
	resource  *resource.Resource
}

var (
	resourceOnce sync.Once
	otelResource *resource.Resource
)

// NewOTelOptions creates a new OTelOptions with sensible defaults
func NewOTelOptions() *OTelOptions {
	hostname, _ := os.Hostname()
	opts := &OTelOptions{
		ServiceName:           "unknown-service",
		ServiceVersion:        "1.0.0",
		ServiceInstanceID:     hostname,
		Environment:           "development",
		Endpoint:              "localhost:4317",
		Insecure:              true,
		WithResource:          false,
		SamplingRatio:         1.0,
		OutputMode:            OutputModeConsole,
		OutputDir:             "./otel-output",
		Level:                 "info",
		AddSource:             false,
		UsePrometheusEndpoint: false,
		UserNativeSlog:        false,
		Slog:                  NewSlogOptions(),
		providers:             &OTelProviders{},
		files:                 make([]io.Closer, 0),
	}

	// Sync slog options
	opts.syncSlogOptions()
	return opts
}

// syncSlogOptions synchronizes slog options with main options
func (o *OTelOptions) syncSlogOptions() {
	if o.Slog != nil {
		o.Slog.Level = o.Level
		o.Slog.AddSource = o.AddSource
	}
}

// Validate validates the configuration
func (o *OTelOptions) Validate() []error {
	var errs []error

	// Sync slog options before validation
	o.syncSlogOptions()

	// Validate slog options
	if o.Slog != nil {
		errs = append(errs, o.Slog.Validate()...)
	}

	// Validate output mode
	if !o.OutputMode.IsValid() {
		errs = append(errs, fmt.Errorf("invalid output mode: %s", o.OutputMode))
	}

	// Validate required fields
	if o.ServiceName == "" {
		errs = append(errs, fmt.Errorf("service name is required"))
	}
	if o.ServiceInstanceID == "" {
		errs = append(errs, fmt.Errorf("service instance ID is required"))
	}

	// Validate OTLP specific settings
	if o.OutputMode == OutputModeOTLP && o.Endpoint == "" {
		errs = append(errs, fmt.Errorf("endpoint is required for OTLP output mode"))
	}

	// Validate sampling ratio
	if o.SamplingRatio < 0 || o.SamplingRatio > 1 {
		errs = append(errs, fmt.Errorf("sampling ratio must be between 0 and 1, got: %f", o.SamplingRatio))
	}

	// Validate output directory for file mode
	if o.OutputMode == OutputModeFile && o.OutputDir == "" {
		errs = append(errs, fmt.Errorf("output directory is required for file output mode"))
	}

	return errs
}

// AddFlags adds command line flags
func (o *OTelOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	fs.StringVar(&o.ServiceName, join(prefixes...)+"otel.service-name", o.ServiceName, "Service name")
	fs.StringVar(&o.ServiceVersion, join(prefixes...)+"otel.service-version", o.ServiceVersion, "Service version")
	fs.StringVar(&o.ServiceInstanceID, join(prefixes...)+"otel.service-instance-id", o.ServiceInstanceID, "Service instance ID (auto-generated if empty)")
	fs.StringVar(&o.Environment, join(prefixes...)+"otel.environment", o.Environment, "Environment")
	fs.StringVar(&o.Endpoint, join(prefixes...)+"otel.endpoint", o.Endpoint, "OTLP endpoint")
	fs.BoolVar(&o.Insecure, join(prefixes...)+"otel.insecure", o.Insecure, "Use insecure connection")
	fs.Float64Var(&o.SamplingRatio, join(prefixes...)+"otel.sampling-ratio", o.SamplingRatio, "Sampling ratio (0.0-1.0)")
	fs.BoolVar(&o.WithResource, join(prefixes...)+"otel.with-resource", o.WithResource, "Include system resource information")
	fs.Var((*outputModeFlag)(&o.OutputMode), join(prefixes...)+"otel.output-mode", "Output mode: otlp, console, file, slog")
	fs.StringVar(&o.OutputDir, join(prefixes...)+"otel.output-dir", o.OutputDir, "Output directory for file mode")
	fs.StringVar(&o.Level, join(prefixes...)+"otel.level", o.Level, "Log level: debug, info, warn, error")
	fs.BoolVar(&o.AddSource, join(prefixes...)+"otel.add-source", o.AddSource, "Add source code position to logs")
	fs.BoolVar(
		&o.UsePrometheusEndpoint,
		join(prefixes...)+"otel.use-prometheus-endpoint",
		o.UsePrometheusEndpoint,
		"Enable Prometheus metrics endpoint (/metrics)",
	)
	fs.BoolVar(&o.UserNativeSlog, join(prefixes...)+"otel.user-native-slog", o.UserNativeSlog, "Use native slog library (bypass OpenTelemetry log pipeline)")

	if o.Slog != nil {
		prefixes = append(prefixes, "otel")
		o.Slog.AddFlags(fs, prefixes...)
	}
}

// GetResource creates or returns cached resource configuration
func (o *OTelOptions) GetResource() *resource.Resource {
	resourceOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Base resource attributes
		attrs := []resource.Option{
			resource.WithAttributes(
				semconv.ServiceName(o.ServiceName),
				semconv.ServiceVersion(o.ServiceVersion),
				semconv.ServiceInstanceID(o.ServiceInstanceID),
				semconv.DeploymentEnvironment(o.Environment),
			),
		}

		// Add system resource information if enabled
		if o.WithResource {
			attrs = append(attrs,
				resource.WithOS(),
				resource.WithProcess(),
				resource.WithContainer(),
				resource.WithHost(),
			)
		}

		var err error
		otelResource, err = resource.New(ctx, attrs...)
		if err != nil {
			// Fallback to default resource
			otelResource = resource.Default()
		}
	})
	return otelResource
}

// createFileWriter creates and manages file writers
func (o *OTelOptions) createFileWriter(name string) (io.Writer, error) {
	if err := os.MkdirAll(o.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := filepath.Join(o.OutputDir, name+".json")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", filename, err)
	}

	o.files = append(o.files, file)
	return file, nil
}

// initTraces initializes tracing
func (o *OTelOptions) initTraces(ctx context.Context) error {
	var (
		exporter trace.SpanExporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeOTLP:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("traces")
		if writerErr != nil {
			return fmt.Errorf("failed to create trace writer: %w", writerErr)
		}
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(writer))
	default: // OutputModeConsole and fallback
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(o.GetResource()),
		trace.WithSampler(trace.TraceIDRatioBased(o.SamplingRatio)),
	)

	o.providers.tracer = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// initMetrics initializes metrics
func (o *OTelOptions) initMetrics(ctx context.Context) error {
	var (
		reader metric.Reader
		err    error
	)

	if o.UsePrometheusEndpoint {
		reader, err = prometheus.New()
	} else {
		exporter, exporterErr := o.createMetricsExporter(ctx)
		if exporterErr != nil {
			return fmt.Errorf("failed to create metrics exporter: %w", exporterErr)
		}
		reader = metric.NewPeriodicReader(exporter)
	}

	if err != nil {
		return fmt.Errorf("failed to create metrics reader: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(o.GetResource()),
	)

	o.providers.meter = mp
	otel.SetMeterProvider(mp)
	return nil
}

// createMetricsExporter creates metrics exporter based on output mode
func (o *OTelOptions) createMetricsExporter(ctx context.Context) (metric.Exporter, error) {
	switch o.OutputMode {
	case OutputModeOTLP:
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(o.Endpoint)}
		if o.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		return otlpmetricgrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, err := o.createFileWriter("metrics")
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics writer: %w", err)
		}
		return stdoutmetric.New(stdoutmetric.WithWriter(writer))
	default: // OutputModeConsole and fallback
		return stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
	}
}

// initLogs initializes logging
func (o *OTelOptions) initLogs(ctx context.Context) error {
	if o.UserNativeSlog {
		return o.Slog.Apply()
	}

	var (
		exporter log.Exporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeOTLP:
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		exporter, err = otlploggrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("logs")
		if writerErr != nil {
			return fmt.Errorf("failed to create logs writer: %w", writerErr)
		}
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(writer))
	default: // OutputModeConsole and fallback
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create log exporter: %w", err)
	}

	var processor log.Processor
	if o.OutputMode == OutputModeConsole {
		processor = log.NewSimpleProcessor(exporter)
	} else {
		processor = log.NewBatchProcessor(exporter)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(o.GetResource()),
	)

	o.providers.logger = lp
	global.SetLoggerProvider(lp)

	// Set up slog integration
	logger := otelslog.NewLogger(
		o.ServiceName,
		otelslog.WithLoggerProvider(global.GetLoggerProvider()),
		otelslog.WithSource(o.AddSource),
		otelslog.WithLevelString(o.Level),
	)
	slog.SetDefault(logger)

	return nil
}

// Apply applies the configuration
func (o *OTelOptions) Apply() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Sync slog options
	o.syncSlogOptions()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize all providers
	if err := o.initTraces(ctx); err != nil {
		return fmt.Errorf("failed to initialize traces: %w", err)
	}

	if err := o.initMetrics(ctx); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	if err := o.initLogs(ctx); err != nil {
		return fmt.Errorf("failed to initialize logs: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down all providers
func (o *OTelOptions) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var errs []error

	// Shutdown providers
	if o.providers.tracer != nil {
		if err := o.providers.tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if o.providers.meter != nil {
		if err := o.providers.meter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}

	if o.providers.logger != nil {
		if err := o.providers.logger.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
		}
	}

	// Close files
	for _, file := range o.files {
		if err := file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("file close: %w", err))
		}
	}
	o.files = o.files[:0] // Clear slice

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// GetTracerProvider returns the tracer provider
func (o *OTelOptions) GetTracerProvider() *trace.TracerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.tracer
}

// GetMeterProvider returns the meter provider
func (o *OTelOptions) GetMeterProvider() *metric.MeterProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.meter
}

// GetLoggerProvider returns the logger provider
func (o *OTelOptions) GetLoggerProvider() *log.LoggerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.logger
}
