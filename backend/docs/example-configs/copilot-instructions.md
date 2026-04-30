## Cryptic Stash

A web app for securely storing 2-factor recovery codes. Users upload files that are encrypted using a key derived from their password via Argon2id. In the event of account lockout, the user can log in and download their file after a waiting period. If an attacker attempts the same, the user is notified and can block the attempt.

Monorepo: `backend/` (Go + Gin + Ent ORM + SQLite) and `frontend/` (SvelteKit 5 + TypeScript + shadcn-svelte + Tailwind CSS v4).

---

## Agent Behaviour

- Prefer tools over CLI for reading/searching. Use `ripgrep`/`grep`/`cat` only when it would significantly reduce unnecessary context or for bulk actions.
- Before completing a request, use the language server to check for syntax and lint errors. Skip this step only if the change is intentionally incomplete and further input is needed.
- Comments should explain WHY, not WHAT. If you feel the need to explain what code does, extract it into a well-named function instead.
- Do not run `gofmt` — use `golangci-lint` for formatting. The formatter will be run on save.
- Prefer tabs over spaces in Go. Do not add extra space indentation on top of existing tabs.
- Check Ent schemas (`ent/schema/`) rather than analysing generated code (`ent/`) where possible.

---

## Backend (Go)

### Package Architecture

| Layer | Package(s) | Purpose |
|---|---|---|
| Shared utilities | `common/` | Error system, crypto, retries, slices, strings, types, logging, env helpers |
| Business logic | `core/` | Encryption, hashing, user/stash operations, download sessions |
| Service wiring | `services/` | Factory functions that construct and wire all services together |
| HTTP server | `server/` | Gin router, middleware, servercommon utilities, endpoints |
| Data access | `ent/` (generated) | Ent ORM client. Modify `ent/schema/` only, never edit generated code |
| Background work | `jobs/`, `schedulers/` | Queued jobs (DB-backed, retriable) vs. scheduled tasks (cron-like) |
| Storage | `keyvalue/`, `tempkeyvalue/` | Persistent DB-backed KV store vs. in-memory KV with expiry |
| Messaging | `messengers/` | Multi-messenger abstraction (Discord, SMTP, SMTP2GO, develop) |
| Auth extras | `twofactoractions/`, `ratelimiting/` | 2FA action registry, rate limiter |

Data flow: `server/endpoints/` → `services/` → `core/` → `ent/` (via `dbcommon` transaction helpers). All layers share `common/`.

### Dependency Injection and endpoints

All services live on `common.App`. Pass `*servercommon.ServerApp` (which embeds `*common.App`) into endpoint factory functions. Never instantiate services directly inside handlers.

```go
// Endpoint factory pattern
func GetAuthorizationCode(app *servercommon.ServerApp) gin.HandlerFunc {
    clock := app.Clock
    // ^ Can be worth creating an alias if frequently referenced
    // But assume app is immutable once initialised
    return servercommon.NewHandler(func(ginCtx *gin.Context) error {
        serverErr := ParseThing()
        if serverErr != nil {
            return serverErr // Might have a status and details to send to the client
        }
        stdErr := DoThing()
        if stdErr != nil {
            return stdErr // Send 500
        }
        ginCtx.JSON(http.StatusOK, MyResponse{...})
        return nil
    })
}
```

These are registered in `server/endpoints/.../<package>/<package>.go`:

```go
// server/endpoints/v1/users/users.go
func ConfigureEndpoints(group *servercommon.Group) {
    router.POST("/get-authorization-code/", GetAuthorizationCode(app)) // getAuthorizationCode.go
    router.GET("/", ListUsers(app)) // listUsers.go (example endpoint)
    stashes.ConfigureEndpoints(group.Group("/stashes")) // child package
}
```

### Naming Conventions

**Go variable names — use full words, not abbreviations, except:**

