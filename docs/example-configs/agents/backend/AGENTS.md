# Backend Agent Instructions

## Architecture

### Dependency Injection Pattern

All services are wired through `common.App` (the central DI container). Service interfaces are defined in `common/services.go`. The pattern:

- **Domain packages** (`auth/`, `core/`) — Pure logic, NO dependency on `*common.App`. Some packages are more extensible (e.g `jobs/`, `messengers/`) and so have persistent engines instantiated with app instances, e.g so a job can use any service in the app
- **Service wrappers** (`services/`) — Hold `*common.App`, delegate to domain packages, implement `common.*Service` interfaces
- **Bootstrap** (`main.go`) — Constructs everything in dependency order, assigns to `app` fields

When adding a new service: define the interface in `common/services.go`, implement domain logic in a new package, wire it in `services/`, add to `common.App`. You will need to update `testhelpers.NewApp` and possibly some unit tests that incidentally call the new service (otherwise they will nil panic).

### Error System (Critical Convention)

This is the most distinctive pattern. **Every function should return `common.WrappedError`, never bare `error`.**

```go
// 1. Declare error wrapper at package level
var ErrWrapperMyOp = common.NewErrorWrapper(common.ErrTypeAuth, ErrTypeMyOp)

// 2. Use it to wrap standard errors
return ErrWrapperMyOp.Wrap(stdErr)

// 3. Dynamic wrappers auto-detect error types (see common/premadeErrors.go)
var ErrWrapperDatabase = common.NewDynamicErrorWrapper(func(err error) WrappedError { ... })
```

Key files: `common/errors.go` (type system), `common/premadeErrors.go` (sentinels + dynamic wrappers), `common/retries.go` (reads retry config from errors).

**Never return `fmt.Errorf(...)` or bare `errors.New(...)`** — always use the wrapper pattern.

### Database (ent ORM)

Schema is defined in `ent/schema/` (`.go` files). Generated code lives alongside. Run codegen from `backend/`:

```bash
go generate ./ent
```

- **Always check `ent/schema/`** for the source of truth - never analyse generated code in `ent/` directly.
- Transactions: In production code and integration tests use `dbcommon.WithReadTx` / `dbcommon.WithWriteTx` / `dbcommon.WithReadWriteTx` — these extract `*ent.Tx` from `*common.App`, inject it into the context, and auto-retry on temporary errors (e.g. SQLite busy). In unit tests use `testcommon.StartWriteTx(t, db)` instead — it returns a `*ent.Tx` with no retry logic so you can assert there are no temporary errors.
- **Encrypted fields**: Custom `ValueScanner` in schema encrypts at DB driver level. Key slots defined via HKDF from `BASE_ENCRYPTION_KEY`
- **Migrations**: goose-based. Scripts in `scripts/migrations/`. Single global mutex (`globals.MigrateMu`) serializes all migrations

### HTTP Layer (Gin)

- Router setup: `services/server.go` → `server/endpoints/` → `server/endpoints/v1/`
- **Auth schemes**: `Bearer <sessionToken>` (users) and `AdminCode <adminCode>` (admins)
- **Middleware chain**: Logging → Timeout → RateLimit → Error → SessionAuth → AdminProtected
- **Endpoint handlers**: Return `*servercommon.Error` — see `server/servercommon/handlers.go` for the `NewHandler` pattern
- Error responses go through `ginCtx.Error(serverErr)` + `ginCtx.Abort()`
- **Data flow**: `server/endpoints/` → `services/` → `core/` → `ent/` (via `dbcommon` transaction helpers)
- Pass `*servercommon.ServerApp` (embeds `*common.App`) into endpoint factory functions. Never instantiate services directly inside handlers.

### Logging

- Uses `log/slog` with custom `loggers.Handler` (async channel-based, batched DB writes)
- Always use a logger instance rather than the top level slog methods, get this from:
- - `app.Logger`
- - An explicit logger argument (if the function is part of the service, the service will provide it)
- - `common.GetLogger(ctx, service)` for some niche cases where you want to depend on another service's logger or use an override in the context (e.g dbcommon does this)
- Tests have `PANIC_ON_ERROR` env var enabled by default so that logged errors cause tests to fail

### Testing

- Use `testify/require` for assertions
- Integration tests: `testhelpers.NewApp(t, options)` creates a fully wired app with mock messengers
- `testcommon/DefaultEnv()` provides test configuration

## Key Packages Quick Reference

