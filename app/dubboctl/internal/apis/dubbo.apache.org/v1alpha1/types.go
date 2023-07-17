// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true

// DubboConfig describes configuration for DubboOperator
type DubboConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *DubboConfigSpec   `json:"spec,omitempty"`
	Status *DubboConfigStatus `json:"status,omitempty"`
}

func (do *DubboConfig) GetProfile() string {
	if do.Spec != nil {
		return do.Spec.Profile
	}
	return ""
}

type DubboConfigStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DubboConfigList is a list of DubboConfig resources
type DubboConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []DubboConfig `json:"items"`
}

type DubboConfigSpec struct {
	ProfilePath    string               `json:"profilePath,omitempty"`
	Profile        string               `json:"profile,omitempty"`
	ChartPath      string               `json:"chartPath,omitempty"`
	Namespace      string               `json:"namespace,omitempty"`
	ComponentsMeta *DubboComponentsMeta `json:"componentsMeta,omitempty"`
	Components     *DubboComponentsSpec `json:"components,omitempty"`
}

func (dcs *DubboConfigSpec) IsAdminEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsAdminEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsGrafanaEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsGrafanaEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsNacosEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsNacosEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsZookeeperEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsZookeeperEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsPrometheusEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsPrometheusEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsSkywalkingEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsSkywalkingEnabled() {
		return true
	}
	return false
}

func (dcs *DubboConfigSpec) IsZipkinEnabled() bool {
	if dcs.ComponentsMeta != nil && dcs.ComponentsMeta.IsZipkinEnabled() {
		return true
	}
	return false
}

type DubboComponentsMeta struct {
	Admin      *AdminMeta      `json:"admin,omitempty"`
	Grafana    *GrafanaMeta    `json:"grafana,omitempty"`
	Nacos      *NacosMeta      `json:"nacos,omitempty"`
	Zookeeper  *ZookeeperMeta  `json:"zookeeper,omitempty"`
	Prometheus *PrometheusMeta `json:"prometheus,omitempty"`
	Skywalking *SkywalkingMeta `json:"skywalking,omitempty"`
	Zipkin     *ZipkinMeta     `json:"zipkin,omitempty"`
}

type BaseMeta struct {
	Enabled bool `json:"enabled,omitempty"`
}

type RemoteMeta struct {
	RepoURL string `json:"repoURL,omitempty"`
	Version string `json:"version,omitempty"`
}

type AdminMeta struct {
	BaseMeta
}

type GrafanaMeta struct {
	BaseMeta
	RemoteMeta
}

type NacosMeta struct {
	BaseMeta
}

type ZookeeperMeta struct {
	BaseMeta
	RemoteMeta
}

type PrometheusMeta struct {
	BaseMeta
	RemoteMeta
}

type SkywalkingMeta struct {
	BaseMeta
	RemoteMeta
}

type ZipkinMeta struct {
	BaseMeta
	RemoteMeta
}

