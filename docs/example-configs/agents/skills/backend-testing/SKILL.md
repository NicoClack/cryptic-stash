---
name: backend-testing
description: 'Write Go tests for the Cryptic Stash backend. Use when: writing unit tests, integration tests, endpoint tests, test helpers, or adding test assertions. Covers test DB setup, common.App wiring, empty services vs mocks, transaction usage, and HTTP test utilities.'
---

# Backend Testing

## Core Rule

**Tests always use a real SQLite database** via `testcommon.CreateDB(t)`. The database is never mocked.

## Test Tiers

### 1. Minimal Unit Tests

Construct `*common.App` with only the services your code needs:

```go
app := &common.App{
    Database: testcommon.CreateDB(t),
    Env:      testcommon.DefaultEnv(),
    Clock:    clockwork.NewFakeClock(), // Note: use a real clock unless you need to control time
    Logger:   testcommon.NewTestLogger(),
}
```

Then create your package's types directly (e.g., `jobs.NewEngine(registry)`, `ratelimiting.NewLimiter(app)`).

### 2. Empty Services (Not Mocks)

When `common.App` has fields your code doesn't use, but some dependency chain might access them as a side effect, fill them with empty services from `common/testcommon/mocks/` to prevent nil pointer panics:

```go
app := &common.App{
    // ... services you need ...
    Core:             mocks.NewEmptyCoreService(),
    TempKeyValue:     mocks.NewEmptyTempKeyValueService(),
    TwoFactorActions: mocks.NewEmptyTwoFactorActionService(),
    RateLimiter:      mocks.NewEmptyRateLimiterService(),
}
```

Empty services (`NewEmptyXxxService()`) implement the interface but return zero values. They are **not** mocks — they exist only to satisfy the `App` struct, not to assert behaviour.

### 3. Real Mocks (and optionally some empty)

Use when you need to assert that a service was called with specific arguments. The only real mock currently is `mocks.NewShutdownService()`:

```go
shutdown := mocks.NewShutdownService()
app.ShutdownService = shutdown
// ... run code ...
shutdown.AssertCalled(t, "expected reason")
shutdown.AssertNotCalled(t)
```


If you need to assert behaviour of a real service, first consider whether an endpoint integration test (which can use `testhelpers.NewApp`) would be clearer.

### 4. Full Integration Tests (currently just for endpoints)

`testhelpers.NewApp(t, options)` wires a complete `*common.App` with all services, mock messengers, and cleanup:

```go
app := testhelpers.NewApp(t, &testhelpers.AppOptions{
    Clock: clock,  // optional overrides
    Env:   env,
})
// app.MockMessenger has sent messages for assertions
// t.Cleanup calls app.Shutdown() automatically
```

`testhelpers.App` embeds `*common.App` plus:
- `MockMessenger` — records sent messages (never sends real ones)
- `TestDatabase` — access to the underlying test DB

**Note**: `services/` package tests do NOT use `testhelpers.NewApp` (would create a circular dependency). They construct `App` manually.

## Test Data Setup

### Simple Setup: Use `.SaveX`

For test data where failures are unlikely, use Ent's `.X` methods for conciseness:

```go
dbClient := app.Database.Client() // Define this at the start of the test, along with other things like the app, clock, env etc.
passkeyOb := dbClient.Passkey.Create().
    SetUserID(userOb.ID).
    SetName("test-passkey").
    SetCredentialID(credentialID).
    // ...
    SaveX(t.Context())
```

`.SaveX` panics on error — acceptable for test setup where errors indicate a test bug, not a code bug.

### When to Use Transactions

**Integration tests and production code** use the `dbcommon` tx helpers — `dbcommon.WithWriteTx` / `dbcommon.WithReadWriteTx` / `dbcommon.WithReadTx` — which auto-retry on temporary errors (e.g. SQLite busy). Use them when:

