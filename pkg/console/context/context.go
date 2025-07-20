package context

import (
	ctx "context"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type Context interface {
	ResourceManager() manager.ResourceManager

	ResourceStore() store.ResourceStore

	Config() app.AdminConfig

	AppContext() ctx.Context
}

var _ Context = &context{}

func NewConsoleContext(coreRt runtime.Runtime) Context {
	return &context{
		coreRt: coreRt,
	}
}

type context struct {
	coreRt runtime.Runtime
}

func (c *context) AppContext() ctx.Context {
	return c.coreRt.AppContext()
}

func (c *context) Config() app.AdminConfig {
	return c.coreRt.Config()
}

func (c *context) ResourceManager() manager.ResourceManager {
	rmc, _ := c.coreRt.GetComponent(runtime.ResourceManager)
	return rmc.(manager.ResourceManagerComponent).ResourceManager()
}

func (c *context) ResourceStore() store.ResourceStore {
	rsc, _ := c.coreRt.GetComponent(runtime.ResourceStore)
	return rsc.(store.BaseResourceStoreComponent).ResourceStore()
}
