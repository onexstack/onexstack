package options

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5" // 引入 JWT 库
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

	// SecretID is the secret identifier used for authentication.
	SecretID string `json:"secret-id" mapstructure:"secret-id"`

	// SecretKey is the secret key used for authentication.
	SecretKey string `json:"secret-key" mapstructure:"secret-key"`

	// Username for basic authentication.
	Username string `json:"username" mapstructure:"username"`

	// Password for basic authentication.
	Password string `json:"password" mapstructure:"password"`

	// Token for bearer token authentication.
	Token string `json:"token" mapstructure:"token"`

	// TraceContextProvider returns a context used for trace propagation.
	// If nil, context.Background() will be used.
	TraceContextProvider func() context.Context `json:"-"`

	fullPrefix string `json:"-" mapstructure:"-"`
	// middlewares are executed before each request is sent.
	middlewares []resty.RequestMiddleware `json:"-" mapstructure:"-"`
	// headers are default headers applied to the client.
	headers map[string]string `json:"-" mapstructure:"-"`
}

// NewRestyOptions creates a RestyOptions with default parameters.
func NewRestyOptions() *RestyOptions {
	return &RestyOptions{
		Endpoint:   "",
		UserAgent:  "onexstack",
		Debug:      false,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		SecretID:   "",
		SecretKey:  "",
		Username:   "",
		Password:   "",
		Token:      "",
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
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+".endpoint is required"))
	} else if u, err := url.ParseRequestURI(o.Endpoint); err != nil {
		errs = append(errs, fmt.Errorf("invalid endpoint %q: %v", o.Endpoint, err))
	} else if u.Scheme != "http" && u.Scheme != "https" {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+".endpoint must use http or https scheme, got %q", u.Scheme))
	}

	// Ensure non-negative RetryCount and positive Timeout.
	if o.RetryCount < 0 {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+".retry-count must be >= 0, got %d", o.RetryCount))
	}
	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--"+o.fullPrefix+".timeout must be > 0, got %s", o.Timeout))
	}

	// Validate authentication configurations
	if (o.SecretID != "" && o.SecretKey == "") || (o.SecretID == "" && o.SecretKey != "") {
		errs = append(errs, fmt.Errorf("both --"+o.fullPrefix+".secret-id and --"+o.fullPrefix+".secret-key must be provided together"))
	}

	if (o.Username != "" && o.Password == "") || (o.Username == "" && o.Password != "") {
		errs = append(errs, fmt.Errorf("both --"+o.fullPrefix+".username and --"+o.fullPrefix+".password must be provided together"))
	}

	return errs
}

// AddFlags adds flags related to HTTP client configuration.
func (o *RestyOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	o.fullPrefix = fullPrefix

	fs.StringVar(&o.Endpoint, fullPrefix+".endpoint", o.Endpoint, "Base URL of the target API server.")
	fs.StringVar(&o.UserAgent, fullPrefix+".user-agent", o.UserAgent, "Used to specify the Resty client User-Agent.")
	fs.BoolVar(&o.Debug, fullPrefix+".debug", o.Debug, "Enables the debug mode on Resty client (string-based).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Request timeout for client.")
	fs.IntVar(&o.RetryCount, fullPrefix+".retry-count", o.RetryCount,
		"Enables retry on Resty client and allows you to set the number of retry attempts.")
	fs.StringVar(&o.SecretID, fullPrefix+".secret-id", o.SecretID, "Secret identifier used for authentication.")
	fs.StringVar(&o.SecretKey, fullPrefix+".secret-key", o.SecretKey, "Secret key used for authentication.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username for basic authentication.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "Password for basic authentication.")
	fs.StringVar(&o.Token, fullPrefix+".token", o.Token, "Bearer token for authentication.")
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
	o.Token = token
	return o
}

// WithSecretCredentials sets the secret ID and key for authentication.
func (o *RestyOptions) WithSecretCredentials(secretID, secretKey string) *RestyOptions {
	o.SecretID = secretID
	o.SecretKey = secretKey
	return o
}

// WithBasicAuth sets the username and password for basic authentication.
func (o *RestyOptions) WithBasicAuth(username, password string) *RestyOptions {
	o.Username = username
	o.Password = password
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

// generateJWT creates a signed JWT using SecretID as kid (and optionally iss/sub).
func (o *RestyOptions) generateJWT() (string, error) {
	claims := jwt.MapClaims{
		"nbf": time.Now().Unix(),                    // token effective time
		"exp": time.Now().Add(2 * time.Hour).Unix(), // token expiration time
		"iat": time.Now().Unix(),                    // token issued at time
	}

	// 1. Create Token object
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 2. [Key modification] Set kid in Header
	// token.Header is a map[string]interface{}
	token.Header["kid"] = o.SecretID

	// 3. Sign and generate string
	return token.SignedString([]byte(o.SecretKey))
}

// addAuthMiddleware handles the authentication logic with priority:
// 1. Dynamic JWT (if SecretID & SecretKey are present)
// 2. Static Token (if Token is present)
func (o *RestyOptions) addAuthMiddleware() *RestyOptions {
	mw := func(c *resty.Client, r *resty.Request) error {
		var finalToken string

		// Priority 1: Generate JWT from Secret Credentials
		if o.SecretID != "" && o.SecretKey != "" {
			var err error
			finalToken, err = o.generateJWT()
			if err != nil {
				// Decide whether to block the request or fallback.
				// Here we return error because bad config implies failed auth.
				return fmt.Errorf("failed to generate jwt token: %w", err)
			}
			// Optional: Inject SecretID for tracing or auditing
			r.SetHeader("X-Secret-ID", o.SecretID)
		}

		// Priority 2: Use Static Token if JWT generation was skipped (no secrets)
		if finalToken == "" && o.Token != "" {
			finalToken = o.Token
		}

		// If we have a valid token (either generated or static), set the Authorization header
		if finalToken != "" {
			r.SetHeader("Authorization", "Bearer "+finalToken)
		}

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

	// Apply basic authentication if provided
	if o.Username != "" && o.Password != "" {
		client.SetBasicAuth(o.Username, o.Password)
	}

	// Apply combined authentication middleware (Secret JWT > Static Token)
	o.addAuthMiddleware()

	// Apply other middlewares (before request hook).
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