1. **The method you're calling requires a transaction** — e.g., `app.Auth.CreateSession` takes `*ent.Tx` and returns errors rather than panicking
2. **Goroutines may interfere** — transactions auto-retry on SQLite busy errors

```go
// You could use WithReadTx if no writes are necessary, but they are here
sessionToken, stdErr := dbcommon.WithReadWriteTx(
    t.Context(), app.Database,
    func(tx *ent.Tx, ctx context.Context) ([]byte, error) {
        _, token, stdErr := app.Auth.CreateSession(...)
        return token, stdErr
    },
)
require.NoError(t, stdErr)
```

Use `WithWriteTx` (no return value) when you just need the transaction semantics:

```go
stdErr := dbcommon.WithWriteTx(t.Context(), app.Database,
    func(tx *ent.Tx, ctx context.Context) error {
        // setup that needs a transaction
        return nil
    },
)
require.NoError(t, stdErr)
```

**Unit tests** use `testcommon.StartWriteTx(t, db)` instead. It starts a transaction with **no retry logic**, so if the code under test hits a temporary error the test fails. That's the point — you want to assert there are no temporary errors, not have them silently retried:

```go
db := testcommon.CreateDB(t)
tx := testcommon.StartWriteTx(t, db)
wrappedErr := auth.RenamePasskey(
    passkeyOb.ID, "new-name", actor,
    tx, t.Context(),
)
require.NoError(t, wrappedErr)
// ...
require.NoError(t, tx.Commit())
// ^ You should commit the transaction before asserting DB state unless you need some atomicity with another operation
```

Every started transaction must be terminated deliberately: commit before asserting persisted state (query via `dbClient`, not the same `tx`), or roll back when the code under test errored or panicked and nothing should persist (this also avoids firing any `OnCommit` hooks registered mid-flight).

## HTTP Test Helpers

From `common/testcommon/requests.go`:

```go
// POST with JSON body
respRecorder := testcommon.Post(
    t, app.Server,
    "/api/v1/path/",
    payload,
)
// GET
respRecorder := testcommon.Get(
    t, app.Server,
    "/api/v1/path/",
)
// With auth
respRecorder := testcommon.Post(
    t, app.Server,
    "/api/v1/path/",
    payload,
    testcommon.WithBearerToken(token),
)
// Assert response, preferred when practical
testcommon.AssertJSONResponse(
  t, respRecorder,
  http.StatusOK,
  expectedStruct{

  },
)
// Alternatively, you sometimes need to do something like this:
require.Equal(t, http.StatusOK, respRecorder.Code)
var inviteResp invites.GetInviteResponse
stdErr := json.Unmarshal(respRecorder.Body.Bytes(), &inviteResp)
require.NoError(t, stdErr)
require.Equal(t, email, inviteResp.Email)
require.Equal(t, expiresAt, inviteResp.ExpiresAt)
```

## Test-Specific Service Wrappers

For packages like `loggers/`, create a test-local wrapper in `*_test.go` that wraps the real service with assertion helpers:

```go
// loggers/loggers_test.go (package loggers_test)
type Logger struct {
    *slog.Logger
    Handler *loggers.Handler
}
func NewLogger(app *common.App) *Logger { ... }
func (l *Logger) AssertWritten(t *testing.T) { ... }
```

## Package Naming

- **External test packages** (`package foo_test`) — the norm. Import your own package like an external consumer. You are unlikely to encounter circular dependencies this way.
- **Internal test packages** (`package foo`) — rare. Only used when tests need access to unexported fields. Avoid importing packages that the package doesn't already import as this can create circular dependencies. The test filename will need to be `foo_internal_test.go` due to the `testpackage` linter.

## Concurrent Tests

- Use `t.Parallel()` extensively, there is very little global state. It should have a blank line afterwards unless the test is very short.
- Use channels to synchronize with background goroutines
- Use `clockwork.NewFakeClock()` to control time in tests with timing-dependent logic. Note that some external packages don't support this, so you may need to reduce some timings in your test's environment variables and use time.Sleep().