| Package | Role |
|---------|------|
| `common/` | Shared types, error system, retries, crypto helpers, the `App` struct |
| `auth/` | WebAuthn login/registration/sessions (pure logic) |
| `core/` | Business logic: stashes, users, hashing (Argon2id), encryption (AES-256-GCM) |
| `ent/` | Database schema + generated ORM code |
| `entps/` | SQLite driver patch (pragmas, vendored) |
| `services/` | Service layer wiring domain packages into `*common.App` |
| `server/` | HTTP routes, middleware, endpoint handlers |
| `jobs/` | Async job engine with retries and weight limits |
| `schedulers/` | Periodic task engine with persistent intervals |
| `messengers/` | Notification dispatch (Discord, SMTP, etc.) |
| `ratelimiting/` | In-memory rate limiter |
| `keyvalue/` | Persistent typed key-value store |
| `tempkeyvalue/` | In-memory TTL store (WebAuthn session data) |
| `loggers/` | Custom slog handler with async DB writes |

## Adding Environment Variables

Env vars flow through four locations. When adding one, update all four and ensure they're all grouped in the same way as in the type definition of `common.Env` in `common/services.go`:

1. **`common.Env` struct** — Add the field. `//exhaustruct:enforce` ensures every field is populated.
2. **`services/env.go`** — Load from the environment in `LoadEnvironmentVariables()`. Group with related vars (deployment config, job config, admin auth, invites, sessions/timing, hashing, encryption, logging, messengers). Some vars aggregate into sub-structs (e.g., `PASSWORD_HASH_SETTINGS` wraps `PASSWORD_HASH_TIME`/`_MEMORY`/`_THREADS`). Prefer `Require*` over `Optional*` — only use `Optional*` when there's a safe default (e.g disabling a feature like a specific messenger).
3. **`services/env.go` `ValidateEnvironmentVariables()`** — Add cross-field validation if needed (e.g., "AUTH_CODE_VALID_FOR must exceed UNLOCK_TIME").
4. **`common/testcommon/env.go` `DefaultEnv()`** — Add a test-safe default. Use intentionally weak values (e.g., hash settings with `Time: 1, Memory: 1024, Threads: 1`). Use realistic time values, these defaults can be overridden if it's impractical for a test to use mocked time.

## Schema Changes

1. Edit the `.go` files in `ent/schema/` — they are the source of truth
2. Run `go generate ./ent` from `backend/`
3. Never edit generated code in `ent/` directly

Encrypted fields use `ValueScanner(EncryptedField[T]{KeyName: "slot_name"})`. Key slots are derived from `BASE_ENCRYPTION_KEY` via HKDF. See `ent/schema/schema.go` for `Init()` and available slots.

## Utility Package Boundaries

| Package | For | Used By |
|---------|-----|---------|
| `common/` | Shared types, interfaces, error system, helpers with no domain-specific knowledge | Every package except ent/**, including tests |
| `common/dbcommon/` | Transaction helpers (`WithReadTx`, `WithWriteTx`), DB error wrappers | Every package except ent/**, including many tests |
| `server/servercommon/` | HTTP-specific: `ServerApp`, `*Error`, handler wrappers, auth parsing | `server/` endpoints and middleware |
| `common/testcommon/` | Test fixtures: `DefaultEnv()`, `CreateDB()`, HTTP request builders, assertion helpers | Test files across all packages |
| `testhelpers/` | Full integration test harness: `NewApp()` with mock messengers | Endpoint tests, high-level integration tests |
| `common/globals/` | `MigrateMu` — single global mutex for migration serialization | Avoid unless absolutely necessary |

## Build & Test Commands

Do not offer to build or run the program itself, instead check for workspace problems using the tool and optionally ask the user to manually test it.

Use the runTests tool instead of `go test`.

Some linters are slow and don't run automatically. Before concluding a task is complete, run the full linters with:

```bash
# Lint
bash scripts/lint.sh
```

## Anti-Patterns to Avoid

- ❌ Using `error` instead of `common.WrappedError` in function signatures
- ❌ `fmt.Errorf` for errors that should carry categories/retry config
- ❌ Direct SQL queries - always use the ent query builder
- ❌ Skipping transaction helpers (`dbcommon.WithReadTx`/`WithWriteTx`/`WithReadWriteTx`)
- - ❌ Returning `struct{}` from transaction callbacks. Use `WithWriteTx` if you don't need to return any data
- ❌ Hardcoding credentials or keys
- ❌ Adding CGo dependencies — the project targets pure-Go SQLite
- ❌ Analysing generated code in `ent/` directly — always check `ent/schema/` for the source of truth
- ❌ Running `gofmt` — the project uses `golangci-lint` for formatting (run on save). Prefer tabs over spaces.
