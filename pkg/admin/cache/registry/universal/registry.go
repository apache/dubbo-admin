package universal

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-admin/pkg/admin/cache/registry"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	gxset "github.com/dubbogo/gost/container/set"
	"net/url"
)

var (
	SUBSCRIBE *common.URL
)

func init() {
	registry.AddRegistry("universal", func(u *common.URL) (registry.AdminRegistry, error) {
		delegate, err := extension.GetRegistry(u.Protocol, u)
		if err != nil {
			logger.Error("Error initialize registry instance.")
			return nil, err
		}

		sdUrl := u.Clone()
		sdUrl.AddParam("registry", u.Protocol)
		sdUrl.Protocol = "service-discovery"
		sdDelegate, err := extension.GetServiceDiscovery(sdUrl)
		if err != nil {
			logger.Error("Error initialize service discovery instance.")
			return nil, err
		}
		return NewRegistry(delegate, sdDelegate), nil
	})

	queryParams := url.Values{
		constant.InterfaceKey:  {constant.AnyValue},
		constant.GroupKey:      {constant.AnyValue},
		constant.VersionKey:    {constant.AnyValue},
		constant.ClassifierKey: {constant.AnyValue},
		constant.CategoryKey: {constant.ProvidersCategory +
			"," + constant.ConsumersCategory +
			"," + constant.RoutersCategory +
			"," + constant.ConfiguratorsCategory},
		constant.EnabledKey: {constant.AnyValue},
		constant.CheckKey:   {"false"},
	}
	SUBSCRIBE, _ = common.NewURL(common.GetLocalIp()+":0",
		common.WithProtocol(constant.AdminProtocol),
		common.WithParams(queryParams),
	)
}

type MappingListener struct {
}

type Registry struct {
	delegate   dubboRegistry.Registry
	sdDelegate dubboRegistry.ServiceDiscovery
}

func NewRegistry(delegate dubboRegistry.Registry, sdDelegate dubboRegistry.ServiceDiscovery) *Registry {
	return &Registry{
		delegate:   delegate,
		sdDelegate: sdDelegate,
	}
}

func (kr *Registry) Subscribe(listener registry.AdminNotifyListener) error {
	delRegistryListener := DubboRegistryNotifyListener{listener: listener}
	go func() {
		err := kr.delegate.Subscribe(SUBSCRIBE, delRegistryListener)
		if err != nil {
			logger.Error("Failed to subscribe to registry, might not be able to show services of the cluster!")
		}
	}()

	go func() {
		mappings, err := getMappingList("mapping")
		for interfaceKey, oldApps := range mappings {
			mappingListener := NewMappingListener(oldApps, delRegistryListener)
			apps, _ := config.MetadataReportCenter.GetServiceAppMapping(interfaceKey, "mapping", mappingListener)
			delSDListener := NewDubboSDNotifyListener(apps)
			for appTmp, _ := range apps.Items {
				app := appTmp.(string)
				instances := kr.sdDelegate.GetInstances(app)
				logger.Infof("Synchronized instance notification on subscription, instance list size %s", len(instances))
				if len(instances) > 0 {
					err = delSDListener.OnEvent(&dubboRegistry.ServiceInstancesChangedEvent{
						ServiceName: app,
						Instances:   instances,
					})
					if err != nil {
						logger.Warnf("[ServiceDiscoveryRegistry] ServiceInstancesChangedListenerImpl handle error:%v", err)
					}
				}
			}
			delSDListener.AddListenerAndNotify(interfaceKey, delRegistryListener)
			err = kr.sdDelegate.AddListener(delSDListener)
		}
	}()

	return nil
}

func (kr *Registry) Destroy() error {

	return nil
}

type DubboRegistryNotifyListener struct {
	listener registry.AdminNotifyListener
}

func (l DubboRegistryNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	// TODO implement me
	serviceURL := event.Service
	l.listener.Notify(serviceURL)
}

func (l DubboRegistryNotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	for _, event := range events {
		l.Notify(event)
	}
}

func getMappingList(group string) (map[string]*gxset.HashSet, error) {
	keys, err := config.MetadataReportCenter.GetConfigKeysByGroup(group)
	if err != nil {
		return nil, err
	}

	list := make(map[string]*gxset.HashSet)
	for k := range keys.Items {
		interfaceKey, _ := k.(string)
		if !(interfaceKey == "org.apache.dubbo.mock.api.MockService") {
			rule, err := config.MetadataReportCenter.GetServiceAppMapping(interfaceKey, group, nil)
			if err != nil {
				return nil, err
			}
			list[interfaceKey] = rule
		}
	}
	return list, nil
}
