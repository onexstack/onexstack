package options

import (
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
)

// Ensure ObservabilityOptions implements the IOptions interface.
var _ IOptions = (*ObservabilityOptions)(nil)

// ObservabilityOptions defines configuration for the auxiliary observability server.
// This server typically handles health checks, profiling (pprof), and metrics (Prometheus).
type ObservabilityOptions struct {
	// BindAddress is the IP address and port to bind the server to.
	BindAddress string `json:"bind-address" mapstructure:"bind-address"`
	// EnablePprof toggles the exposure of runtime profiling data via HTTP.
	EnablePprof bool `json:"enable-pprof" mapstructure:"enable-pprof"`
}

// NewObservabilityOptions creates a new instance with default values.
func NewObservabilityOptions() *ObservabilityOptions {
	return &ObservabilityOptions{
		BindAddress: "0.0.0.0:20250",
		EnablePprof: false,
	}
}

// Validate verifies the configuration flags.
func (o *ObservabilityOptions) Validate() []error {
	return []error{}
}

// AddFlags registers flags related to observability to the provided FlagSet.
func (o *ObservabilityOptions) AddFlags(fs *pflag.FlagSet, prefix string) {
	fs.StringVar(&o.BindAddress, prefix+".bind-address", o.BindAddress, "Specifies the bind address for health checks and metrics.")
	fs.BoolVar(&o.EnablePprof, prefix+".enable-pprof", o.EnablePprof, "Expose runtime profiling data via HTTP.")
}

// Serve starts the observability HTTP server with default handlers.
// Default handlers: "/healthz" (Health Check) and "/metrics" (Prometheus).
func (o *ObservabilityOptions) Serve() error {
	// Pass nil to trigger default behavior in startServer (or just pass empty map).
	// But logically, Serve() just means "Serve with defaults".
	return o.ServeWithHandlers(nil)
}

// ServeWithHandlers starts the server with default handlers PLUS extra handlers.
// The default handlers ("/healthz", "/metrics") are always added.
// If extraHandlers contains these keys, they will overwrite the defaults.
func (o *ObservabilityOptions) ServeWithHandlers(extraHandlers map[string]http.Handler) error {
	// 1. Start with default handlers
	finalHandlers := map[string]http.Handler{
		"/healthz": o.DefaultHealthzHandler(),
		"/metrics": o.DefaultMetricsHandler(),
	}

	// 2. Merge extra handlers (this allows overriding defaults if needed)
	for path, handler := range extraHandlers {
		finalHandlers[path] = handler
	}

	// 3. Delegate to internal server starter
	return o.startServer(finalHandlers)
}

// startServer contains the core logic to configure and run the HTTP server.
// It avoids code duplication between Serve and ServeWithHandlers.
func (o *ObservabilityOptions) startServer(handlers map[string]http.Handler) error {
	mux := http.NewServeMux()

	// Register all handlers
	// We wrap each handler to strictly enforce HTTP GET method.
	for path, handler := range handlers {
		wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			handler.ServeHTTP(w, r)
		})

		mux.Handle(path, wrappedHandler)
		slog.Info("registered observability handler", "path", path, "method", http.MethodGet)
	}

	// Register pprof handlers if enabled
	if o.EnablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		slog.Info("pprof profiling enabled", "path_prefix", "/debug/pprof/")
	}

	// Register a catch-all handler for 404 Not Found
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code": "NotFound.PageNotFound", "message": "The requested page was not found."}`))
	})

	// Create a custom server to ensure timeouts are set
	server := &http.Server{
		Addr:              o.BindAddress,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	slog.Info("starting observability server", "addr", o.BindAddress)

	return server.ListenAndServe()
}

// DefaultMetricsHandler returns a configured Prometheus handler.
func (o *ObservabilityOptions) DefaultMetricsHandler() http.Handler {
	return promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

// DefaultHealthzHandler returns a simple liveness check handler.
func (o *ObservabilityOptions) DefaultHealthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Ignore write errors as health checks are transient
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})
}
