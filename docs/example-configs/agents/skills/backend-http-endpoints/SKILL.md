---
name: backend-http-endpoints
description: 'Add or modify HTTP endpoints in the Cryptic Stash backend. Use when: creating new API routes, writing endpoint handlers, handling errors in handlers, configuring auth middleware, or returning JSON error responses.'
---

# HTTP Endpoints & Error Handling

## Route Structure

Endpoints follow a `ConfigureEndpoints(group *servercommon.Group)` pattern. Each resource package gets its own file:

```go
// server/endpoints/v1/self/self.go
func ConfigureEndpoints(group *servercommon.Group) {
    passkeyGroup := group.Group("/passkeys")
    passkeyGroup.Use(group.App.SuperUserModeMiddleware)
    passkeys.ConfigureEndpoints(passkeyGroup)
}
```

`group.Group("/path")` creates a nested `*servercommon.Group` (wraps `*gin.RouterGroup` + `*ServerApp`).

## Auth Middleware

Three middleware functions on `*servercommon.ServerApp`:

| Middleware | Scheme | Endpoint Type |
|---|---|---|
| `DefaultAuthMiddleware` | `Bearer <token>` | Any authenticated user |
| `SuperUserModeMiddleware` | `Bearer <token>` (elevated) | Sensitive operations (passkey management, account changes) |
| `AdminMiddleware` | `AdminCode <code>` | Admin-only operations |

`SuperUserModeMiddleware` is `SessionAuth` with `RequireSuperuser: true` — the user must re-authenticate via WebAuthn before using these endpoints.

## Handler Patterns

### Standard Handler

Use `servercommon.NewHandler` — returns `*servercommon.Error` or `nil`:

```go
func MyEndpoint(app *servercommon.ServerApp) gin.HandlerFunc {
    return servercommon.NewHandler(func(ginCtx *gin.Context) error {
        // Parse body
        body := MyPayload{}
        if serverErr := servercommon.ParseBody(&body, ginCtx); serverErr != nil {
            return serverErr
        }
        // ... business logic ...
        ginCtx.JSON(http.StatusOK, MyResponse{Errors: []servercommon.ErrorDetail{}})
        return nil
    })
}
```

### Object ID Handler (wrapper around NewHandler)

`servercommon.NewObjectIDHandler` parses a UUID path parameter:

```go
group.GET("/:objectId/", servercommon.NewObjectIDHandler(
    func(ginCtx *gin.Context, objectID uuid.UUID) error { ... },
))
```

## Error Handling

### Error Type

`*servercommon.Error` wraps `common.WrappedError` with HTTP metadata:

```go
type Error struct {
    child     common.WrappedError  // underlying error
    status    int                  // HTTP status (-1 = not set = 500 status)
    details   []ErrorDetail        // client-facing messages
    shouldLog bool                 // whether to log server-side
}
type ErrorDetail struct {
    Message string `json:"message"`
    Code    string `json:"code"`
}
```

### Starters vs Chainers

**Starters** create a new `*servercommon.Error` from a raw error. Use at the start of error handling:

| Starter | Purpose |
|---------|---------|
| `NewError(stdErr)` | Wrap any error |
| `NewRollbackError()` | Signal transaction rollback (logging disabled) |
| `Send404IfNotFound(stdErr)` | Shortcut: 404 if `ent.NotFound` |
| `SendUnauthorizedIfNotFound(stdErr)` | Shortcut: 401 if `ent.NotFound` |
| `ExpectError(stdErr, expected, code, detail)` | Match specific error → set status + detail |
| `ExpectAnyOfErrors(stdErr, expectedStdErrs, code, detail)` | Match any of several errors |

**Chainers** operate on an existing `*servercommon.Error`. Chain after a starter:

| Chainer | Purpose |
|---------|---------|
| `.SetStatus(code)` | Set HTTP status |
| `.AddDetail(detail)` | Add client-facing error detail |
| `.Send404IfNotFound()` | 404 if underlying error is `ent.NotFound` |
| `.SendUnauthorizedIfNotFound()` | 401 if underlying error is `ent.NotFound` |
| `.Expect(expected, code, detail)` | Match error → set status + detail |
| `.ExpectAnyOf(expectedStdErrs, code, detail)` | Match any of several |
| `.DisableLogging()` / `.EnableLogging()` | Control server-side logging |
| `.ConfigureRetries(max, backoff, multiplier)` | Used for interoperability with common.WrappedError* |

*Mostly used to remove retry config rather than add it. `servercommon.NewHandler` doesn't implement retries.

### Naming Convention

The starter has `Error` in the name (`ExpectError`). The chaining equivalent drops it (`.Expect`) because you're already in the context of an error.

### Premade Sentinels

Sentinel errors must either be cloned or wrapped before being returned to avoid accidental global mutation. The `NewXXXError` methods return a clone of the sentinel, e.g `NewUnauthorizedError` -> `ErrUnauthorized`.

```go
servercommon.NewUnauthorizedError()                           // 401, logging disabled
servercommon.NewForbiddenError()                              // 403, logging disabled
servercommon.NewNotFoundError()                               // 404, logging disabled
servercommon.NewBadRequestError(field, message, errorCode)    // 400 with detail
```

### Error Flow

In `NewHandler`-wrapped handlers: **return** the `*servercommon.Error`. Gin middleware handles the errors.

In middleware (raw `gin.HandlerFunc`s): use `ginCtx.Error(serverErr)` + `ginCtx.Abort()`.

### Responses

All JSON responses include an `"errors"` field (even if empty):

```go
type MyResponse struct {
    Errors []servercommon.ErrorDetail `json:"errors"`
    // ... data fields ...
}
```

## Route Registration

Top-level v1 router in `server/endpoints/v1/v1.go`. Register new resource groups in `ConfigureEndpoints`:

```go
func ConfigureEndpoints(group *servercommon.Group) {
    // Public endpoints
    myResource.ConfigureEndpoints(group.Group("/my-resource"))

    // Auth'd endpoints
    authGroup := group.Group("/my-auth-resource")
    authGroup.Use(group.App.DefaultAuthMiddleware)
    myAuthResource.ConfigureEndpoints(authGroup)
}
```
