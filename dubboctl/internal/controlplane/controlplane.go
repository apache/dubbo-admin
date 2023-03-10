package controlplane

import (
	"errors"
	"fmt"
	"github.com/dubbogo/dubbogo-cli/internal/apis/dubbo.apache.org/v1alpha1"
)

type DubboControlPlane struct {
	spec       *v1alpha1.DubboOperatorSpec
	started    bool
	components map[ComponentName]Component
}

func (dcp *DubboControlPlane) Run() error {
	for name, component := range dcp.components {
		if err := component.Run(); err != nil {
			return fmt.Errorf("component %s run failed, err: %s", name, err)
		}
	}
	dcp.started = true
	return nil
}

func (dcp *DubboControlPlane) RenderManifest() (map[ComponentName]string, error) {
	if !dcp.started {
		return nil, errors.New("DubboControlPlane is not running")
	}
	res := make(map[ComponentName]string)
	for name, component := range dcp.components {
		manifest, err := component.RenderManifest()
		if err != nil {
			return nil, fmt.Errorf("component %s RenderManifest err: %v", name, err)
		}
		res[name] = manifest
	}
	return res, nil
}

func NewDubboControlPlane(spec *v1alpha1.DubboOperatorSpec) (*DubboControlPlane, error) {
	if spec == nil {
		return nil, errors.New("DubboOperatorSpec is empty")
	}
	// initialized
	components := make(map[ComponentName]Component)
	if spec.IsAdminEnabled() {
		admin, err := NewAdminComponent(spec.Components.Admin, spec.ComponentsMeta.Admin.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Admin] = admin
	}
	if spec.IsGrafanaEnabled() {
		grafana, err := NewGrafanaComponent(spec.Components.Grafana, spec.ComponentsMeta.Grafana.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Grafana] = grafana
	}
	if spec.IsNacosEnabled() {
		nacos, err := NewNacosComponent(spec.Components.Nacos, spec.ComponentsMeta.Nacos.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Nacos] = nacos
	}
	if spec.IsZookeeperEnabled() {
		zookeeper, err := NewZookeeperComponent(spec.Components.Zookeeper, spec.ComponentsMeta.Zookeeper.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Zookeeper] = zookeeper
	}
	dcp := &DubboControlPlane{
		spec:       spec,
		components: components,
	}
	return dcp, nil
}
