# Console API Web MVC Implementation

This reference explains how dubbo-admin exposes backend HTTP APIs through Gin handlers, typed models, console services, resource managers, and stores.

## End-to-end call chain

```text
runtime.RegisterComponent(&consoleWebServer{})
  -> Bootstrap initializes consoleWebServer.Init
  -> Init creates gin.Engine, middleware, static UI, health route
  -> Runtime starts consoleWebServer.Start
  -> Start creates consolectx.NewConsoleContext(coreRt)
  -> router.InitRouter(c.Engine, c.cs)
  -> HTTP request under /api/v1
  -> handler binds query/path/body
  -> handler calls service function
  -> service uses consolectx.Context and ctx.ResourceManager()
  -> manager/store reads or writes resources
  -> service returns typed model response
  -> handler wraps with model.NewSuccessResp or util.Handle*Error
```

## Console component

Key file: `pkg/console/component.go`

Console is registered as a runtime component and depends on `runtime.ResourceManager`:

```go
func init() {
    runtime.RegisterComponent(&consoleWebServer{})
}

func (c *consoleWebServer) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{runtime.ResourceManager}
}
```

`Init` sets up Gin:

- embedded admin UI mounted at `/admin`
- SPA fallback for `/admin/**`
- `/health` endpoint
- cookie session store
- auth middleware
- zap logging and recovery middleware
- Gin mode from config

`Start` creates console context and registers `/api/v1` routes:

```go
c.cs = consolectx.NewConsoleContext(coreRt)
router.InitRouter(c.Engine, c.cs)
httpServer := c.startHttpServer(errChan)
```

On stop, the HTTP server shuts down with `Shutdown(context.Background())`.

## Auth behavior

`authMiddleware` skips paths ending in `/login`. Other requests require a session value named `user`. Missing user returns HTTP 401 with `model.NewBizErrorResp`.

This means many API handlers assume authentication already passed.

## Console context

Key file: `pkg/console/context/context.go`

`consolectx.Context` wraps runtime access:

```go
type Context interface {
    ResourceManager() manager.ResourceManager
    CounterManager() counter.CounterManager
    Config() app.AdminConfig
    AppContext() context.Context
    LockManager() lock.Lock
}
```

`ResourceManager()` retrieves the runtime ResourceManager component and returns its manager:

```go
rmc, _ := c.coreRt.GetComponent(runtime.ResourceManager)
return rmc.(manager.ResourceManagerComponent).ResourceManager()
```

CounterManager and LockManager are optional and may return nil.

## Route registration

Key file: `pkg/console/router/router.go`

All Console API routes are grouped under `/api/v1`:

```go
router := r.Group("/api/v1")
```

Common groups include:

- `/auth`
- `/instance`
- `/application`
- `/service`
- `/configurator`
- `/condition-rule`
- `/tag-rule`
- global `/search`, `/overview`, `/metadata`, `/meshes`

When adding an endpoint, register the route in the existing group that matches frontend URL structure.

## Handler pattern

Handlers usually close over `consolectx.Context` and return `gin.HandlerFunc`:

```go
func GetApplicationDetail(ctx consolectx.Context) gin.HandlerFunc {
    return func(c *gin.Context) {
        req := &model.ApplicationDetailReq{}
        if err := c.ShouldBindQuery(req); err != nil {
            util.HandleArgumentError(c, err)
            return
        }
        resp, err := service.GetApplicationDetail(ctx, req)
        if err != nil {
            util.HandleServiceError(c, err)
            return
        }
        c.JSON(http.StatusOK, model.NewSuccessResp(resp))
    }
}
```

Read endpoints commonly use `ShouldBindQuery`. Mutation endpoints commonly use `ShouldBindJSON` or path parameters depending on existing handler style.

## Error and response model

Key files:

- `pkg/console/model/common.go`
- `pkg/console/util/error.go`

Success response:

```go
model.NewSuccessResp(data)
```

Service errors are normalized to `bizerror.Error` and returned with HTTP 200:

```go
func HandleServiceError(ctx *gin.Context, err error) {
    var e bizerror.Error
    if !errors.As(err, &e) {
        e = bizerror.New(bizerror.UnknownError, err.Error())
    }
    ctx.JSON(http.StatusOK, model.NewBizErrorResp(e))
}
```

Argument errors use `bizerror.InvalidArgument`. Auth middleware is an exception and returns HTTP 401.

## Service layer

Key directory: `pkg/console/service/`

Services receive `consolectx.Context` and typed request models. They usually access resources through generic manager helpers or `ctx.ResourceManager()` directly.

Typical resource query:

```go
resources, err := manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
    ctx.ResourceManager(),
    meshresource.ServiceProviderMetadataKind,
    []index.IndexCondition{...},
)
```

Do not put store/index details into handlers. Keep HTTP binding in handlers and resource logic in services.

## Frontend contract

Frontend API callers live under `ui-vue3/src/api/`. If a response field, request parameter, or endpoint path changes, check both Console model structs and frontend caller usage.

## Common failure modes

- Route registered under wrong group or duplicate service group block.
- Handler binds query when frontend sends JSON, or vice versa.
- Model tags do not match frontend parameter names.
- Service returns raw resource shape not expected by frontend view.
- Error returned directly instead of through `util.HandleServiceError`.
- Optional managers such as CounterManager or LockManager are nil.

## Review checklist

- Route path and HTTP method match frontend usage.
- Handler binding matches request source.
- Request/response models have correct tags and JSON field names.
- Service owns business logic and uses ResourceManager/store helpers.
- Response uses `CommonResp` consistently.
- Frontend API client is updated when contract changes.