| Short form | Type |
|---|---|
| `ctx` | `context.Context` |
| `ginCtx` | `*gin.Context` |
| `stdErr` | Standard `error` interface |
| `wrappedErr` | `common.WrappedError` interface |
| `serverErr` | `*servercommon.Error` |
| `commErr` | `*common.Error` |

**Entity instances** (Ent ORM results) — suffix with `Ob` (singular) or `Obs` (plural) to avoid collisions with their package names:
- `userOb *ent.User`, `stashOb *ent.Stash`, `downloadSessionOb *ent.DownloadSession`
- `userObs []*ent.User`, `downloadSessionObs []*ent.DownloadSession`
- Do NOT apply this suffix to plain JSON response structs or service instances.

**HTTP external responses:**
- `resp` — `*http.Response`
- `respBytes` — raw `[]byte` from response body
- `respBody` — decoded struct from response body

**File names:** `camelCase.go` (e.g. `getAuthorizationCode.go`, `premadeErrors.go`).

**Request/response types:** `EndpointNamePayload` / `EndpointNameResponse` (e.g. `func GetAuthorizationCode` -> `GetAuthorizationCodePayload`, `func Download` -> `DownloadResponse`).

### Error System

Errors flow through three layers:

```
stdlib/3rd-party error (std error interface)
  └─ common.WrappedError  (adds categories, retry config, debug values)
       └─ servercommon.Error  (adds HTTP status, error details, logging flag)
```

**Categories** are hierarchical strings defined as constants in `common/errors.go` (general: `ErrTypeDatabase`, `ErrTypeCore`, etc.) and in each package's `errors.go`. Lower-level categories are more specific. It's common for some error wrappers like `ErrWrapperDatabase` to be defined in multiple packages: the error is wrapped by `common.ErrWrapperDatabase` and then the package adds its own category.

**Wrapping pattern:**
```go
// In core/errors.go — define wrappers once per operation
var ErrWrapperSendActiveDownloadSessionReminders = common.NewErrorWrapper(
    common.ErrTypeCore,
    ErrTypeSendActiveDownloadSessionReminders,
)

// In usage
return ErrWrapperSendActiveDownloadSessionReminders.Wrap(
    common.ErrWrapperDatabase.Wrap(stdErr),
)
```

**HTTP error building:**
```go
// Cloning pre-made errors is preferred. Use the NewXXX functions or call ErrXXX.Clone() / ErrXXX.CloneAsWrappedError()
return servercommon.NewUnauthorizedError()

// Building ad-hoc
return servercommon.NewError(stdErr).
    SetStatus(http.StatusBadRequest).
    AddDetail(servercommon.ErrorDetail{
        Message: "auth code is not valid base64",
        Code:    "MALFORMED_AUTH_CODE",
    }).
    DisableLogging()  // for 4xx client errors

// Shortcuts for common patterns
return servercommon.SendUnauthorizedIfNotFound(stdErr)
return servercommon.Send404IfNotFound(stdErr)
```

Errors are returned from handlers and collected by the error middleware, which writes the JSON response. Do not call `ginCtx.JSON` for error paths — just `return serverErr`.

### Handler Pattern

```go
func MyEndpoint(app *servercommon.ServerApp) gin.HandlerFunc {
    // Capture any app values needed here (avoids repeated map lookups)
    return servercommon.NewHandler(func(ginCtx *gin.Context) error {
        // 1. Parse and validate body
        body := MyPayload{}
        if ctxErr := servercommon.ParseBody(&body, ginCtx); ctxErr != nil {
            return ctxErr
        }

        // 2. Business logic via transactions
        result, stdErr := dbcommon.WithReadWriteTx(
            ginCtx.Request.Context(), app.Database,
            func(tx *ent.Tx, ctx context.Context) (*MyResponse, error) {
                // ... ent queries ...
                return &MyResponse{...}, nil
            },
        )
        if stdErr != nil {
            return stdErr
        }

        // 3. Write success response (no error return after this point)
        ginCtx.JSON(http.StatusOK, result)
        return nil
    })
}
```

