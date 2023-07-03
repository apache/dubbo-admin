package runtime

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/core"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"os"
	"time"

	"github.com/pkg/errors"
)

// BuilderContext provides access to Builder's interim state.
type BuilderContext interface {
	ComponentManager() component.Manager
	ResourceStore() core_store.ResourceStore
	SecretStore() store.SecretStore
	ConfigStore() core_store.ResourceStore
	ResourceManager() core_manager.CustomizableResourceManager
	Config() kuma_cp.Config
	DataSourceLoader() datasource.Loader
	Extensions() context.Context
	ConfigManager() config_manager.ConfigManager
	LeaderInfo() component.LeaderInfo
	Metrics() metrics.Metrics
	EventReaderFactory() events.ListenerFactory
	APIManager() api_server.APIManager
	XDSHooks() *xds_hooks.Hooks
	CAProvider() secrets.CaProvider
	DpServer() *dp_server.DpServer
	ResourceValidators() ResourceValidators
	KDSContext() *kds_context.Context
	APIServerAuthenticator() authn.Authenticator
	Access() Access
	TokenIssuers() builtin.TokenIssuers
	MeshCache() *mesh.Cache
	InterCPClientPool() *client.Pool
}

var _ BuilderContext = &Builder{}

// Builder represents a multi-step initialization process.
type Builder struct {
	cfg kuma_cp.Config
	cm  component.Manager
	rs  core_store.ResourceStore
	*runtimeInfo
}

func BuilderFor(appCtx context.Context, cfg kuma_cp.Config) (*Builder, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not get hostname")
	}
	suffix := core.NewUUID()[0:4]
	return &Builder{
		cfg: cfg,
		ext: context.Background(),
		cam: core_ca.Managers{},
		runtimeInfo: &runtimeInfo{
			instanceId: fmt.Sprintf("%s-%s", hostname, suffix),
			startTime:  time.Now(),
		},
		appCtx: appCtx,
	}, nil
}

func (b *Builder) WithComponentManager(cm component.Manager) *Builder {
	b.cm = cm
	return b
}

func (b *Builder) Build() (Runtime, error) {
	if b.cm == nil {
		return nil, errors.Errorf("ComponentManager has not been configured")
	}
	return &runtime{
		RuntimeInfo: b.runtimeInfo,
		RuntimeContext: &runtimeContext{
			cfg:            b.cfg,
			rm:             b.rm,
			rom:            b.rom,
			rs:             b.rs,
			ss:             b.ss,
			cam:            b.cam,
			dsl:            b.dsl,
			ext:            b.ext,
			configm:        b.configm,
			leadInfo:       b.leadInfo,
			lif:            b.lif,
			eac:            b.eac,
			metrics:        b.metrics,
			erf:            b.erf,
			apim:           b.apim,
			xdsauth:        b.xdsauth,
			xdsCallbacks:   b.xdsCallbacks,
			xdsh:           b.xdsh,
			cap:            b.cap,
			dps:            b.dps,
			kdsctx:         b.kdsctx,
			rv:             b.rv,
			au:             b.au,
			acc:            b.acc,
			appCtx:         b.appCtx,
			extraReportsFn: b.extraReportsFn,
			tokenIssuers:   b.tokenIssuers,
			meshCache:      b.meshCache,
			interCpPool:    b.interCpPool,
		},
		Manager: b.cm,
	}, nil
}

func (b *Builder) ComponentManager() component.Manager {
	return b.cm
}
