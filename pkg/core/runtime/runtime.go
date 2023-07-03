package runtime

import (
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"sync"
	"time"
)

// Runtime represents initialized application state.
type Runtime interface {
	RuntimeInfo
	RuntimeContext
	component.Manager
}

type RuntimeInfo interface {
	GetInstanceId() string
	SetClusterId(clusterId string)
	GetClusterId() string
	GetStartTime() time.Time
}

type RuntimeContext interface {
	Config() kuma_cp.Config
	DataSourceLoader() datasource.Loader
	ResourceManager() core_manager.ResourceManager
}

type ExtraReportsFn func(Runtime) (map[string]string, error)

var _ Runtime = &runtime{}

type runtime struct {
	RuntimeInfo
	RuntimeContext
	component.Manager
}

var _ RuntimeInfo = &runtimeInfo{}

type runtimeInfo struct {
	mtx sync.RWMutex

	instanceId string
	clusterId  string
	startTime  time.Time
}

func (i *runtimeInfo) GetInstanceId() string {
	return i.instanceId
}

func (i *runtimeInfo) SetClusterId(clusterId string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	i.clusterId = clusterId
}

func (i *runtimeInfo) GetClusterId() string {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	return i.clusterId
}

func (i *runtimeInfo) GetStartTime() time.Time {
	return i.startTime
}

var _ RuntimeContext = &runtimeContext{}

type runtimeContext struct {
	cfg kuma_cp.Config
	rm  core_manager.ResourceManager
	rs  core_store.ResourceStore
}

func (rc *runtimeContext) Config() kuma_cp.Config {
	return rc.cfg
}

func (rc *runtimeContext) ResourceManager() core_manager.ResourceManager {
	return rc.rm
}

func (rc *runtimeContext) ResourceStore() core_store.ResourceStore {
	return rc.rs
}