### Transaction Helpers (`common/dbcommon`)

| Function | Use when |
|---|---|
| `dbcommon.WithReadTx(ctx, db, fn)` | Read-only queries; returns a value |
| `dbcommon.WithWriteTx(ctx, db, fn)` | Write-only; only returns error value |
| `dbcommon.WithReadWriteTx(ctx, db, fn)` | Mixed read-write; returns a value |

- Transactions are injected into `ctx` and retrieved with `ent.TxFromContext(ctx)` in services (returning `ErrNoTxInContext` if missing). Other functions should generally take an explicit `*ent.Tx` argument to reduce boilerplate, especially for private functions.
- All helpers include automatic retry logic for transient SQLite errors. Other errors can also be retried if they are wrapped and have retries configured. On retry, the transaction is rolled back and the whole callback is rerun. For this reason and due to the limitations of SQLite transactions without WAL, computations and network calls should almost always be moved outside the transaction, even if this requires re-checking data.

### Ent ORM Usage

- Define and modify entities only in `ent/schema/`. Never edit generated files under `ent/`.
- All fields use camelCase names (e.g. `createdAt`, `validFrom`, `hashedAuthCode`).
- Always set `createdAt` and `updatedAt` explicitly on creates.
- Load relationships eagerly with `.With...()` query builders; do not access `.Edges.X` without loading.

### Scheduler vs Jobs

- **Scheduler** (`schedulers/`): used to call functions on regular intervals, either with:
- - `SimpleFixedInterval` - runs on startup and is offset based on that. Restarts will reset the offset, which can result in it running more often than expected. Generally used for cleanup tasks or for modifying in-memory state.
- - `PersistentFixedInterval` - runs on first startup and then is offset based on that. Logs a warning on startup if calls were missed. Used for some more important tasks like enqueuing jobs to send active download session reminders.
- **Jobs** (`jobs/`): DB-queued, one-time execution, retriable, use for event-triggered async work like sending messages. More observable and can run for slightly longer than scheduler tasks.

Scheduled tasks should be defined centrally in `services/scheduler.go`. If a service needs something to be run regularly, it should expose a function in the interface for the scheduler to call. Jobs work somewhat similarly but are more often defined directly in the registry, a bit like HTTP handlers. If a service generates job definitions, like the messenger service, the concrete service struct should have a `RegisterJobs` method, which can be passed to `services.NewJobs()`. This shouldn't be part of the service interface.

### Environment Configuration

- All env vars live on `*common.Env`, loaded once at startup via `services.LoadEnvironmentVariables()`.
- Required vars use `common.RequireXxxEnv("VAR_NAME")` — panics on missing.
- Optional vars use `common.OptionalXxxEnv("VAR_NAME", defaultValue)`.
- Access config anywhere via `app.Env.VAR_NAME`.
- Add normalisation to `NormalizeEnvironmentVariables` and validation to `ValidateEnvironmentVariables` in `services/env.go`.
- Try to group environment variables into different sections, separated by new lines. Maintain the order between `common.Env`, `services/LoadEnvironmentVariables` and `testcommon.DefaultEnv`.
- When adding a new environment variable, ensure `testcommon.DefaultEnv` is updated.
- Environment variables should generally be required rather than optional unless they are for an optional feature or have a very sensible default. Due to the security critical nature of this app, admins are encouraged to review all the settings before deploying.

---

## Frontend (SvelteKit / TypeScript)

### Tech Stack

- **SvelteKit 5** with `ssr = false` and `prerender = false` (fully client-side).
- **Svelte 5 runes** — use `$state`, `$derived`, `$derived.by()`, `$props()`. Do not use legacy `$:` reactive declarations or Svelte stores.
- **Tailwind CSS v4** + **shadcn-svelte** component library (`$lib/components/ui/`).
- **`@sjsf/form`** with `@sjsf/shadcn4-theme` and `@sjsf/ajv8-validator` for schema-driven forms.
- **`cn()`** from `$lib/utils` (clsx + tailwind-merge) for conditional class composition.
- Currently lacking much structure, take the following instructions with a grain of salt

