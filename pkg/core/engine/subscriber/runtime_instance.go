package subscriber

import (
	"reflect"

	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type RuntimeInstanceEventSubscriber struct {
	instanceResourceStore store.ResourceStore
}

func (s *RuntimeInstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.RuntimeInstanceKind
}

func (s *RuntimeInstanceEventSubscriber) Name() string {
	return "Engine-" + s.ResourceKind().ToString()
}

func (s *RuntimeInstanceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.RuntimeInstanceResource)
	if !ok {
		return bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(event.NewObj()).Name())
	}
	oldObj, ok := event.OldObj().(*meshresource.RuntimeInstanceResource)
	if !ok {
		return bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(event.OldObj()).Name())
	}
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		return s.processUpsert(newObj)
	case cache.Deleted:
		return s.processDelete(oldObj)
	}
	return nil
}

func (s *RuntimeInstanceEventSubscriber) getRelatedInstanceResource(
	rtInstance *meshresource.RuntimeInstanceResource) (*meshresource.InstanceResource, error) {
	resources, err := s.instanceResourceStore.ListByIndexes(map[string]string{
		index.ByInstanceIpIndex: rtInstance.Spec.Ip,
	})
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil
	}
	instanceResources := make([]*meshresource.InstanceResource, len(resources))
	for i, item := range resources {
		if res, ok := item.(*meshresource.InstanceResource); ok {
			instanceResources[i] = res
		} else {
			return nil, bizerror.NewAssertionError("InstanceResource", reflect.TypeOf(item).Name())
		}
	}
	return instanceResources[0], nil
}

func (s *RuntimeInstanceEventSubscriber) mergeRuntimeInstance(
	instanceRes *meshresource.InstanceResource,
	rtInstanceRes *meshresource.RuntimeInstanceResource) {
	instanceRes.Name = rtInstanceRes.Name
	instanceRes.Spec.Name = rtInstanceRes.Spec.Name
	instanceRes.Spec.Ip = rtInstanceRes.Spec.Ip
	instanceRes.Labels = rtInstanceRes.Labels
	instanceRes.Spec.Image = rtInstanceRes.Spec.Image
	instanceRes.Spec.CreateTime = rtInstanceRes.Spec.CreateTime
	instanceRes.Spec.StartTime = rtInstanceRes.Spec.StartTime
	instanceRes.Spec.ReadyTime = rtInstanceRes.Spec.ReadyTime
	instanceRes.Spec.DeployState = rtInstanceRes.Spec.Phase
	instanceRes.Spec.WorkloadType = rtInstanceRes.Spec.WorkloadType
	instanceRes.Spec.WorkloadName = rtInstanceRes.Spec.WorkloadName
	instanceRes.Spec.Node = rtInstanceRes.Spec.Node
	instanceRes.Spec.Probes = rtInstanceRes.Spec.Probes
	instanceRes.Spec.Conditions = rtInstanceRes.Spec.Conditions
}

func (s *RuntimeInstanceEventSubscriber) fromRuntimeInstance(
	rtInstanceRes *meshresource.RuntimeInstanceResource) *meshresource.InstanceResource {
	instanceRes := meshresource.NewInstanceResourceWithAttributes(rtInstanceRes.Mesh, rtInstanceRes.Name)
	s.mergeRuntimeInstance(instanceRes, rtInstanceRes)
	return instanceRes
}

// processUpsert when runtime instance added or updated, we should add/update the corresponding instance resource
func (s *RuntimeInstanceEventSubscriber) processUpsert(rtInstanceRes *meshresource.RuntimeInstanceResource) error {
	instanceResource, err := s.getRelatedInstanceResource(rtInstanceRes)
	if err != nil {
		return err
	}
	// If instance resource exists, the rpc instance resource exists in remote registry and has been watched by discovery.
	// So we should merge the runtime info into it
	if instanceResource != nil {
		s.mergeRuntimeInstance(instanceResource, rtInstanceRes)
		return s.instanceResourceStore.Update(instanceResource)
	}
	// If instance resource does not exist, we should create a new instance resource by runtime instance
	// If the app name is empty, we cannot identify it as a dubbo app, so we skip it
	if strutil.IsBlank(rtInstanceRes.Spec.AppName) {
		logger.Warnf("cannot identify runtime instance %s as a dubbo app, skipped updating instance", rtInstanceRes.Name)
		return nil
	}
	// Otherwise we can create a new instance resource by runtime instance
	instanceRes := s.fromRuntimeInstance(rtInstanceRes)
	return s.instanceResourceStore.Add(instanceRes)
}

// processDelete when runtime instance deleted, we should delete the corresponding instance resource
func (s *RuntimeInstanceEventSubscriber) processDelete(rtInstanceRes *meshresource.RuntimeInstanceResource) error {
	instanceResource, err := s.getRelatedInstanceResource(rtInstanceRes)
	if err != nil {
		return err
	}
	if instanceResource == nil {
		return nil
	}
	return s.instanceResourceStore.Delete(instanceResource.ResourceKey())
}

func NewRuntimeInstanceEventSubscriber(instanceResourceStore store.ResourceStore) events.Subscriber {
	return &RuntimeInstanceEventSubscriber{
		instanceResourceStore: instanceResourceStore,
	}
}
