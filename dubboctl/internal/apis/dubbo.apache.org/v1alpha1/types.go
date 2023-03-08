package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DubboOperator struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
	Spec          *DubboOperatorSpec `json:"spec,omitempty"`
}

type DubboOperatorSpec struct {
	ManifestPath string               `json:"manifestPath"`
	Components   *DubboComponentsSpec `json:"components,omitempty"`
}

func (dos *DubboOperatorSpec) IsAdminEnabled() bool {
	if dos.Components != nil && dos.Components.IsAdminEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsGrafanaEnabled() bool {
	if dos.Components != nil && dos.Components.IsGrafanaEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsNacosEnabled() bool {
	if dos.Components != nil && dos.Components.IsNacosEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsZookeeperEnabled() bool {
	if dos.Components != nil && dos.Components.IsZookeeperEnabled() {
		return true
	}
	return false
}

type DubboComponentsSpec struct {
	Admin     *AdminSpec     `json:"admin,omitempty"`
	Grafana   *GrafanaSpec   `json:"grafana,omitempty"`
	Nacos     *NacosSpec     `json:"nacos,omitempty"`
	Zookeeper *ZookeeperSpec `json:"zookeeper,omitempty"`
}

func (dcs *DubboComponentsSpec) IsAdminEnabled() bool {
	if dcs.Admin != nil && dcs.Admin.Enabled {
		return true
	}
	return false
}

func (dcs *DubboComponentsSpec) IsGrafanaEnabled() bool {
	if dcs.Grafana != nil && dcs.Grafana.Enabled {
		return true
	}
	return false
}

func (dcs *DubboComponentsSpec) IsNacosEnabled() bool {
	if dcs.Nacos != nil && dcs.Nacos.Enabled {
		return true
	}
	return false
}

func (dcs *DubboComponentsSpec) IsZookeeperEnabled() bool {
	if dcs.Zookeeper != nil && dcs.Zookeeper.Enabled {
		return true
	}
	return false
}

type AdminSpec struct {
	Enabled        bool            `json:"enabled"`
	NameSpace      string          `json:"nameSpace"`
	Image          *Image          `json:"image,omitempty"`
	Replicas       uint32          `json:"replicas"`
	Global         *Global         `json:"global,omitempty"`
	Rbac           *Rbac           `json:"rbac,omitempty"`
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`
}

type Image struct {
	Registry    string   `json:"registry"`
	Tag         string   `json:"tag"`
	Debug       bool     `json:"debug"`
	PullPolicy  string   `json:"pullPolicy"`
	PullSecrets []string `json:"pullSecrets"`
}

type Global struct {
	ImageRegistry    string   `json:"imageRegistry"`
	ImagePullSecrets []string `json:"imagePullSecrets"`
}

type Rbac struct {
	Enabled    bool `json:"enabled"`
	PspEnabled bool `json:"pspEnabled"`
}

type ServiceAccount struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	NameTest string `json:"nameTest"`
}

type GrafanaSpec struct {
	Enabled   bool   `json:"enabled"`
	NameSpace string `json:"nameSpace"`
}

type NacosSpec struct {
	Enabled   bool   `json:"enabled"`
	NameSpace string `json:"nameSpace"`
}

type ZookeeperSpec struct {
	Enabled   bool   `json:"enabled"`
	NameSpace string `json:"nameSpace"`
}