### API Client

Use `fetchJson` / `fetchAdminJson` from `$lib/api.ts`. Never use `fetch` directly in routes or components.

```ts
const response = await fetchJson(fetch, "/api/v1/users/download/", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" },
});
response.throwForStatus();
const data = response.data as MyResponseType;
```

- `fetchJson` auto-redirects to setup if the backend returns `ENDPOINT_NOT_FOUND`.
- `fetchAdminJson` automatically injects the admin auth header.
- Use `responseHasErrorCode(response, "ERROR_CODE")` to check for specific API error codes.

### Forms

Use `<JsonForm>` from `$lib/form/JsonForm.svelte` for all user-facing forms. Pass a JSON Schema and optional UI schema:

```svelte
<JsonForm
    schema={mySchema}
    uiSchema={myUiSchema}
    initialValue={{}}
    submitLabel="Submit"
    onSubmit={handleSubmit}
/>
```

For custom field rendering, extend the theme in `$lib/form/theme.ts`.

### Routing

- File-based routing under `src/routes/`. Each route folder has `+page.svelte` and optionally `+page.ts` for load functions.
- Use `resolve()` from `$app/paths` for internal links, not raw string paths.
- The root `+layout.ts` exports `ssr = false`, `prerender = false`, `trailingSlash = "always"`.

---

## Testing

### Go Tests

- **Always** use `_test` package suffix (enforced by `testpackage` linter): `package mypackage_test`.
- **Always** call `t.Parallel()` as the first line of every test and sub-test. There is very little implicit state in this project.
- Use `github.com/stretchr/testify/require` for assertions (`require.NoError`, `require.Equal`, etc.).
- Name error variables consistently in test code: `stdErr` for standard errors, etc.

**Integration tests** (endpoints, jobs): use `testhelpers.NewApp(t, options)` to get a fully wired app with an in-memory SQLite database. Call `testcommon.Post(t, app.Server, "/api/...", payload)` and assert with `testcommon.AssertJSONResponse`.

**Unit tests** (packages in isolation): construct only the specific services needed; do not call `testhelpers.NewApp`.

**Time-dependent tests:** use `clockwork.NewFakeClock()` and pass it via `testhelpers.AppOptions{Clock: clock}`. Otherwise just omit the option to use a real clock, except in unit tests where the app is created manually instead of using `testhelpers.NewApp`, in which case you'll need to set `Clock` to `clockwork.NewRealClock()`.

**Database setup:** `testcommon.CreateDB(t)` returns a `*testcommon.TestDatabase` backed by a unique SQLite in-memory database per test. Migrations run automatically via goose. It's hardly ever worth trying to mock the database.

**Entity creation in tests:** use `testcommon.NewDummyUser(counter, db.Client(), ctx, clock)` for quick scaffolding, or build entities directly via `tx.Entity.Create()...Save(ctx)` for test-specific data.

**Mock messengers:** `testhelpers.NewMockMessenger("NAME")` captures sent messages for assertion.

### Frontend Tests

- **Unit tests:** Vitest, co-located as `*.spec.ts` files, run with `npm run test:unit`.
- **E2E tests:** Playwright, located in `e2e/` as `*.test.ts` files, run with `npm run test:e2e`.

---

## Security Notes

- Use `subtle.ConstantTimeCompare` when appropriate.
- Secret codes should be stored hashed where possible: read access to the database shouldn't enable unauthorised actions. Jobs currently break this rule a bit, although currently successful jobs are immediately deleted, so there's a limited window for abuse.
- The login and stash systems are currently WIP. E2E encryption is planned but stashes are only encrypted at rest using the user's password.