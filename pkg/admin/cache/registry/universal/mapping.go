package universal

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
	"strings"
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
)

type ServiceMappingChangedListenerImpl struct {
	oldServiceNames *gxset.HashSet
	listener        registry.NotifyListener
	interfaceKey    string

	mux           sync.Mutex
	delSDRegistry registry.ServiceDiscovery
}

func NewMappingListener(oldServiceNames *gxset.HashSet, listener registry.NotifyListener) *ServiceMappingChangedListenerImpl {
	return &ServiceMappingChangedListenerImpl{
		listener:        listener,
		oldServiceNames: oldServiceNames,
	}
}

func parseServices(literalServices string) *gxset.HashSet {
	set := gxset.NewSet()
	if len(literalServices) == 0 {
		return set
	}
	var splitServices = strings.Split(literalServices, ",")
	for _, s := range splitServices {
		if len(s) != 0 {
			set.Add(s)
		}
	}
	return set
}

// OnEvent on ServiceMappingChangedEvent the service mapping change event
func (lstn *ServiceMappingChangedListenerImpl) OnEvent(e observer.Event) error {
	lstn.mux.Lock()

	sm, ok := e.(*registry.ServiceMappingChangeEvent)
	if !ok {
		return nil
	}
	newServiceNames := sm.GetServiceNames()
	oldServiceNames := lstn.oldServiceNames
	// serviceMapping is orderly
	if newServiceNames.Empty() || oldServiceNames.String() == newServiceNames.String() {
		return nil
	}

	err := lstn.updateListener(lstn.interfaceKey, newServiceNames)
	if err != nil {
		return err
	}
	lstn.oldServiceNames = newServiceNames
	lstn.mux.Unlock()

	return nil
}

func (lstn *ServiceMappingChangedListenerImpl) updateListener(interfaceKey string, apps *gxset.HashSet) error {
	delSDListener := NewDubboSDNotifyListener(apps)
	delSDListener.AddListenerAndNotify(interfaceKey, lstn.listener)
	err := lstn.delSDRegistry.AddListener(delSDListener)
	return err
}

// Stop on ServiceMappingChangedEvent the service mapping change event
func (lstn *ServiceMappingChangedListenerImpl) Stop() {

}
