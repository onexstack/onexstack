package options

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"resty.dev/v3"

	restylogger "github.com/onexstack/onexstack/pkg/logger/slog/resty"
)

// RestyOptions contains configuration related to HTTP client behavior.
type RestyOptions struct {
	// Endpoint is the base endpoint of the target API server.
	Endpoint string `json:"endpoint" mapstructure:"endpoint"`

	// UserAgent sets the User-Agent header used by the client.
	UserAgent string `json:"user-agent" mapstructure:"user-agent"`

	// Debug controls the client debug mode.
	Debug bool `json:"debug" mapstructure:"debug"`

	// Timeout controls the request timeout for the client.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// RetryCount enables retry and controls the number of retry attempts.
	RetryCount int `json:"retry-count" mapstructure:"retry-count"`

	// TraceContextProvider returns a context used for trace propagation.
	// If nil, context.Background() will be used.
	TraceContextProvider func() context.Context `json:"-"`

	fullPrefix string `json:"-" mapstructure:"-"`
	// middlewares are executed before each request is sent.
	middlewares []resty.RequestMiddleware `json:"-" mapstructure:"-"`
	// headers are default headers applied to the client.
	headers map[string]string `json:"-" mapstructure:"-"`
	// token used for Authorization header injection.
	token string `json:"-" mapstructure:"-"`
}

// NewRestyOptions creates a RestyOptions with default parameters.
func NewRestyOptions() *RestyOptions {
	return &RestyOptions{
		Endpoint:   "",
		UserAgent:  "onexstack",
		Debug:      false,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// Validate checks the RestyOptions fields for basic constraints.
func (o *RestyOptions) Validate() []error {
	if o == nil {
		return nil
	}

	var errs []error

	// Endpoint must be non-empty and a valid URL with http/https scheme.
	if o.Endpoint == "" {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+"endpoint is required"))
	} else if u, err := url.ParseRequestURI(o.Endpoint); err != nil {
		errs = append(errs, fmt.Errorf("invalid resty.endpoint %q: %v", o.Endpoint, err))
	} else if u.Scheme != "http" && u.Scheme != "https" {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+"endpoint must use http or https scheme, got %q", u.Scheme))
	}

	// Ensure non-negative RetryCount and positive Timeout.
	if o.RetryCount < 0 {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+"retry-count must be >= 0, got %d", o.RetryCount))
	}
	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+"timeout must be > 0, got %s", o.Timeout))
	}

	return errs
}

// AddFlags adds flags related to HTTP client configuration.
func (o *RestyOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Endpoint, fullPrefix+".endpoint", o.Endpoint,
		"Base URL of the target API server.")
	fs.StringVar(&o.UserAgent, fullPrefix+".user-agent", o.UserAgent,
		"Used to specify the Resty client User-Agent.")
	fs.BoolVar(&o.Debug, fullPrefix+".debug", o.Debug,
		"Enables the debug mode on Resty client (string-based).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout,
		"Request timeout for client.")
	fs.IntVar(&o.RetryCount, fullPrefix+".retry-count", o.RetryCount,
		"Enables retry on Resty client and allows you to set the number of retry attempts.")
}

// WithMiddlewares sets the request middlewares to be applied on each new request.
// It replaces any previously configured middlewares.
func (o *RestyOptions) WithMiddlewares(middlewares ...resty.RequestMiddleware) *RestyOptions {
	o.middlewares = middlewares
	return o
}

// WithHeaders sets default headers for the client.
// It merges with existing headers; later calls override keys from earlier ones.
func (o *RestyOptions) WithHeaders(headers map[string]string) *RestyOptions {
	if o.headers == nil {
		o.headers = make(map[string]string)
	}
	for k, v := range headers {
		o.headers[k] = v
	}
	return o
}

// WithToken sets the token used to inject Authorization header by middleware.
func (o *RestyOptions) WithToken(token string) *RestyOptions {
	o.token = token
	return o
}

// WithTrace configures a trace middleware that injects distributed tracing headers
// into each outgoing request using the global otel propagator.
func (o *RestyOptions) WithTrace() *RestyOptions {
	if o.TraceContextProvider == nil {
		o.TraceContextProvider = func() context.Context { return context.Background() }
	}

	mw := func(c *resty.Client, r *resty.Request) error {
		ctx := r.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))
		return nil
	}

	o.middlewares = append(o.middlewares, mw)
	return o
}

// WithTokenMiddleware adds a middleware that injects the Authorization header
// on each request based on o.Token.
func (o *RestyOptions) addTokenMiddleware() *RestyOptions {
	mw := func(c *resty.Client, r *resty.Request) error {
		if o.token == "" {
			return nil
		}
		r.SetHeader("Authorization", "Bearer "+o.token)
		return nil
	}

	o.middlewares = append(o.middlewares, mw)
	return o
}

// applyToClient applies RestyOptions to the given resty.Client.
func (o *RestyOptions) applyToClient(client *resty.Client) {
	client.SetBaseURL(o.Endpoint).
		SetTimeout(o.Timeout).
		SetRetryCount(o.RetryCount).
		SetDebug(o.Debug).
		SetLogger(restylogger.NewLogger()).
		// Default User-Agent
		SetHeader("User-Agent", o.UserAgent)

	// Apply custom headers (overrides existing ones with same key).
	if len(o.headers) > 0 {
		client.SetHeaders(o.headers)
	}

	if o.token != "" {
		o.addTokenMiddleware()
	}

	// Apply middlewares (before request hook).
	for _, mw := range o.middlewares {
		client.AddRequestMiddleware(mw)
	}
}

// NewClient creates a new resty.Client configured from RestyOptions.
func (o *RestyOptions) NewClient() *resty.Client {
	client := resty.New()
	o.applyToClient(client)
	return client
}

// NewRequest creates a new resty.Request configured from RestyOptions.
func (o *RestyOptions) NewRequest() *resty.Request {
	return o.NewClient().R()
}
