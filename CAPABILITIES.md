# CAPABILITIES.md

> Auto-generated catalog of all reusable packages in `github.com/onexstack/onexstack`.
> **Purpose:** Help AI coding assistants discover and correctly reuse existing capabilities.

---

## Table of Contents

1. [Application Framework](#1-application-framework)
2. [Authentication & Authorization](#2-authentication--authorization)
3. [Data & Storage](#3-data--storage)
4. [Request Handling & Core Utilities](#4-request-handling--core-utilities)
5. [Logging](#5-logging)
6. [HTTP/gRPC Middleware](#6-httpgrpc-middleware)
7. [Observability (OpenTelemetry)](#7-observability-opentelemetry)
8. [CLI Utilities](#8-cli-utilities)
9. [Utility Packages](#9-utility-packages)

---

## 1. Application Framework

### 1.1 `pkg/app` — Application Bootstrap Framework

**Import:** `github.com/onexstack/onexstack/pkg/app`

Builds a Cobra-based CLI application with config management, health checks, and structured logging.

| API | Signature | Description |
|-----|-----------|-------------|
| `NewApp` | `func NewApp(name, shortDesc string, opts ...Option) *App` | Create a new application instance |
| `(*App).Run` | `func (app *App) Run()` | Start the application |
| `(*App).Command` | `func (app *App) Command() *cobra.Command` | Get the underlying Cobra command |
| `WithOptions` | `func WithOptions(opts any) Option` | Bind options that implement `NamedFlagSetOptions` or `FlagSetOptions` |
| `WithRunFunc` | `func WithRunFunc(run RunFunc) Option` | Set the main run function |
| `WithDescription` | `func WithDescription(desc string) Option` | Set app description |
| `WithHealthCheckFunc` | `func WithHealthCheckFunc(fn HealthCheckFunc) Option` | Custom health check |
| `WithDefaultHealthCheckFunc` | `func WithDefaultHealthCheckFunc() Option` | Default health check (pings dependencies) |
| `WithSilence` | `func WithSilence() Option` | Suppress usage on error |
| `WithNoConfig` | `func WithNoConfig() Option` | Skip config file loading |
| `WithValidArgs` | `func WithValidArgs(args cobra.PositionalArgs) Option` | Custom positional arg validation |
| `WithDefaultValidArgs` | `func WithDefaultValidArgs() Option` | No-args default validation |
| `WithWatchConfig` | `func WithWatchConfig() Option` | Live-reload config on file changes |
| `WithLoggerContextExtractor` | `func WithLoggerContextExtractor(extractors map[string]func(context.Context) string) Option` | Extract fields from context for logging |
| `RunFunc` | `type RunFunc func() error` | Signature for the main run function |
| `HealthCheckFunc` | `type HealthCheckFunc func() error` | Signature for health check function |
| `OptionsValidator` | `interface { Complete() error; Validate() error }` | Interface for options validation |
| `NamedFlagSetOptions` | `interface { Flags() cliflag.NamedFlagSets; OptionsValidator }` | Options with named flag sets |
| `FlagSetOptions` | `interface { AddFlags(*pflag.FlagSet); OptionsValidator }` | Options with flat flag set |
| `PrintConfig` | `func PrintConfig()` | Print current configuration |

### 1.2 `pkg/server` — Server Implementations

**Import:** `github.com/onexstack/onexstack/pkg/server`

| API | Signature | Description |
|-----|-----------|-------------|
| `Server` | `interface { RunOrDie(ctx context.Context); GracefulStop(ctx context.Context) }` | Common server interface |
| `NewGRPCServer` | `func NewGRPCServer(grpcOptions, tlsOptions, serverOptions, registerBuilder) (*GRPCServer, error)` | Create a gRPC server |
| `NewHTTPServer` | `func NewHTTPServer(insecureOpts, secureOpts, handler) *HTTPServer` | Create an HTTP server |
| `NewKratosServer` | `func NewKratosServer(cfg KratosAppConfig, servers ...transport.Server) (*KratosServer, error)` | Create a Kratos microservice app |
| `NewPolarisServer` | `func NewPolarisServer(polarisOpts, grpcOpts, tlsOpts, serverOpts, registerBuilder) (*PolarisServer, error)` | Create a Polaris-registered gRPC server |
| `NewGRPCGatewayServer` | `func NewGRPCGatewayServer(insecureOpts, grpcOpts, secureOpts, registerHandler) (*GRPCGatewayServer, error)` | gRPC-Gateway reverse proxy server |
| `Serve` | `func Serve(ctx context.Context, srv Server) error` | Serve with graceful shutdown on SIGINT/SIGTERM |
| `KratosAppConfig` | `struct{ ID, Name, Version string; Metadata map[string]string; Registrar registry.Registrar }` | Kratos app configuration |
| `NewEtcdRegistrar` | `func NewEtcdRegistrar(opts *genericoptions.EtcdOptions) registry.Registrar` | Create Etcd service registrar |
| `NewConsulRegistrar` | `func NewConsulRegistrar(opts *genericoptions.ConsulOptions) registry.Registrar` | Create Consul service registrar |
| `NewKratosLogger` | `func NewKratosLogger(id, name, version string) krtlog.Logger` | Kratos-compatible logger |

### 1.3 `pkg/options` — Infrastructure Configuration Options

**Import:** `github.com/onexstack/onexstack/pkg/options`

Standard options structs with `Validate()` and `AddFlags()` methods for consistent CLI flag binding.

| Struct | Key Fields | Description |
|--------|-----------|-------------|
| `MySQLOptions` | Addr, Username, Password, Database, MaxIdleConnections, MaxOpenConnections, MaxConnectionLifeTime, LogLevel | MySQL connection options; has `NewDB()` method |
| `PostgreSQLOptions` | Addr, Username, Password, Database, MaxIdleConnections, MaxOpenConnections, MaxConnectionLifeTime, LogLevel | PostgreSQL connection options; has `NewDB()` method |
| `SQLiteOptions` | Addr, Username, Password, Database, MaxIdleConnections, MaxOpenConnections, MaxConnectionLifeTime, LogLevel | SQLite connection options; has `NewDB()` method |
| `RedisOptions` | Addr, Username, Password, Database, MaxRetries, MinIdleConns, DialTimeout, ReadTimeout, WriteTimeout, PoolTimeout, PoolSize, EnableTrace | Redis options; has `NewClient()` method |
| `MongoOptions` | URL, Database, Collection, Username, Password, Timeout, TLSOptions | MongoDB options; has `NewClient()` method |
| `GRPCOptions` | Network, Addr, Timeout | gRPC server listening options |
| `InsecureServingOptions` | Addr, Timeout | HTTP (non-TLS) serving options |
| `SecureServingOptions` | Addr, Timeout, TLSOptions | HTTPS serving options |
| `TLSOptions` | Enabled, SkipVerify, CA, Cert, Key | TLS configuration; has `TLSConfig()`, `MustTLSConfig()`, `Scheme()` |
| `JWTOptions` | Key, Expired, MaxRefresh, SigningMethod | JWT signing configuration |
| `EtcdOptions` | Endpoints, DialTimeout, Username, Password, TLSOptions | Etcd connection options |
| `ConsulOptions` | Addr, Scheme | Consul connection options |
| `KafkaOptions` | Brokers, Topic, ClientID, Timeout, TLSOptions, SASLMechanism, Username, Password, Compressed, WriterOptions, ReaderOptions | Kafka producer/consumer options; has `NewClient()` |
| `PolarisOptions` | Addr, Timeout, RetryCount, Provider | Polaris service mesh options; has `Register()`, `Deregister()`, `NewProviderAPI()`, `NewConsumerAPI()` |
| `RestyOptions` | Endpoint, UserAgent, Debug, Timeout, RetryCount, SecretID, SecretKey, Username, Password, Token, TLSOptions, TraceContextProvider | REST client options; has `NewClient()`, `NewRequest()` |
| `HealthOptions` | HTTPProfile, HealthCheckPath, HealthCheckAddress | Health check endpoint; has `Serve()` |
| `ObservabilityOptions` | BindAddress, EnablePprof | Metrics/healthz/pprof serving; has `Serve()`, `ServeWithHandlers()` |
| `LogOptions` | Format, FlushFrequency, Verbosity | Kubernetes klog options; has `Native()` |
| `FluxOptions` | MaxKeys, TTL, BufferItems, LoadTimeout, EnableAsync, RefreshInterval, RefreshMode, MaxConcurrency, BatchSize, EnableStats | Cache options |
| `OTelOptions` | Endpoint, Insecure, ServiceName, ServiceVersion, ServiceInstanceID, Environment, SamplingRatio, WithResource, OutputMode, OutputDir, Level, AddSource, Slog | OpenTelemetry configuration; has `Apply()`, `Shutdown()` |
| `SlogOptions` | Level, AddSource, Format, TimeFormat, Output | Structured logging options; has `BuildLogger()`, `BuildHandler()`, `Apply()` |
| `WatchOptions` | LockName, HealthzPort, DisableWatchers, MaxWorkers, WatchTimeout, PerConcurrency | Config watch/cron options |
| `AWSSecretManagerOptions` | Region, SecretName | AWS Secrets Manager options |
| `ClientCertAuthenticationOptions` | ClientCA | Client certificate authentication |
| `JaegerOptions` | Server, ServiceName, Env | Jaeger tracing (legacy) |
| `MetricsOptions` | ShowHiddenMetricsForVersion, DisabledMetrics, AllowListMapping | Metrics configuration |

**Helper functions:** `Join(prefixes ...string) string` — join prefix parts into a config key, `ValidateAddress(addr string) error`, `CreateListener(addr string) (net.Listener, int, error)`.

### 1.4 `pkg/watch` — Configuration Watcher and Cron Job Manager

**Import:** `github.com/onexstack/onexstack/pkg/watch`

A generic cron-based watcher framework for periodic tasks.

| API | Signature | Description |
|-----|-----------|-------------|
| `NewWatch` | `func NewWatch(opts *Options, db *gorm.DB, withOptions ...Option) (*Watch, error)` | Create a watcher |
| `(*Watch).Start` | `func (w *Watch) Start(stopCh <-chan struct{})` | Start watching |
| `(*Watch).Stop` | `func (w *Watch) Stop()` | Stop watching |
| `Options` | `struct{ LockName, HealthzAddr, MetricsAddr string; DisableWatchers []string; MaxWorkers int64; WatchTimeout, PerConcurrency }` | Watcher configuration |
| `Watcher` | Embeddable struct with `WatchTimeout`, `PerConcurrency`, `Limiter` | Base watcher type |
| `Logger` | `interface { cron.Logger; Debug(...); }` | Watcher logger interface |

**Sub-packages:**

| Sub-package | Import | Key APIs |
|-------------|--------|----------|
| `watch/registry` | `.../pkg/watch/registry` | `Register(name string, watcher Watcher)` — register watchers; `ListWatchers() map[string]Watcher`; `Watcher` interface (`cron.Job`); `ISpec` interface (`Spec() string`) |
| `watch/manager` | `.../pkg/watch/manager` | `NewJobManager(opts ...Option) *JobManager` — manages cron jobs; `Add/Del/Update/GetJobs/Exists/Start/Stop` |
| `watch/initializer` | `.../pkg/watch/initializer` | `WatcherInitializer` interface; `WantsJobManager`, `WantsRateLimit`, `WantsWatchTimeout`, `WantsPerConcurrency` interfaces |
| `watch/logger/empty` | `.../pkg/watch/logger/empty` | `NewLogger()` — silent watcher logger |
| `watch/logger/onex` | `.../pkg/watch/logger/onex` | `NewLogger()` — onex-flavored watcher logger |
| `watch/logger/slog` | `.../pkg/watch/logger/slog` | `NewLogger()` — slog-backed watcher logger |

---

## 2. Authentication & Authorization

### 2.1 `pkg/authn` — Authentication Interface

**Import:** `github.com/onexstack/onexstack/pkg/authn`

| API | Signature | Description |
|-----|-----------|-------------|
| `Authenticator` | `interface { Sign(ctx, userID) (IToken, error); Destroy(ctx, accessToken) error; ParseClaims(ctx, accessToken) (*jwt.RegisteredClaims, error); Release() error }` | Core authentication interface |
| `IToken` | `interface { GetToken() string; GetTokenType() string; GetExpiresAt() int64; EncodeToJSON() ([]byte, error) }` | Token interface |
| `Encrypt` | `func Encrypt(source string) (string, error)` | Encrypt a string |
| `Compare` | `func Compare(hashedPassword, password string) error` | Compare hashed password |

### 2.2 `pkg/authn/jwt` — JWT Authentication Implementation

**Import:** `github.com/onexstack/onexstack/pkg/authn/jwt`

| API | Signature | Description |
|-----|-----------|-------------|
| `New` | `func New(store Storer, opts ...Option) *JWTAuth` | Create JWT authenticator |
| `(*JWTAuth).Sign` | `func (a *JWTAuth) Sign(ctx, userID) (authn.IToken, error)` | Sign and return a JWT token |
| `(*JWTAuth).Destroy` | `func (a *JWTAuth) Destroy(ctx, refreshToken) error` | Invalidate a token |
| `(*JWTAuth).ParseClaims` | `func (a *JWTAuth) ParseClaims(ctx, refreshToken) (*jwt.RegisteredClaims, error)` | Parse token claims |
| `(*JWTAuth).Release` | `func (a *JWTAuth) Release() error` | Release resources |
| `Storer` | `interface { Set(ctx, accessToken, expiration); Delete(ctx, accessToken) (bool, error); Check(ctx, accessToken) (bool, error); Close() error }` | Token storage interface |
| `WithSigningMethod` | `func WithSigningMethod(method jwt.SigningMethod) Option` | Set JWT signing method |
| `WithIssuer` | `func WithIssuer(issuer string) Option` | Set token issuer |
| `WithSigningKey` | `func WithSigningKey(key any) Option` | Set signing key |
| `WithKeyfunc` | `func WithKeyfunc(keyFunc jwt.Keyfunc) Option` | Set key function |
| `WithExpired` | `func WithExpired(expired time.Duration) Option` | Set token expiration |
| `WithTokenHeader` | `func WithTokenHeader(header map[string]any) Option` | Set custom token header |

**Predefined errors:** `ErrTokenInvalid`, `ErrTokenExpired`, `ErrTokenParseFail`, `ErrUnSupportSigningMethod`, `ErrSignTokenFailed`.

### 2.3 `pkg/authn/jwt/store/redis` — Redis Token Store

**Import:** `github.com/onexstack/onexstack/pkg/authn/jwt/store/redis`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewStore` | `func NewStore(cfg *Config) *Store` | Create Redis-backed token store |
| `Config` | `struct{ Addr, Username, Password string; Database int; KeyPrefix string }` | Redis config |

### 2.4 `pkg/authz` — Authorization (Casbin RBAC)

**Import:** `github.com/onexstack/onexstack/pkg/authz`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewAuthz` | `func NewAuthz(db *gorm.DB, opts ...Option) (*Authz, error)` | Create Casbin enforcer with GORM adapter |
| `(*Authz).Authorize` | `func (a *Authz) Authorize(sub, obj, act string) (bool, error)` | Check if subject can perform action on object |
| `Authz` | `struct{ *casbin.SyncedEnforcer }` | Thread-safe Casbin enforcer |
| `WithAclModel` | `func WithAclModel(model string) Option` | Set custom ACL model |
| `WithAutoLoadPolicyTime` | `func WithAutoLoadPolicyTime(interval time.Duration) Option` | Auto-reload policy |
| `ProviderSet` | `wire.ProviderSet` (includes `NewAuthz`, `DefaultOptions`) | Wire injection provider |
| `DefaultOptions` | `func DefaultOptions() []Option` | Get default options |

### 2.5 `pkg/token` — Token Management

**Import:** `github.com/onexstack/onexstack/pkg/token`

Centralized JWT token management with identity extraction, auto-refresh, and request parsing.

| API | Signature | Description |
|-----|-----------|-------------|
| `Init` | `func Init(key string, opts ...Option)` | Initialize the global token configuration |
| `Sign` | `func Sign(identityValue string) (string, time.Time, error)` | Sign a JWT with the identity |
| `SignWithClaims` | `func SignWithClaims(customClaims jwt.MapClaims) (string, time.Time, error)` | Sign with custom claims |
| `Parse` | `func Parse(tokenString string) error` | Validate a token |
| `ParseWithKey` | `func ParseWithKey(tokenString, key string) (jwt.MapClaims, error)` | Parse and validate with specific key |
| `ParseRequest` | `func ParseRequest(ctx context.Context) (string, error)` | Extract identity from HTTP request (respects skip paths) |
| `ParseRequestIgnoreSkip` | `func ParseRequestIgnoreSkip(ctx context.Context) (string, error)` | Extract identity, ignoring skip paths |
| `ParseIdentity` | `func ParseIdentity(tokenString, key string) (string, error)` | Parse identity from token |
| `GetClaims` | `func GetClaims(tokenString string) (jwt.MapClaims, error)` | Get all claims from a token |
| `Reset` | `func Reset()` | Reset global config |
| `GetConfig` | `func GetConfig() Config` | Get current config |
| `IsIdentityRequired` | `func IsIdentityRequired() bool` | Check if identity is required |
| `GetExpiration` | `func GetExpiration() time.Duration` | Get token expiration duration |
| `GetSkipPaths` | `func GetSkipPaths() []string` | Get paths that skip auth |
| `IsPathSkipped` | `func IsPathSkipped(path string) bool` | Check if a path skips auth |
| `WithKey` | `func WithKey(key string) Option` | Set signing key |
| `WithIdentityKey` | `func WithIdentityKey(identityKey string) Option` | Set identity claim key |
| `WithExpiration` | `func WithExpiration(expiration time.Duration) Option` | Set expiration |
| `WithSkipPaths` | `func WithSkipPaths(paths ...string) Option` | Skip auth for paths |
| `WithCommonSkipPaths` | `func WithCommonSkipPaths() Option` | Skip common paths (health, metrics) |
| `WithSkipPathsPattern` | `func WithSkipPathsPattern(patterns ...string) Option` | Glob-based skip patterns |

---

## 3. Data & Storage

### 3.1 `pkg/db` — Database Connections

**Import:** `github.com/onexstack/onexstack/pkg/db`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewMySQL` | `func NewMySQL(opts *MySQLOptions) (*gorm.DB, error)` | Create MySQL GORM instance |
| `NewPostgreSQL` | `func NewPostgreSQL(opts *PostgreSQLOptions) (*gorm.DB, error)` | Create PostgreSQL GORM instance |
| `NewSQLite` | `func NewSQLite(opts *SQLiteOptions) (*gorm.DB, error)` | Create SQLite GORM instance |
| `NewInMemorySQLite` | `func NewInMemorySQLite(dbFile string) (*gorm.DB, error)` | Create in-memory SQLite (for testing) |
| `NewRedis` | `func NewRedis(opts *RedisOptions) (*redis.Client, error)` | Create Redis client |
| `MustRawDB` | `func MustRawDB(db *gorm.DB) *sql.DB` | Extract underlying `*sql.DB`, panics on error |
| `TracePlugin` | `struct{}` — GORM plugin for SQL tracing | Registers before/after callbacks for all CRUD operations |
| `ProviderSet` | `wire.ProviderSet` | Wire injection set (MySQL + Redis) |
| `MySQLOptions` | `struct{ Addr, Username, Password, Database string; MaxIdleConnections, MaxOpenConnections int; MaxConnectionLifeTime time.Duration; Logger logger.Interface }` | MySQL options; has `DSN()` |
| `PostgreSQLOptions` | Similar struct to MySQLOptions | PostgreSQL options; has `DSN()` |
| `SQLiteOptions` | Similar struct to MySQLOptions | SQLite options; has `DSN()` |
| `RedisOptions` | `struct{ Addr, Username, Password string; Database, MaxRetries, MinIdleConns, PoolSize int; DialTimeout, ReadTimeout, WriteTimeout, PoolTimeout time.Duration }` | Redis options |

### 3.2 `pkg/store` — Generic Data Store

**Import:** `github.com/onexstack/onexstack/pkg/store`

A generic GORM-based data access layer using Go generics.

| API | Signature | Description |
|-----|-----------|-------------|
| `NewStore[T]` | `func NewStore[T any](storage DBProvider, logger Logger) *Store[T]` | Create a typed store |
| `(*Store[T]).Create` | `func (s *Store[T]) Create(ctx context.Context, obj *T) error` | Create a record |
| `(*Store[T]).Update` | `func (s *Store[T]) Update(ctx context.Context, obj *T) error` | Update a record |
| `(*Store[T]).Delete` | `func (s *Store[T]) Delete(ctx context.Context, opts *where.Options) error` | Delete record(s) |
| `(*Store[T]).Get` | `func (s *Store[T]) Get(ctx context.Context, opts *where.Options) (*T, error)` | Get a single record |
| `(*Store[T]).List` | `func (s *Store[T]) List(ctx context.Context, opts *where.Options) (count int64, ret []*T, err error)` | List records with count |
| `DBProvider` | `interface { DB(ctx, wheres ...where.Where) *gorm.DB }` | Database provider interface |
| `Logger` | `interface { Error(ctx, err, message, kvs ...any) }` | Store logger interface |

**Logger sub-packages:**

| Sub-package | Import | Description |
|-------------|--------|-------------|
| `store/logger/empty` | `.../pkg/store/logger/empty` | `NewLogger()` — no-op store logger |
| `store/logger/onex` | `.../pkg/store/logger/onex` | `NewLogger()` — onex-flavored store logger |

### 3.3 `pkg/store/where` — Query Condition Builder

**Import:** `github.com/onexstack/onexstack/pkg/store/where`

Fluent API for building GORM Where query conditions.

| API | Signature | Description |
|-----|-----------|-------------|
| `NewWhere` | `func NewWhere(opts ...Option) *Options` | Create new where options |
| `WithOffset` | `func WithOffset(offset int64) Option` | Set offset |
| `WithLimit` | `func WithLimit(limit int64) Option` | Set limit |
| `WithPage` | `func WithPage(page, pageSize int) Option` | Set pagination (page number + page size) |
| `WithFilter` | `func WithFilter(filter map[any]any) Option` | Set filter map |
| `WithClauses` | `func WithClauses(conds ...clause.Expression) Option` | Add GORM clause expressions |
| `WithQuery` | `func WithQuery(query interface{}, args ...interface{}) Option` | Add raw SQL query |
| `(*Options).Where` | `func (whr *Options) Where(db *gorm.DB) *gorm.DB` | Apply all conditions to a GORM query |
| **Shorthand functions:** | `O(offset)`, `L(limit)`, `P(page, pageSize)`, `C(conds...)`, `Q(query, args...)`, `T(ctx)`, `F(kvs...)` | Single-letter convenience constructors for `*Options` |
| `Where` | `interface { Where(db *gorm.DB) *gorm.DB }` | Where clause interface |
| `RegisterTenant` | `func RegisterTenant(key string, valueFunc func(context.Context) string)` | Register a tenant filter (`T()` uses it) |
| `Tenant` | `struct{ Key string; ValueFunc func(ctx) string }` | Tenant where clause |

**Usage pattern:** `store.List(ctx, where.P(1, 20).F("status", "active").Q("name LIKE ?", "%test%"))`

### 3.4 `pkg/store/registry` — GORM Model Registry

**Import:** `github.com/onexstack/onexstack/pkg/store/registry`

| API | Signature | Description |
|-----|-----------|-------------|
| `Register` | `func Register(model interface{})` | Register a model for auto-migration |
| `Migrate` | `func Migrate(db *gorm.DB) error` | Auto-migrate all registered models |
| `NewRegistry` | `func NewRegistry() *Registry` | Create a new model registry |

### 3.5 `pkg/distlock` — Distributed Locking

**Import:** `github.com/onexstack/onexstack/pkg/distlock`

| API | Signature | Description |
|-----|-----------|-------------|
| `Locker` | `interface { Lock(ctx) error; Unlock(ctx) error; Renew(ctx) error }` | Common lock interface |
| `NewOptions` | `func NewOptions() *Options` | Default lock options |
| `ApplyOptions` | `func ApplyOptions(opts ...Option) *Options` | Apply functional options |
| `WithLockName` | `func WithLockName(name string) Option` | Set lock name |
| `WithLockTimeout` | `func WithLockTimeout(timeout time.Duration) Option` | Set lock timeout |
| `WithOwnerID` | `func WithOwnerID(ownerID string) Option` | Set owner identifier |
| `WithLogger` | `func WithLogger(logger logger.Logger) Option` | Set logger |
| `WithAutoMigrate` | `func WithAutoMigrate(autoMigrate bool) Option` | Enable GORM auto-migration |

**Backend implementations (all implement `Locker`):**

| Constructor | Signature | Backend |
|-------------|-----------|---------|
| `NewRedisLocker` | `func NewRedisLocker(client *redis.Client, opts ...Option) *RedisLocker` | Redis |
| `NewGORMLocker` | `func NewGORMLocker(db *gorm.DB, opts ...Option) (*GORMLocker, error)` | MySQL/PostgreSQL/SQLite |
| `NewEtcdLocker` | `func NewEtcdLocker(endpoints []string, opts ...Option) (*EtcdLocker, error)` | Etcd |
| `NewConsulLocker` | `func NewConsulLocker(consulAddr string, opts ...Option) (*ConsulLocker, error)` | Consul |
| `NewMongoLocker` | `func NewMongoLocker(mongoURI, dbName string, opts ...Option) (*MongoLocker, error)` | MongoDB |
| `NewMemcachedLocker` | `func NewMemcachedLocker(memcachedAddr string, opts ...Option) *MemcachedLocker` | Memcached |
| `NewZookeeperLocker` | `func NewZookeeperLocker(zkServers []string, opts ...Option) (*ZookeeperLocker, error)` | ZooKeeper |
| `NewNoopLocker` | `func NewNoopLocker(opts ...Option) *NoopLocker` | No-op (testing/disabled) |

---

## 4. Request Handling & Core Utilities

### 4.1 `pkg/core` — Request Handler Framework

**Import:** `github.com/onexstack/onexstack/pkg/core`

Generic request handlers for Gin that handle binding → validation → execution → response.

| API | Signature | Description |
|-----|-----------|-------------|
| `HandleAllRequest` | `func HandleAllRequest[T, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T])` | Bind URI+Query+JSON, validate, execute, respond |
| `HandleJSONRequest` | `func HandleJSONRequest[T, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T])` | Bind JSON body, validate, execute, respond |
| `HandleQueryRequest` | `func HandleQueryRequest[T, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T])` | Bind query params, validate, execute, respond |
| `HandleUriRequest` | `func HandleUriRequest[T, R any](c *gin.Context, handler Handler[T, R], validators ...Validator[T])` | Bind URI params, validate, execute, respond |
| `HandleRequest` | `func HandleRequest[T, R any](c *gin.Context, binder Binder, handler Handler[T, R], validators ...Validator[T])` | Generic: custom binder, validate, execute, respond |
| `ShouldBindJSON` | `func ShouldBindJSON[T any](c *gin.Context, rq *T, validators ...Validator[T]) error` | Bind JSON and validate |
| `ShouldBindQuery` | `func ShouldBindQuery[T any](c *gin.Context, rq *T, validators ...Validator[T]) error` | Bind query params and validate |
| `ShouldBindUri` | `func ShouldBindUri[T any](c *gin.Context, rq *T, validators ...Validator[T]) error` | Bind URI params and validate |
| `ShouldBindAll` | `func ShouldBindAll[T any](c *gin.Context, rq *T, validators ...Validator[T]) error` | Bind URI+Query+JSON and validate |
| `ReadRequest` | `func ReadRequest[T any](c *gin.Context, rq *T, binder Binder, validators ...Validator[T]) error` | Generic bind + validate |
| `FinalizeRequest` | `func FinalizeRequest[T any](c *gin.Context, rq *T, validators ...Validator[T]) error` | Call Default() then validate |
| `WriteResponse` | `func WriteResponse(c *gin.Context, data any, err error)` | Write JSON response (success or error) |
| `Handler[T, R]` | `type Handler[T any, R any] func(ctx context.Context, req *T) (R, error)` | Handler function type |
| `Validator[T]` | `type Validator[T any] func(context.Context, *T) error` | Validator function type |
| `Binder` | `type Binder func(any) error` | Binder function type |
| `ErrorResponse` | `struct{ Reason, Message string; Metadata map[string]string }` | Standard error response |

**Error handling helpers:**
- `RecordSpanError(ctx, span, err, attrs...)` — record error on OTel span
- `RecordSpanErrorWithLog(ctx, span, err, message, attrs...)` — record + log
- `LogSpanError(ctx, span, err, attrs...)` — convenience wrapper
- `RecordSpanErrorf(ctx, span, err, format, args...)` — formatted error recording
- `RecordSpanErrorfWithAttrs(ctx, span, err, format, args, attrs...)` — formatted with attrs

**Config & Copy:**
- `OnInitialize(cfgFile, envPrefix, searchPaths, configName) func()` — Viper config initializer
- `Copy(to, from any) error` — shallow copy via copier library
- `CopyWithConverters(to, from any, converters ...copier.TypeConverter) error` — deep copy with type converters
- `TypeConverters() []copier.TypeConverter` — default `time.Time ↔ *timestamppb.Timestamp` converters

### 4.2 `pkg/errorsx` — Structured Error Handling

**Import:** `github.com/onexstack/onexstack/pkg/errorsx`

| API | Signature | Description |
|-----|-----------|-------------|
| `New` | `func New(code int, reason, format string, args ...any) *ErrorX` | Create a new error |
| `(*ErrorX).Error` | `func (err *ErrorX) Error() string` | Implements `error` interface |
| `(*ErrorX).WithMessage` | `func (err *ErrorX) WithMessage(format string, args ...any) *ErrorX` | Set message with formatting |
| `(*ErrorX).WithMetadata` | `func (err *ErrorX) WithMetadata(md map[string]string) *ErrorX` | Set metadata |
| `(*ErrorX).KV` | `func (err *ErrorX) KV(kvs ...string) *ErrorX` | Set metadata via key-value pairs |
| `(*ErrorX).WithRequestID` | `func (err *ErrorX) WithRequestID(requestID string) *ErrorX` | Attach request ID |
| `(*ErrorX).GRPCStatus` | `func (err *ErrorX) GRPCStatus() *status.Status` | Convert to gRPC status with error details |
| `(*ErrorX).Is` | `func (err *ErrorX) Is(target error) bool` | Implements `errors.Is` (matches Code + Reason) |
| `Code` | `func Code(err error) int` | Extract HTTP status code from error |
| `Reason` | `func Reason(err error) string` | Extract reason string from error |
| `FromError` | `func FromError(err error) *ErrorX` | Convert any error to `*ErrorX` (handles gRPC status too) |
| `Is` / `As` / `Unwrap` | Wrappers for `errors.Is`, `errors.As`, `errors.Unwrap` | Standard error utilities |

**Predefined errors (with HTTP codes):**

| Variable | Code | Reason |
|----------|------|--------|
| `OK` | 200 | — |
| `ErrInternal` | 500 | `InternalError` |
| `ErrNotFound` | 404 | `NotFound` |
| `ErrBind` | 400 | `BindError` |
| `ErrInvalidArgument` | 400 | `InvalidArgument` |
| `ErrUnauthenticated` | 401 | `Unauthenticated` |
| `ErrPermissionDenied` | 403 | `PermissionDenied` |
| `ErrOperationFailed` | 409 | `OperationFailed` |

### 4.3 `pkg/binding` — Request Binding

**Import:** `github.com/onexstack/onexstack/pkg/binding`

| API | Signature | Description |
|-----|-----------|-------------|
| `Bind` | `func Bind(c *gin.Context, obj interface{}, bindFuncs ...func(*gin.Context, interface{}) error) error` | Chain multiple binders (URI+JSON, etc.) |
| `URI` | `func URI(c *gin.Context, obj interface{}) error` | Bind URI parameters |
| `JSON` | `func JSON(c *gin.Context, obj interface{}) error` | Bind JSON body |

Uses a custom validator engine (replaces Gin's default validator).

### 4.4 `pkg/validation` — Request Validation

**Import:** `github.com/onexstack/onexstack/pkg/validation`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewValidator` | `func NewValidator(customValidator any) *Validator` | Create validator with custom validation rules |
| `(*Validator).Validate` | `func (v *Validator) Validate(ctx context.Context, request any) error` | Validate a request |
| `ValidRequired` | `func ValidRequired(obj any, requiredFields ...string) error` | Validate required fields are non-empty |
| `ValidateAllFields` | `func ValidateAllFields(obj any, rules Rules) error` | Validate all fields with rules |
| `ValidateSelectedFields` | `func ValidateSelectedFields(obj any, rules Rules, fields ...string) error` | Validate selected fields |
| `GetExportedFieldNames` | `func GetExportedFieldNames(obj any) []string` | Get exported field names of a struct |
| `ValidatorFunc` | `type ValidatorFunc func(value any) error` | Custom validator function type |
| `Rules` | `type Rules map[string]ValidatorFunc` | Field name → validator function |

### 4.5 `pkg/ptr` — Generic Pointer Utilities

**Import:** `github.com/onexstack/onexstack/pkg/ptr`

| API | Signature | Description |
|-----|-----------|-------------|
| `To[T]` | `func To[T any](v T) *T` | Convert value to pointer |
| `From[T]` | `func From[T any](v *T) T` | Dereference pointer (zero value if nil) |
| `FromOr[T]` | `func FromOr[T any](ptr *T, def T) T` | Dereference with default fallback |
| `IsNil[T]` | `func IsNil[T any](p *T) bool` | Check if pointer is nil |
| `IsNotNil[T]` | `func IsNotNil[T any](p *T) bool` | Check if pointer is not nil |
| `Clone[T]` | `func Clone[T any](p *T) *T` | Shallow clone of pointer value |
| `CloneBy[T]` | `func CloneBy[T any](p *T, f func(T) T) *T` | Clone with transformation |
| `Equal[T comparable]` | `func Equal[T comparable](a, b *T) bool` | Compare two pointers |
| `EqualTo[T comparable]` | `func EqualTo[T comparable](p *T, v T) bool` | Check if pointer equals value |
| `Map[F, T]` | `func Map[F, T any](p *F, f func(F) T) *T` | Apply function to pointer value |
| `AllPtrFieldsNil` | `func AllPtrFieldsNil(obj interface{}) bool` | Check if all pointer fields in struct are nil |

### 4.6 `pkg/id` — ID Generation

**Import:** `github.com/onexstack/onexstack/pkg/id`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewCode` | `func NewCode(id uint64, options ...func(*CodeOptions)) string` | Generate a short code from a uint64 |
| `NewSonyflake` | `func NewSonyflake(options ...func(*SonyflakeOptions)) *Sonyflake` | Create a Sonyflake ID generator |
| `(*Sonyflake).Id` | `func (s *Sonyflake) Id(ctx context.Context) uint64` | Generate a unique ID |
| `WithCodeChars` / `WithCodeN1` / `WithCodeN2` / `WithCodeL` / `WithCodeSalt` | Code generation options | Customize short code generation |
| `WithSonyflakeMachineId` / `WithSonyflakeStartTime` | Sonyflake options | Customize Sonyflake generator |

### 4.7 `pkg/rid` — Resource ID Generation

**Import:** `github.com/onexstack/onexstack/pkg/rid`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewResourceID` | `func NewResourceID(prefix string) ResourceID` | Create a resource ID generator with prefix |
| `ResourceID.New` | `func (rid ResourceID) New(counter uint64) string` | Generate a prefixed ID (e.g., `"{prefix}-{6-char code}"`) |
| `Salt` | `func Salt() uint64` | Get machine-based salt value |
| `ReadMachineID` | `func ReadMachineID() []byte` | Read machine ID from OS |

### 4.8 `pkg/version` — Version Information

**Import:** `github.com/onexstack/onexstack/pkg/version`

| API | Signature | Description |
|-----|-----------|-------------|
| `Get` | `func Get() Info` | Get current build version info |
| `GetFromDebugInfo` | `func GetFromDebugInfo(modulePath string) Info` | Get version from debug.BuildInfo |
| `SetDynamicVersion` | `func SetDynamicVersion(dynamicVersion string) error` | Set dynamic version string |
| `ValidateDynamicVersion` | `func ValidateDynamicVersion(dynamicVersion string) error` | Validate version format |
| `PrintAndExitIfRequested` | `func PrintAndExitIfRequested()` | Print version and exit if `--version` flag is set |
| `AddFlags` | `func AddFlags(fs *pflag.FlagSet)` | Add version flags to a flag set |
| `Info` | `struct{ GitVersion, GitCommit, GitTreeState, BuildDate, GoVersion, Compiler, Platform string }` | Version info; methods: `String()`, `ToJSON()`, `Text()` |

### 4.9 `pkg/i18n` — Internationalization

**Import:** `github.com/onexstack/onexstack/pkg/i18n`

| API | Signature | Description |
|-----|-----------|-------------|
| `New` | `func New(options ...func(*Options)) *I18n` | Create an i18n instance |
| `(*I18n).Select` | `func (i I18n) Select(lang language.Tag) *I18n` | Select a language |
| `(*I18n).T` | `func (i I18n) T(id string) string` | Translate a message by ID |
| `(*I18n).E` | `func (i I18n) E(id string) error` | Translate a message as error |
| `(*I18n).LocalizeT` | `func (i I18n) LocalizeT(message *i18n.Message) string` | Localize a go-i18n message |
| `(*I18n).LocalizeE` | `func (i I18n) LocalizeE(message *i18n.Message) error` | Localize a go-i18n message as error |
| `(*I18n).Language` | `func (i I18n) Language() language.Tag` | Get current language |
| `(*I18n).Add` | `func (i *I18n) Add(f string)` | Add a translation file |
| `(*I18n).AddFS` | `func (i *I18n) AddFS(fs embed.FS)` | Add translation files from embedded FS |
| `WithContext` | `func WithContext(ctx context.Context, i *I18n) context.Context` | Store i18n in context |
| `FromContext` | `func FromContext(ctx context.Context) *I18n` | Retrieve i18n from context |
| `WithFormat` / `WithLanguage` / `WithFile` / `WithFS` | i18n options | Configure i18n instance |

### 4.10 `pkg/flux` — Generic Cache with Async Refresh

**Import:** `github.com/onexstack/onexstack/pkg/flux`

A type-safe, in-memory cache with multiple refresh modes and loader interfaces.

| API | Signature | Description |
|-----|-----------|-------------|
| `NewFlux[K, V]` | `func NewFlux[K Key, V any](opts ...Option[K, V]) (*Flux[K, V], error)` | Create a new cache |
| `(*Flux).Get` | `func (f *Flux[K, V]) Get(ctx context.Context, key K) (V, error)` | Get a value (loads if not cached) |
| `(*Flux).MustGet` | `func (f *Flux[K, V]) MustGet(ctx context.Context, key K) V` | Get a value, panic on error |
| `(*Flux).Set` | `func (f *Flux[K, V]) Set(ctx context.Context, key K, value V) error` | Set a value directly |
| `(*Flux).Del` | `func (f *Flux[K, V]) Del(key K) error` | Delete a cached key |
| `(*Flux).Refresh` | `func (f *Flux[K, V]) Refresh(ctx context.Context, key K) error` | Force-refresh a key |
| `(*Flux).Keys` | `func (f *Flux[K, V]) Keys() []K` | List cached keys |
| `(*Flux).Stats` | `func (f *Flux[K, V]) Stats() *Stats` | Get cache statistics |
| `(*Flux).GetLoaderCapabilities` | `func (f *Flux[K, V]) GetLoaderCapabilities() map[string]bool` | Check loader capabilities |
| `(*Flux).Close` | `func (f *Flux[K, V]) Close() error` | Close the cache |
| `Config[K, V]` | `struct{ MaxKeys, BufferItems int64; TTL, LoadTimeout, RefreshInterval time.Duration; Loader; EnableAsync bool; RefreshMode; MaxConcurrency, BatchSize int; EnableStats bool }` | Cache configuration |
| `Loader[K, V]` | `interface { Load(ctx, key K) (V, error) }` | Basic loader |
| `BatchLoader[K, V]` | `interface { Loader[K, V]; LoadBatch(ctx, keys []K) (map[K]V, error) }` | Batch-loading support |
| `AllLoader[K, V]` | `interface { Loader[K, V]; LoadAll(ctx) (map[K]V, error) }` | Full-load support |
| `FullLoader[K, V]` | `interface { AllLoader[K, V]; BatchLoader[K, V] }` | Both batch + full load |
| `RefreshMode` | `RefreshModeSingle` (0), `RefreshModeBatch` (1), `RefreshModeAll` (2), `RefreshModeAuto` (3) | Refresh strategies |

**Options:** `WithMaxKeys`, `WithTTL`, `WithNoExpiration`, `WithLoader`, `WithLoadTimeout`, `WithAsyncRefresh`, `WithEnableAsync`, `WithRefreshMode`, `WithMaxConcurrency`, `WithBatchSize`, `WithBufferItems`, `WithStats`, `WithAsyncConfig`, `WithConfig`.

---

## 5. Logging

### 5.1 `pkg/log` — Primary Logging (Zap-based)

**Import:** `github.com/onexstack/onexstack/pkg/log`

A Zap-based logger that implements both Kratos and GORM logger interfaces.

| API | Signature | Description |
|-----|-----------|-------------|
| `Init` | `func Init(opts *Options, options ...Option)` | Initialize the global logger |
| `NewLogger` | `func NewLogger(opts *Options, options ...Option) *zapLogger` | Create a new logger instance |
| `NewOptions` | `func NewOptions() *Options` | Create default options |
| `Default` | `func Default() Logger` | Get the default (global) logger |
| `Options` | `struct{ DisableCaller, DisableStacktrace, EnableColor bool; Level, Format string; OutputPaths []string }` | Logger options; has `Validate()`, `AddFlags()` |
| `Logger` | Interface (embeds `krtlog.Logger` + `gormlogger.Interface`): `Debugf/W/Infof/W/Warnf/W/Errorf/W/Panicf/W/Fatalf/W`, `W(ctx) Logger`, `AddCallerSkip(skip) Logger`, `Sync()` | Full logger interface |
| `WithContextExtractor` | `func WithContextExtractor(contextExtractors ContextExtractors) Option` | Extract fields from context for logging |
| `ContextExtractors` | `type ContextExtractors map[string]func(context.Context) string` | Context extractor map |

**Package-level convenience functions:** `Sync()`, `Debugf/W`, `Infof/W`, `Warnf/W`, `Errorf/W`, `Panicf/W`, `Fatalf/W`, `W(ctx)`, `AddCallerSkip(skip)`.

### 5.2 `pkg/logger` — Slim Logger Interface

**Import:** `github.com/onexstack/onexstack/pkg/logger`

A minimal logger interface used by components that don't need the full `log.Logger`.

| API | Signature | Description |
|-----|-----------|-------------|
| `Logger` | `interface { Debug/Warn/Info/Error(msg, kvs...); DebugContext/WarnContext/InfoContext/ErrorContext(ctx, msg, kvs...); With(args...) Logger }` | Slim logger interface |

**Adapter implementations:**

| Sub-package | Import | Constructor | Description |
|-------------|--------|-------------|-------------|
| `logger/empty` | `.../pkg/logger/empty` | `NewLogger() logger.Logger` | No-op logger (testing/disabled) |
| `logger/onex` | `.../pkg/logger/onex` | `NewLogger()` | Bridges `pkg/log` to `logger.Logger` |
| `logger/slog` | `.../pkg/logger/slog` | `NewSlogAdapter(l *slog.Logger) logger.Logger` | Wraps `log/slog` as `logger.Logger` |
| `logger/gorm/empty` | `.../pkg/logger/gorm/empty` | `NewSilentLogger() logger.Interface` | Silent GORM logger |
| `logger/slog/gorm` | `.../pkg/logger/slog/gorm` | `New(l *slog.Logger, opts ...Option) *SlogLogger` | GORM logger backed by slog; options: `WithLogLevel`, `WithSlowThreshold`, `WithIgnoreRecordNotFoundError` |
| `logger/slog/breeze` | `.../pkg/logger/slog/breeze` | `NewLogger() *Logger` | Breeze framework logger adapter |
| `logger/slog/resty` | `.../pkg/logger/slog/resty` | `NewLogger() *Logger` | Resty HTTP client logger adapter |
| `logger/slog/lark` | `.../pkg/logger/slog/lark` | `NewSlogWrapper(l *slog.Logger) *SlogWrapper` | Lark/Feishu SDK logger adapter |
| `logger/klog/kratos` | `.../pkg/logger/klog/kratos` | `NewLogger() *Logger` | Kratos logger via klog |
| `logger/klog/store` | `.../pkg/logger/klog/store` | `NewLogger() *Logger` | Store logger via klog |
| `logger/slog/kratos` | `.../pkg/logger/slog/kratos` | `NewLogger() *Logger` | Kratos logger via slog |
| `logger/slog/store` | `.../pkg/logger/slog/store` | `NewLogger() *Logger` | Store logger via slog |

---

## 6. HTTP/gRPC Middleware

### 6.1 `pkg/middleware/gin` — Gin HTTP Middleware

**Import:** `github.com/onexstack/onexstack/pkg/middleware/gin`

| API | Signature | Description |
|-----|-----------|-------------|
| `Observability` | `func Observability(opts ...Option) gin.HandlerFunc` | Full observability middleware (tracing, request logging, metrics) |
| `ObservabilityWithW3CTraceContext` | `func ObservabilityWithW3CTraceContext() gin.HandlerFunc` | W3C trace context propagation |
| `ObservabilityWithTraceID` | `func ObservabilityWithTraceID() gin.HandlerFunc` | Trace ID only mode |
| `ObservabilityWithCustomHeader` | `func ObservabilityWithCustomHeader(headerName string) gin.HandlerFunc` | Custom trace header |
| `ObservabilitySkipMetrics` | `func ObservabilitySkipMetrics() gin.HandlerFunc` | Observability without metrics |
| `ObservabilityWithSkipPaths` | `func ObservabilityWithSkipPaths(paths ...string) gin.HandlerFunc` | Skip observability for paths |
| `ObservabilityOptions` | `struct{ TraceInjectionMode; CustomTraceHeader string; SkipPaths []string; Logger *slog.Logger; DisableBodyLog bool }` | Middleware options |
| `TraceInjectionMode` | `InjectW3CTraceContext` (0), `InjectTraceIDOnly` (1), `InjectBoth` (2), `InjectNone` (3) | Trace injection modes |

### 6.2 `pkg/middleware/grpc` — gRPC Middleware

**Import:** `github.com/onexstack/onexstack/pkg/middleware/grpc`

| API | Signature | Description |
|-----|-----------|-------------|
| `Observability` | `func Observability(opts ...Option) grpc.UnaryServerInterceptor` | Full gRPC observability interceptor |
| `ObservabilityWithW3CTraceContext` | `func ObservabilityWithW3CTraceContext() grpc.UnaryServerInterceptor` | W3C trace context |
| `ObservabilityWithTraceID` | `func ObservabilityWithTraceID() grpc.UnaryServerInterceptor` | Trace ID only |
| `ObservabilityWithCustomHeader` | `func ObservabilityWithCustomHeader(headerName string) grpc.UnaryServerInterceptor` | Custom trace header |
| `ObservabilityOptions` | `struct{ TraceInjectionMode; CustomTraceHeader string }` | Interceptor options |
| `TraceInjectionMode` | `InjectW3CTraceContext` (0), `InjectTraceIDOnly` (1), `InjectBoth` (2), `InjectNone` (3) | Trace injection modes |

---

## 7. Observability (OpenTelemetry)

### 7.1 `pkg/otelslog` — OpenTelemetry + slog Bridge

**Import:** `github.com/onexstack/onexstack/pkg/otelslog`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewLogger` | `func NewLogger(name string, options ...Option) *slog.Logger` | Create OTel-backed slog logger |
| `NewHandler` | `func NewHandler(name string, options ...Option) *Handler` | Create OTel-backed slog handler |
| `WithVersion` | `func WithVersion(version string) Option` | Set version |
| `WithSchemaURL` | `func WithSchemaURL(schemaURL string) Option` | Set schema URL |
| `WithAttributes` | `func WithAttributes(attributes ...attribute.KeyValue) Option` | Set static attributes |
| `WithLoggerProvider` | `func WithLoggerProvider(provider log.LoggerProvider) Option` | Set logger provider |
| `WithSource` | `func WithSource(source bool) Option` | Enable source location |
| `WithLevel` | `func WithLevel(level slog.Level) Option` | Set log level |
| `WithLevelString` | `func WithLevelString(levelStr string) Option` | Set log level from string |

### 7.2 `pkg/otel/emptyexporter` — Fake OTel Exporter

**Import:** `github.com/onexstack/onexstack/pkg/otel/emptyexporter`

| API | Signature | Description |
|-----|-----------|-------------|
| `InitFakeTracer` | `func InitFakeTracer(serviceName string, logSpans bool)` | Initialize a fake tracer for testing |
| `FakeExporter` | `struct{ LogSpans bool; Handle func(ctx, span) }` | Customizable fake span exporter |

### 7.3 `pkg/otel/exporter/empty` — Empty OTel Exporter

**Import:** `github.com/onexstack/onexstack/pkg/otel/exporter/empty`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewEmptyExporter` | `func NewEmptyExporter() *Exporter` | Create a no-op exporter |

---

## 8. CLI Utilities

### 8.1 `pkg/cli/cli` — Config File Path Helpers

**Import:** `github.com/onexstack/onexstack/pkg/cli/cli`

| API | Signature | Description |
|-----|-----------|-------------|
| `SearchDirs` | `func SearchDirs(defaultHomeDir string) []string` | Get config file search directories (XDG-compatible) |
| `FilePath` | `func FilePath(defaultHomeDir, defaultConfigName string) string` | Get the config file path |

### 8.2 `pkg/cli/genericclioptions` — CLI I/O Utilities

**Import:** `github.com/onexstack/onexstack/pkg/cli/genericclioptions`

| API | Signature | Description |
|-----|-----------|-------------|
| `IOStreams` | `struct{ In io.Reader; Out, ErrOut io.Writer }` | Standard I/O streams |
| `NewTestIOStreams` | `func NewTestIOStreams() (IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer)` | Create test I/O streams |
| `NewTestIOStreamsDiscard` | `func NewTestIOStreamsDiscard() IOStreams` | Create discard-only I/O streams |
| `CommandHeaderRoundTripper` | `struct{ Delegate http.RoundTripper; Headers map[string]string }` | HTTP round tripper with command headers; methods: `RoundTrip`, `ParseCommandHeaders`, `CancelRequest` |

### 8.3 `pkg/polaris` — Polaris Load Balancer

**Import:** `github.com/onexstack/onexstack/pkg/polaris`

| API | Signature | Description |
|-----|-----------|-------------|
| `NewPolarisLB` | `func NewPolarisLB(namespace, service, protocol, addr string) (*PolarisLB, error)` | Create Polaris load balancer for resty |
| `(*PolarisLB).Next` | `func (m *PolarisLB) Next() (string, error)` | Get next instance address |
| `(*PolarisLB).Feedback` | `func (m *PolarisLB) Feedback(rf *resty.RequestFeedback)` | Provide feedback for load balancing |
| `(*PolarisLB).Close` | `func (m *PolarisLB) Close() error` | Close the load balancer |

---

## 9. Utility Packages

### 9.1 `pkg/util/retry` — Retry & Polling

**Import:** `github.com/onexstack/onexstack/pkg/util/retry`

| API | Signature | Description |
|-----|-----------|-------------|
| `Retry` | `func Retry(fn wait.ConditionFunc, initialBackoffSec int) error` | Exponential backoff retry |
| `Poll` | `func Poll(interval, timeout time.Duration, condition wait.ConditionFunc) error` | Poll until condition true or timeout |
| `PollImmediate` | `func PollImmediate(interval, timeout time.Duration, condition wait.ConditionFunc) error` | Poll immediately, then at interval |
| `RunImmediatelyThenPeriod` | `func RunImmediatelyThenPeriod(ctx context.Context, f func(context.Context) error, period time.Duration) error` | Run once sync, then periodically async |

### 9.2 `pkg/util/file` — File System Operations

**Import:** `github.com/onexstack/onexstack/pkg/util/file`

| API | Signature | Description |
|-----|-----------|-------------|
| `FileExists` | `func FileExists(path string) (bool, error)` | Check if file exists |
| `DirExists` | `func DirExists(path string) (bool, error)` | Check if directory exists |
| `EnsureDir` | `func EnsureDir(path string) error` | Create directory if not exists |
| `EnsureDirAll` | `func EnsureDirAll(path string) error` | Create all directories (like `mkdir -p`) |
| `RemoveDir` | `func RemoveDir(path string) error` | Remove a directory |
| `EmptyDir` | `func EmptyDir(path string) error` | Empty a directory |
| `Touch` | `func Touch(path string) error` | Touch a file (create if not exists) |
| `WriteFile` | `func WriteFile(path string, file []byte) error` | Write bytes to file |
| `ListDir` | `func ListDir(path string) []string` | List directory contents |
| `FileType` | `func FileType(filePath string) (types.Type, error)` | Detect MIME type of a file |
| `GetHomeDirectory` | `func GetHomeDirectory() string` | Get user home directory |
| `SafeMove` | `func SafeMove(src, dst string) error` | Safe file move with validation |
| `GetIntraDir` | `func GetIntraDir(pattern string, depth, length int) string` | Generate intra-directory path |
| `GetParent` | `func GetParent(path string) *string` | Get parent directory path |
| `IsZipFileUncompressed` | `func IsZipFileUncompressed(path string) (bool, error)` | Check if a zip file is uncompressed |
| `ServeFileNoCache` | `func ServeFileNoCache(w http.ResponseWriter, r *http.Request, filepath string)` | Serve file with no-cache headers |
| `MatchEntries` | `func MatchEntries(dir, pattern string) ([]string, error)` | Glob match files in directory |

### 9.3 `pkg/util/strings` — String Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/strings`

| API | Signature | Description |
|-----|-----------|-------------|
| `CamelCaseToUnderscore` | `func CamelCaseToUnderscore(str string) string` | Convert camelCase/PascalCase to snake_case |
| `UnderscoreToCamelCase` | `func UnderscoreToCamelCase(str string) string` | Convert snake_case to CamelCase |
| `Diff` | `func Diff(base, exclude []string) []string` | Set difference (base - exclude) |
| `Include` | `func Include(base, include []string) []string` | Set union |
| `Unique` | `func Unique(ss []string) []string` | Deduplicate strings |
| `FindString` | `func FindString(array []string, str string) int` | Find index of string |
| `StringIn` | `func StringIn(str string, array []string) bool` | Check if string is in array |
| `Contains` | `func Contains(list []string, strToSearch string) bool` | Check if list contains string |
| `ContainsEqualFold` | `func ContainsEqualFold(slice []string, s string) bool` | Case-insensitive contains |
| `Reverse` | `func Reverse(s string) string` | Reverse a string |
| `Filter` | `func Filter(list []string, strToFilter string) []string` | Remove a string from list |
| `Add` | `func Add(list []string, str string) []string` | Add string if not present |
| `FrequencySort` | `func FrequencySort(list []string) []string` | Sort by frequency (descending) |
| `DecodeBase64` | `func DecodeBase64(i string) ([]byte, error)` | Decode base64 string |

### 9.4 `pkg/util/reflect` — Reflection Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/reflect`

| API | Signature | Description |
|-----|-----------|-------------|
| `ToGormDBMap` | `func ToGormDBMap(obj any, fields []string) (map[string]any, error)` | Convert struct to GORM-compatible map |
| `GetObjFieldsMap` | `func GetObjFieldsMap(obj any, fields []string) map[string]any` | Get selected fields as map |
| `CopyObj` | `func CopyObj(from any, to any, fields []string) (changed bool, err error)` | Copy selected fields between structs |
| `CopyObjViaYaml` | `func CopyObjViaYaml(to any, from any) error` | Deep copy via YAML marshal/unmarshal |
| `StructName` | `func StructName(obj any) string` | Get struct type name |

### 9.5 `pkg/util/ip` — IP Address Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/ip`

| API | Signature | Description |
|-----|-----------|-------------|
| `GetLocalIP` | `func GetLocalIP() string` | Get local machine IP |
| `RemoteIP` | `func RemoteIP(req *http.Request) string` | Extract client IP from request (handles X-Forwarded-For, X-Real-IP, x-client-ip) |

### 9.6 `pkg/util/version` — Semantic Version Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/version`

| API | Signature | Description |
|-----|-----------|-------------|
| `ParseGeneric` | `func ParseGeneric(str string) (*Version, error)` | Parse a version string (generic format) |
| `MustParseGeneric` | `func MustParseGeneric(str string) *Version` | Parse or panic |
| `ParseSemantic` | `func ParseSemantic(str string) (*Version, error)` | Parse semantic version |
| `MustParseSemantic` | `func MustParseSemantic(str string) *Version` | Parse semantic or panic |
| `MajorMinor` | `func MajorMinor(major, minor uint) *Version` | Create version from major+minor |
| `HighestSupportedVersion` | `func HighestSupportedVersion(versions []string) (*Version, error)` | Find highest version in a list |
| `(*Version).Major/Minor/Patch/PreRelease/BuildMetadata/Components` | Accessor methods | Version component access |
| `(*Version).AtLeast` | `func (v *Version) AtLeast(min *Version) bool` | Check if v >= min |
| `(*Version).LessThan` | `func (v *Version) LessThan(other *Version) bool` | Check if v < other |
| `(*Version).Compare` | `func (v *Version) Compare(other string) (int, error)` | Compare two versions |
| `(*Version).WithMajor/WithMinor/WithPatch/WithPreRelease/WithBuildMetadata` | Builder methods | Create modified version |

### 9.7 `pkg/util/pagination` — Pagination Helper

**Import:** `github.com/onexstack/onexstack/pkg/util/pagination`

| API | Signature | Description |
|-----|-----------|-------------|
| `GetPageOffset` | `func GetPageOffset(pageNum, pageSize int64) int64` | Calculate SQL offset from page number and size |

### 9.8 `pkg/util/controller` — Controller Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/controller`

| API | Signature | Description |
|-----|-----------|-------------|
| `AppendPortIfNeeded` | `func AppendPortIfNeeded(addr string, port int32) string` | Append port to address if missing |

### 9.9 `pkg/util/gen` — Code Generation Utilities

**Import:** `github.com/onexstack/onexstack/pkg/util/gen`

| API | Signature | Description |
|-----|-----------|-------------|
| `OutDir` | `func OutDir(path string) (string, error)` | Resolve output directory for generated code |

### 9.10 `pkg/util/lint` — Custom Go Linters

**Import:** `github.com/onexstack/onexstack/pkg/util/lint`

Not intended for import; provides custom `go/analysis` analyzers for use with `go vet` and other linter tools.

| Sub-package | Import | Description |
|-------------|--------|-------------|
| `util/lint/errcheck` | `.../pkg/util/lint/errcheck` | `var Analyzer` — errcheck analyzer (bazel build only) |
| `util/lint/hash` | `.../pkg/util/lint/hash` | `var Analyzer` — checks for correct use of `hash.Hash` |
| `util/lint/shadow` | `.../pkg/util/lint/shadow` | `var Analyzer` — shadow variable detection (bazel build only) |
| `util/lint/pass` | `.../pkg/util/lint/pass` | `FindContainingFile`, `HasNolintComment` — analysis pass helpers |

---

## Package Dependency Map (Key Relationships)

```
app ───────────────────────────────────────────► log, options, version
server ────────────────────────────────────────► options, log
  ├── grpc_server ─────────────────────────────► options
  ├── kratos_server ───────────────────────────► options, log
  └── reverse_proxy_server ────────────────────► options

core ──────────────────────────────────────────► errorsx
options ────────────────────────────────────────► (standalone — flag/config structs)
errorsx ────────────────────────────────────────► (standalone — error types)

authn ──────────────────────────────────────────► (authn/jwt)
  └── jwt ──────────────────────────────────────► jwt store (redis)

authz ──────────────────────────────────────────► gorm, casbin

db ─────────────────────────────────────────────► gorm, redis
store ──────────────────────────────────────────► gorm, store/where
store/where ────────────────────────────────────► gorm
distlock ───────────────────────────────────────► logger, various backends

log ────────────────────────────────────────────► zap, kratos/log, gorm/logger
logger ─────────────────────────────────────────► log/slog, log, various adapters

token ──────────────────────────────────────────► jwt
flux ───────────────────────────────────────────► ristretto
id ─────────────────────────────────────────────► sonyflake
rid ────────────────────────────────────────────► (standalone, OS machine-id)
i18n ───────────────────────────────────────────► go-i18n
ptr ────────────────────────────────────────────► (standalone — generics)
validation ─────────────────────────────────────► govalidator
binding ────────────────────────────────────────► gin
version ────────────────────────────────────────► (standalone — debug.BuildInfo)

middleware/gin ─────────────────────────────────► gin, otel
middleware/grpc ────────────────────────────────► grpc, otel

otelslog ───────────────────────────────────────► otel/sdk/log, log/slog
otel/emptyexporter ─────────────────────────────► otel/sdk/trace

watch ──────────────────────────────────────────► gorm, cron, ratelimit
  ├── watch/registry ───────────────────────────► cron
  ├── watch/manager ────────────────────────────► cron
  └── watch/initializer ────────────────────────► watch/registry, watch/manager

cli/cli ────────────────────────────────────────► (standalone — filesystem paths)
cli/genericclioptions ──────────────────────────► cobra, http
polaris ────────────────────────────────────────► polarismesh, resty

util/retry ─────────────────────────────────────► k8s.io/apimachinery
util/file ──────────────────────────────────────► h2non/filetype
util/strings ───────────────────────────────────► (standalone)
util/reflect ───────────────────────────────────► gorm, yaml
util/ip ────────────────────────────────────────► (standalone)
util/version ───────────────────────────────────► k8s.io/apimachinery
util/pagination ────────────────────────────────► (standalone)
util/controller ────────────────────────────────► (standalone)
util/gen ───────────────────────────────────────► (standalone)
```

---

## Usage Patterns

### Bootstrapping an Application

```go
// cmd/myapp/main.go
import (
    "github.com/onexstack/onexstack/pkg/app"
    "github.com/onexstack/onexstack/pkg/options"
    "github.com/onexstack/onexstack/pkg/log"
)

func main() {
    opts := options.NewMySQLOptions()
    app.NewApp("myapp", "My Application",
        app.WithOptions(opts),
        app.WithRunFunc(func() error {
            // init and run
            return nil
        }),
        app.WithDefaultHealthCheckFunc(),
        app.WithWatchConfig(),
    ).Run()
}
```

### Structured Error Handling

```go
import "github.com/onexstack/onexstack/pkg/errorsx"

// Create and enrich errors
err := errorsx.ErrNotFound.WithMessage("user %s not found", userID).KV("user_id", userID)

// Write in Gin handlers
import "github.com/onexstack/onexstack/pkg/core"
core.WriteResponse(c, nil, err)
```

### Generic Request Handling

```go
import "github.com/onexstack/onexstack/pkg/core"

type CreateUserReq struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateUserReq) Default() { /* set defaults */ }

// In Gin handler:
core.HandleJSONRequest(c, func(ctx context.Context, req *CreateUserReq) (*CreateUserResp, error) {
    // req is already bound and validated
    return createUser(ctx, req)
})
```

### Token Management

```go
import "github.com/onexstack/onexstack/pkg/token"

// Initialize
token.Init("my-secret-key",
    token.WithExpiration(24*time.Hour),
    token.WithCommonSkipPaths(),
)

// Sign
tokenStr, expiresAt, _ := token.Sign(userID)

// Parse from HTTP request
identity, _ := token.ParseRequest(ctx)
```

### Using Flux Cache

```go
import "github.com/onexstack/onexstack/pkg/flux"

type UserLoader struct{}

func (l *UserLoader) Load(ctx context.Context, key string) (*User, error) {
    return fetchUserFromDB(ctx, key)
}

cache, _ := flux.NewFlux[string, *User](
    flux.WithTTL[string, *User](5 * time.Minute),
    flux.WithLoader[string, *User](&UserLoader{}),
    flux.WithAsyncRefresh[string, *User](1 * time.Minute),
)
defer cache.Close()

user, _ := cache.Get(ctx, "user-123")
```

### Distributed Locking

```go
import "github.com/onexstack/onexstack/pkg/distlock"

locker := distlock.NewRedisLocker(redisClient,
    distlock.WithLockName("my-lock"),
    distlock.WithLockTimeout(30*time.Second),
)
locker.Lock(ctx)
defer locker.Unlock(ctx)
// ... critical section ...
```

### GORM Where Conditions

```go
import (
    "github.com/onexstack/onexstack/pkg/store"
    "github.com/onexstack/onexstack/pkg/store/where"
)

count, users, err := userStore.List(ctx,
    where.P(1, 20).F("status", "active").Q("name LIKE ?", "%test%"))
```

### i18n

```go
import "github.com/onexstack/onexstack/pkg/i18n"

var I = i18n.New(
    i18n.WithLanguage(language.English),
    i18n.WithFile("locales/en.json"),
)

// In handler:
ctx = i18n.WithContext(ctx, I.Select(language.Chinese))
// Later:
localizer := i18n.FromContext(ctx)
msg := localizer.T("welcome_message")
```
