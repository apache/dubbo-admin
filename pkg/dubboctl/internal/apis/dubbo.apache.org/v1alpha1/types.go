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

type DubboOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              *DubboOperatorSpec `json:"spec,omitempty"`
}

func (do *DubboOperator) GetProfile() string {
	if do.Spec != nil {
		return do.Spec.Profile
	}
	return ""
}

type DubboOperatorSpec struct {
	ProfilePath    string               `json:"profilePath,omitempty"`
	Profile        string               `json:"profile,omitempty"`
	ChartPath      string               `json:"chartPath,omitempty"`
	ComponentsMeta *DubboComponentsMeta `json:"componentsMeta,omitempty"`
	Components     *DubboComponentsSpec `json:"components,omitempty"`
}

func (dos *DubboOperatorSpec) IsAdminEnabled() bool {
	if dos.ComponentsMeta != nil && dos.ComponentsMeta.IsAdminEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsGrafanaEnabled() bool {
	if dos.ComponentsMeta != nil && dos.ComponentsMeta.IsGrafanaEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsNacosEnabled() bool {
	if dos.ComponentsMeta != nil && dos.ComponentsMeta.IsNacosEnabled() {
		return true
	}
	return false
}

func (dos *DubboOperatorSpec) IsZookeeperEnabled() bool {
	if dos.ComponentsMeta != nil && dos.ComponentsMeta.IsZookeeperEnabled() {
		return true
	}
	return false
}

type DubboComponentsMeta struct {
	Admin     *AdminMeta     `json:"admin,omitempty"`
	Grafana   *GrafanaMeta   `json:"grafana,omitempty"`
	Nacos     *NacosMeta     `json:"nacos,omitempty"`
	Zookeeper *ZookeeperMeta `json:"zookeeper,omitempty"`
}

type BaseMeta struct {
	Enabled   bool   `json:"enabled,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type AdminMeta struct {
	BaseMeta
}

type GrafanaMeta struct {
	BaseMeta
}

type NacosMeta struct {
	BaseMeta
}

type ZookeeperMeta struct {
	BaseMeta
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

type DubboComponentsSpec struct {
	Admin     *AdminSpec     `json:"admin,omitempty"`
	Grafana   *GrafanaSpec   `json:"grafana,omitempty"`
	Nacos     *NacosSpec     `json:"nacos,omitempty"`
	Zookeeper *ZookeeperSpec `json:"zookeeper,omitempty"`
}

type AdminSpec struct {
	Image              *Image              `json:"image,omitempty"`
	Replicas           uint32              `json:"replicas"`
	Global             *Global             `json:"global,omitempty"`
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

type Global struct {
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

type GrafanaSpec struct{}

type NacosSpec struct{}

type ZookeeperSpec struct {
	Image                    *ZookeeperImage                   `json:"image,omitempty"`
	Replicas                 uint32                            `json:"replicas,omitempty"`
	Persistence              *Persistence                      `json:"persistence,omitempty"`
	Autoscaling              *Autoscaling                      `json:"autoscaling,omitempty"`
	Service                  *Service                          `json:"service,omitempty"`
	ContainerPorts           *ContainerPorts                   `json:"containerPorts,omitempty"`
	LivenessProbe            *Probe                            `json:"livenessProbe,omitempty"`
	ReadinessProbe           *Probe                            `json:"readinessProbe,omitempty"`
	StartupProbe             *BaseProbe                        `json:"startupProbe,omitempty"`
	CustomLivenessProbe      *Probe                            `json:"customLivenessProbe,omitempty"`
	CustomReadinessProbe     *Probe                            `json:"customReadinessProbe,omitempty"`
	CustomStartupProbe       *BaseProbe                        `json:"customStartupProbe,omitempty"`
	LifecycleHooks           *corev1.Lifecycle                 `json:"lifecycleHooks,omitempty"`
	Resources                *corev1.ResourceRequirements      `json:"resources,omitempty"`
	PodSecurityContext       *PodSecurityContext               `json:"podSecurityContext,omitempty"`
	ContainerSecurityContext *ContainerSecurityContext         `json:"containerSecurityContext,omitempty"`
	HostAliases              []*corev1.HostAlias               `json:"hostAliases,omitempty"`
	ExtraVolume              []*corev1.Volume                  `json:"extraVolume,omitempty"`
	ExtraVolumeMounts        []*corev1.VolumeMount             `json:"extraVolumeMounts,omitempty"`
	Auth                     *Auth                             `json:"auth,omitempty"`
	TickTime                 uint32                            `json:"tickTime,omitempty"`
	InitLimit                uint32                            `json:"initLimit,omitempty"`
	SyncLimit                uint32                            `json:"syncLimit,omitempty"`
	PreAllocSize             uint32                            `json:"preAllocSize,omitempty"`
	SnapCount                uint32                            `json:"snapCount,omitempty"`
	MaxClientCnxns           uint32                            `json:"maxClientCnxns,omitempty"`
	MaxSessionTimeout        uint32                            `json:"maxSessionTimeout,omitempty"`
	HeapSize                 uint32                            `json:"heapSize,omitempty"`
	FourlwCommandsWhitelist  []string                          `json:"fourlwCommandsWhitelist,omitempty"`
	MinServerId              uint32                            `json:"minServerId,omitempty"`
	ListenOnAllIPs           bool                              `json:"listenOnAllIPs,omitempty"`
	Autopurge                *Autopurge                        `json:"autopurge,omitempty"`
	LogLevel                 LogLevel                          `json:"logLevel,omitempty"`
	JvmFlags                 string                            `json:"jvmFlags,omitempty"`
	DataLogDir               string                            `json:"dataLogDir,omitempty"`
	Configuration            string                            `json:"configuration,omitempty"`
	ExistingConfigmap        string                            `json:"existingConfigmap,omitempty"`
	ClusterDomain            string                            `json:"clusterDomain,omitempty"`
	ExtraDeploy              []string                          `json:"extraDeploy,omitempty"`
	Labels                   map[string]string                 `json:"labels,omitempty"`
	Annotations              map[string]string                 `json:"annotations,omitempty"`
	DiagnosticMode           *DiagnosticMode                   `json:"diagnosticMode,omitempty"`
	PodDisruptionBudget      *policyv1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	Ingress                  *Ingress                          `json:"ingress,omitempty"`
	NetworkPolicy            *NetworkPolicy                    `json:"networkPolicy,omitempty"`
}

type ZookeeperImage struct {
	Repository    string            `json:"repository,omitempty"`
	Tag           string            `json:"tag,omitempty"`
	Digest        string            `json:"digest,omitempty"`
	Debug         bool              `json:"debug,omitempty"`
	PullPolicy    corev1.PullPolicy `json:"pullPolicy,omitempty"`
	NetworkPolicy *NetworkPolicy    `json:"networkPolicy,omitempty"`
}

type Persistence struct {
	Enabled       bool                                `json:"enabled,omitempty"`
	ExistingClaim string                              `json:"existingClaim,omitempty"`
	StorageClass  string                              `json:"storageClass,omitempty"`
	AccessModes   []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	Size          *resource.Quantity                  `json:"size,omitempty"`
	Labels        map[string]string                   `json:"labels,omitempty"`
	Selector      *metav1.LabelSelector               `json:"selector,omitempty"`
	DataLogDir    *DataLogDir                         `json:"dataLogDir,omitempty"`
}

type DataLogDir struct {
	Size          *resource.Quantity    `json:"size,omitempty"`
	ExistingClaim string                `json:"existingClaim,omitempty"`
	Selector      *metav1.LabelSelector `json:"selector,omitempty"`
}

type Service struct {
	Type                     corev1.ServiceType                      `json:"type,omitempty"`
	Ports                    *Ports                                  `json:"ports,omitempty"`
	NodePorts                *NodePorts                              `json:"nodePorts,omitempty"`
	DisableBaseClientPort    bool                                    `json:"disableBaseClientPort,omitempty"`
	SessionAffinity          corev1.ServiceAffinity                  `json:"sessionAffinity,omitempty"`
	SessionAffinityConfig    *corev1.SessionAffinityConfig           `json:"sessionAffinityConfig,omitempty"`
	ClusterIP                string                                  `json:"clusterIP,omitempty"`
	LoadBalancerIP           string                                  `json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges []string                                `json:"LoadBalancerSourceRanges,omitempty"`
	ExternalTrafficPolicy    corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
	Annotations              map[string]string                       `json:"annotations,omitempty"`
	ExtraPorts               []uint32                                `json:"extraPorts,omitempty"`
	Headless                 *Headless                               `json:"headless,omitempty"`
}

type Ports struct {
	Client   uint32 `json:"client,omitempty"`
	Follower uint32 `json:"follower,omitempty"`
	Election uint32 `json:"election,omitempty"`
}

type NodePorts struct {
	Client string `json:"client,omitempty"`
	Tls    string `json:"tls,omitempty"`
}

type Headless struct {
	PublishNotReadyAddresses bool              `json:"publishNotReadyAddresses,omitempty"`
	Annotations              map[string]string `json:"annotations,omitempty"`
	ServicenameOverride      string            `json:"servicenameOverride,omitempty"`
}

type ContainerPorts struct {
	Client   uint32 `json:"client,omitempty"`
	Tls      uint32 `json:"tls,omitempty"`
	Follower uint32 `json:"follower,omitempty"`
	Election uint32 `json:"election,omitempty"`
}

type BaseProbe struct {
	Enabled             bool   `json:"enabled,omitempty"`
	InitialDelaySeconds uint32 `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       uint32 `json:"periodSeconds,omitempty"`
	TimeoutSeconds      uint32 `json:"timeoutSeconds,omitempty"`
	FailureThreshold    uint32 `json:"failureThreshold,omitempty"`
	SuccessThreshold    uint32 `json:"successThreshold,omitempty"`
}

type Probe struct {
	BaseProbe
	ProbeCommandTimeout uint32 `json:"probeCommandTimeout,omitempty"`
}

type PodSecurityContext struct {
	Enabled bool   `json:"enabled,omitempty"`
	FsGroup uint64 `json:"fsGroup,omitempty"`
}

type ContainerSecurityContext struct {
	Enabled bool `json:"enabled,omitempty"`
	corev1.PodSecurityContext
	RunAsUser                uint64 `json:"runAsUser,omitempty"`
	RunAsNonRoot             bool   `json:"runAsNonRoot"`
	AllowPrivilegeEscalation bool   `json:"allowPrivilegeEscalation,omitempty"`
}

type Auth struct {
	Client *Client `json:"client,omitempty"`
	Quorum *Quorum `json:"quorum,omitempty"`
}

type Client struct {
	Enabled         bool   `json:"enabled,omitempty"`
	ClientUser      string `json:"clientUser,omitempty"`
	ClientPassword  string `json:"clientPassword,omitempty"`
	ServerUsers     string `json:"serverUsers,omitempty"`
	ServerPasswords string `json:"serverPasswords,omitempty"`
	ExistingSecret  string `json:"existingSecret,omitempty"`
}

type Quorum struct {
	Enabled         bool   `json:"enabled,omitempty"`
	LearnerUser     string `json:"LearnerUser,omitempty"`
	LearnerPassword string `json:"LearnerPassword,omitempty"`
	ServerUsers     string `json:"serverUsers,omitempty"`
	ServerPasswords string `json:"serverPasswords,omitempty"`
	ExistingSecret  string `json:"existingSecret,omitempty"`
}

type Autopurge struct {
	SnapRetainCount uint32 `json:"snapRetainCount,omitempty"`
	PurgeInterval   uint32 `json:"purgeInterval,omitempty"`
}

type LogLevel string

const (
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
	WARN  LogLevel = "WARN"
)

type DiagnosticMode struct {
	Enabled bool     `json:"enabled,omitempty"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type Ingress struct {
	Enabled     bool               `json:"enabled,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Path        string             `json:"path,omitempty"`
	PathType    netv1.PathType     `json:"pathType,omitempty"`
	Hosts       []string           `json:"hosts,omitempty"`
	ExtraPaths  []ExtraPath        `json:"extraPaths,omitempty"`
	Tls         []netv1.IngressTLS `json:"tls,omitempty"`
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