func (dcm *DubboComponentsMeta) IsAdminEnabled() bool {
	if dcm.Admin != nil && dcm.Admin.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsGrafanaEnabled() bool {
	if dcm.Grafana != nil && dcm.Grafana.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsNacosEnabled() bool {
	if dcm.Nacos != nil && dcm.Nacos.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsZookeeperEnabled() bool {
	if dcm.Zookeeper != nil && dcm.Zookeeper.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsPrometheusEnabled() bool {
	if dcm.Prometheus != nil && dcm.Prometheus.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsSkywalkingEnabled() bool {
	if dcm.Skywalking != nil && dcm.Skywalking.Enabled {
		return true
	}
	return false
}

func (dcm *DubboComponentsMeta) IsZipkinEnabled() bool {
	if dcm.Zipkin != nil && dcm.Zipkin.Enabled {
		return true
	}
	return false
}

type DubboComponentsSpec struct {
	Admin      *AdminSpec      `json:"admin,omitempty"`
	Grafana    *GrafanaSpec    `json:"grafana,omitempty"`
	Nacos      *NacosSpec      `json:"nacos,omitempty"`
	Zookeeper  *ZookeeperSpec  `json:"zookeeper,omitempty"`
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
	Skywalking *SkywalkingSpec `json:"skywalking,omitempty"`
	Zipkin     *ZipkinSpec     `json:"zipkin,omitempty"`
}

type AdminSpec struct {
	Image              *Image              `json:"image,omitempty"`
	Replicas           uint32              `json:"replicas"`
	Global             *AdminGlobal        `json:"global,omitempty"`
	Rbac               *Rbac               `json:"rbac,omitempty"`
	ServiceAccount     *ServiceAccount     `json:"serviceAccount,omitempty"`
	ImagePullSecrets   []string            `json:"imagePullSecrets,omitempty"`
	Autoscaling        *Autoscaling        `json:"autoscaling,omitempty"`
	DeploymentStrategy *DeploymentStrategy `json:"deploymentStrategy,omitempty"`
	corev1.ContainerImage
}

type Image struct {
	Registry    string   `json:"registry"`
	Tag         string   `json:"tag"`
	Debug       bool     `json:"debug"`
	PullPolicy  string   `json:"pullPolicy"`
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type AdminGlobal struct {
	ImageRegistry    string   `json:"imageRegistry"`
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

type Rbac struct {
	Enabled    bool `json:"enabled"`
	PspEnabled bool `json:"pspEnabled"`
}

type ServiceAccount struct {
	Enabled     bool              `json:"enabled"`
	Name        map[string]string `json:"name,omitempty"`
	NameTest    map[string]string `json:"nameTest,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Autoscaling struct {
	Enabled      bool              `json:"enabled,omitempty"`
	MinReplicas  uint32            `json:"minReplicas,omitempty"`
	MaxReplicas  uint32            `json:"maxReplicas,omitempty"`
	TargetCPU    uint32            `json:"targetCPU,omitempty"`
	TargetMemory uint32            `json:"targetMemory,omitempty"`
	Behavior     map[string]string `json:"behavior,omitempty"`
}

type DeploymentStrategy struct {
	Type string `json:"type"`
}

type GrafanaSpec map[string]any

func (in *GrafanaSpec) DeepCopyInto(out *GrafanaSpec) {
	if in == nil {
		return
	}
	var spec GrafanaSpec = map[string]any{}
	for key, val := range *in {
		spec[key] = val
	}
	*out = spec
}

type NacosSpec struct {
	Global              *NacosGlobal                  `json:"global,omitempty"`
	Image               *NacosImage                   `json:"image,omitempty"`
	Plugin              *NacosPlugin                  `json:"plugin,omitempty"`
	Replicas            uint32                        `json:"replicas,omitempty"`
	DomainName          string                        `json:"domainName,omitempty"`
	Storage             *NacosStorage                 `json:"storage,omitempty"`
	Service             *NacosService                 `json:"service,omitempty"`
	Persistence         *NacosPersistence             `json:"persistence,omitempty"`
	PodDisruptionBudget *policyv1.PodDisruptionBudget `json:"podDisruptionBudget,omitempty"`
	Ingress             *Ingress                      `json:"ingress,omitempty"`
	NetworkPolicy       *NetworkPolicy                `json:"networkPolicy,omitempty"`
}

type Mode string

const (
	Cluster    Mode = "cluster"
	Standalone Mode = "standalone"
)

type NacosGlobal struct {
	Mode Mode `json:"mode,omitempty"`
}

type NacosImage struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

type NacosPlugin struct {
	Enable bool              `json:"enable,omitempty"`
	Image  *NacosPluginImage `json:"image,omitempty"`
}

type NacosPluginImage struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	PullPolicy string `json:"pullPolicy,omitempty"`
}

type NacosStorage struct {
	Type string          `json:"type,omitempty"`
	Db   *NacosStorageDb `json:"db,omitempty"`
}

type NacosStorageDb struct {
	Host     string `json:"host,omitempty"`
	Name     string `json:"name,omitempty"`
	Port     uint32 `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Param    string `json:"param,omitempty"`
}

type NacosService struct {
	Name                     string             `json:"name,omitempty"`
	Type                     corev1.ServiceType `json:"type,omitempty"`
	Port                     uint32             `json:"port,omitempty"`
	NodePort                 uint32             `json:"nodePort,omitempty"`
	ClusterIP                string             `json:"clusterIP,omitempty"`
	LoadBalancerIP           string             `json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges string             `json:"loadBalancerSourceRanges,omitempty"`
	ExternalIPs              string             `json:"externalIPs,omitempty"`
}

type NacosPersistence struct {
	Enabled          bool                                `json:"enabled,omitempty"`
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	StorageClassName string                              `json:"storageClassName,omitempty"`
	Size             *resource.Quantity                  `json:"size,omitempty"`
	// todo: need to reconcile between this struct and values.yaml
	ClaimName string                       `json:"claimName,omitempty"`
	EmptyDir  *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

type Ingress struct {
	Enabled     bool                `json:"enabled,omitempty"`
	Annotations map[string]string   `json:"annotations,omitempty"`
	Labels      map[string]string   `json:"labels,omitempty"`
	Path        string              `json:"path,omitempty"`
	PathType    netv1.PathType      `json:"pathType,omitempty"`
	Hosts       []string            `json:"hosts,omitempty"`
	ExtraPaths  []*ExtraPath        `json:"extraPaths,omitempty"`
	Tls         []*netv1.IngressTLS `json:"tls,omitempty"`
}

type ExtraPath struct {
	Path     string               `json:"path,omitempty"`
	PathType netv1.PathType       `json:"pathType,omitempty"`
	Backend  netv1.IngressBackend `json:"backend,omitempty"`
}

type NetworkPolicy struct {
	Enabled bool    `json:"enabled,omitempty"`
	Ingress bool    `json:"ingress,omitempty"`
	Egress  *Egress `json:"egress,omitempty"`
}

type Egress struct {
	Enabled bool     `json:"enabled,omitempty"`
	Ports   []uint32 `json:"ports,omitempty"`
}

type ZookeeperSpec map[string]any

func (in *ZookeeperSpec) DeepCopyInto(out *ZookeeperSpec) {
	if in == nil {
		return
	}
	var spec ZookeeperSpec = map[string]any{}
	for key, val := range *in {
		spec[key] = val
	}
	*out = spec
}

type PrometheusSpec map[string]any

func (in *PrometheusSpec) DeepCopyInto(out *PrometheusSpec) {
	if in == nil {
		return
	}
	var spec PrometheusSpec = map[string]any{}
	for key, val := range *in {
		spec[key] = val
	}
	*out = spec
}

type SkywalkingSpec map[string]any

func (in *SkywalkingSpec) DeepCopyInto(out *SkywalkingSpec) {
	if in == nil {
		return
	}
	var spec SkywalkingSpec = map[string]any{}
	for key, val := range *in {
		spec[key] = val
	}
	*out = spec
}

type ZipkinSpec map[string]any

func (in *ZipkinSpec) DeepCopyInto(out *ZipkinSpec) {
	if in == nil {
		return
	}
	var spec ZipkinSpec = map[string]any{}
	for key, val := range *in {
		spec[key] = val
	}
	*out = spec
}
