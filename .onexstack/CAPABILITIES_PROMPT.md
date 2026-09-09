# CAPABILITIES.md Regeneration Prompt

> **Purpose:** When this file is read by an AI coding assistant, it should regenerate the `CAPABILITIES.md` file at the project root with a complete, accurate catalog of all reusable packages in this Go shared library.

## Instructions

You are tasked with regenerating the `CAPABILITIES.md` file for the `github.com/onexstack/onexstack` Go module. This is a shared Go package library that provides infrastructure components for building microservices. The `CAPABILITIES.md` file serves as a machine-readable catalog for AI assistants to discover and correctly reuse existing code.

### Step 1: Explore All Packages

Recursively explore all Go source files under `pkg/`. For each package (including sub-packages):

- Read every `.go` file (skip `*_test.go` files)
- Identify every **exported** symbol: types, interfaces, functions, methods, constants, and variables
- Record the full signature of each exported symbol (parameter names/types, return values)
- Note the package's purpose and the key dependencies it wraps

**Important:** Some packages have deeply nested sub-packages (e.g., `pkg/logger/slog/gorm/`, `pkg/authn/jwt/store/redis/`, `pkg/watch/manager/`). Read ALL of them.

### Step 2: Read `go.mod` for Key Dependencies

Read `go.mod` to understand the technology stack. Key dependencies to note include:

- `github.com/gin-gonic/gin` — HTTP web framework
- `github.com/spf13/cobra` + `github.com/spf13/pflag` + `github.com/spf13/viper` — CLI and config
- `gorm.io/gorm` — ORM
- `github.com/redis/go-redis/v9` — Redis client
- `github.com/golang-jwt/jwt/v4` + `jwt/v5` — JWT handling
- `github.com/casbin/casbin/v2` — RBAC authorization
- `go.opentelemetry.io/otel` — Observability
- `go.uber.org/zap` — Logging
- `github.com/nicksnyder/go-i18n/v2` — Internationalization
- `github.com/go-kratos/kratos/v2` — Microservice framework
- `google.golang.org/grpc` — gRPC
- `github.com/polarismesh/polaris-go` — Service mesh
- `github.com/sony/sonyflake` — ID generation
- `github.com/robfig/cron/v3` — Cron scheduling
- `github.com/google/wire` — Dependency injection

### Step 3: Organize Capabilities by Category

Group the packages into the following categories (match the existing CAPABILITIES.md structure):

1. **Application Framework** — `app`, `server`, `options`, `watch` (and their sub-packages)
2. **Authentication & Authorization** — `authn`, `authn/jwt`, `authn/jwt/store/redis`, `authz`, `token`
3. **Data & Storage** — `db`, `store`, `store/where`, `store/registry`, `distlock`
4. **Request Handling & Core Utilities** — `core`, `errorsx`, `binding`, `validation`, `ptr`, `id`, `rid`, `version`, `i18n`, `flux`
5. **Logging** — `log`, `logger` (and all sub-packages)
6. **HTTP/gRPC Middleware** — `middleware/gin`, `middleware/grpc`
7. **Observability (OpenTelemetry)** — `otelslog`, `otel/emptyexporter`, `otel/exporter/empty`
8. **CLI Utilities** — `cli/cli`, `cli/genericclioptions`, `polaris`
9. **Utility Packages** — `util/retry`, `util/file`, `util/strings`, `util/reflect`, `util/ip`, `util/version`, `util/pagination`, `util/controller`, `util/gen`

### Step 4: Write CAPABILITIES.md

Write the file at the project root (`CAPABILITIES.md`) with the following requirements:

#### Format Requirements

- **Import path** for every package (e.g., `github.com/onexstack/onexstack/pkg/app`)
- **Table of Contents** with numbered sections matching the categories above
- **API tables** for each package: columns are API name, full signature, and a one-line description
- **Group related APIs**: put constructors first, then methods on the type, then options/helpers
- **Include ALL exported symbols** — do not omit anything. This is critical for AI reuse.
- **Usage pattern code examples** at the bottom showing common patterns (app bootstrap, error handling, request handling, token management, flux cache, distributed locking, GORM where conditions, i18n)
- **Package dependency map** showing key relationships between packages

#### Content Quality Requirements

- Be **exhaustive, not selective** — the AI needs to see everything available
- Use **precise Go type signatures** — include generic type parameters, parameter names, return types
- **Highlight interfaces** — they define the contracts consumers need to implement
- **Note design patterns** — functional options, generic stores, adapter patterns
- **Include predefined errors and constants** — they're part of the public API
- **Describe each package's role** — a one-sentence summary at the beginning of each section

#### Structure Reference

Each package section should follow this pattern:

```markdown
### X.Y `pkg/<name>` — <Short Description>

**Import:** `github.com/onexstack/onexstack/pkg/<name>`

<One-sentence purpose statement.>

| API | Signature | Description |
|-----|-----------|-------------|
| `FuncName` | `func FuncName(params) returns` | What it does |
```

For structs with methods, list the struct first, then its methods:

```markdown
| `TypeName` | `struct{ ExportedField type; ... }` | What the type represents |
| `(*TypeName).Method` | `func (r *TypeName) Method(params) returns` | What the method does |
```

#### Important Notes

- Do NOT skip any exported symbol — completeness is more important than brevity
- If a sub-package is only for internal use (testdata, examples), note it but don't detail its API
- Verify import paths match the module name in `go.mod` (`github.com/onexstack/onexstack`)
- The file should be self-contained and readable both by humans and AI assistants
- Use the exact signature from the source code — do not paraphrase or simplify
