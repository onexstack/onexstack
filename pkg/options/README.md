# options

Package `options` 提供 onexstack 通用 API server 的公共命令行 flag 与配置项集合。它只承载配置与 flag 绑定逻辑，供 `pkg/app`、`pkg/server` 等多个组件复用。

## 接口约定

所有 option 结构体实现 `IOptions` 接口：

```go
type IOptions interface {
    // Validate 校验配置项，返回全部错误（而非首个错误）。
    Validate() []error

    // AddFlags 将配置项注册为 flag，fullPrefix 为完整前缀，如 "onex.otel"。
    AddFlags(fs *pflag.FlagSet, fullPrefix string)
}
```

`ServerOptions` 同时满足 `pkg/app.FlagSetOptions` 要求的形状（`AddFlags(*pflag.FlagSet)` / `Validate() []error`），可直接传给 `app.WithOptions`。

## 主要 option 类型

- **服务监听**：`SlogOptions`、`InsecureServingOptions`、`SecureServingOptions`、`GRPCOptions`、`HealthOptions`
- **数据库**：`MySQLOptions`、`PostgreSQLOptions`、`SQLiteOptions`、`RedisOptions`、`MongoOptions`
- **中间件/基础设施**：`KafkaOptions`、`EtcdOptions`、`TLSOptions`、`JWTOptions`、`ConsulOptions`、`PolarisOptions`
- **可观测性**：`OTelOptions`、`ObservabilityOptions`、`MetricsOptions`、`JaegerOptions`、`LogOptions`
- **其它**：`RestyOptions`（HTTP 客户端）、`FluxOptions`（缓存）、`WatchOptions`（watch server）、`AWSSecretManagerOptions`

## 基本用法

单个 option：

```go
opts := options.NewMySQLOptions()
fs := pflag.NewFlagSet("server", pflag.ContinueOnError)
opts.AddFlags(fs, "mysql")     // 注册 --mysql.addr 等 flag
if errs := opts.Validate(); len(errs) > 0 {
    // 处理校验错误
}
```

服务端组合入口：

```go
opts := options.NewServerOptions()   // 聚合 slog/insecure/secure/grpc/health
opts.AddFlags(fs)                    // 按 log/insecure/secure/grpc/health 前缀注册
app.NewApp("myserver", "My Server",
    app.WithOptions(opts),
    app.WithSlogOptions(opts.SlogOptions),
    // ...
).Run()
```
