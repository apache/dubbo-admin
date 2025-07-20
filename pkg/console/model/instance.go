/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	gxset "github.com/dubbogo/gost/container/set"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type SearchInstanceReq struct {
	AppName  string `form:"appName"`
	Keywords string `form:"keywords"`
	PageReq
}

func NewSearchInstanceReq() *SearchInstanceReq {
	return &SearchInstanceReq{
		PageReq: PageReq{PageSize: 15},
	}
}

type InstanceDetailReq struct {
	InstanceName string `form:"instanceName"`
}

type SearchPaginationResult struct {
	List     any                   `json:"list"`
	PageInfo *coremodel.Pagination `json:"pageInfo"`
}

func NewSearchPaginationResult() *SearchPaginationResult {
	return &SearchPaginationResult{}
}

type SearchInstanceResp struct {
	Ip                  string            `json:"ip"`
	Name                string            `json:"name"`
	WorkloadName        string            `json:"workloadName"`
	AppName             string            `json:"appName"`
	DeployState         string            `json:"deployState"`
	DeployCluster       string            `json:"deployCluster"`
	RegisterState       string            `json:"registerState"`
	RegisterClustersSet *gxset.HashSet    `json:"-"`
	RegisterClusters    []string          `json:"registerClusters"`
	CreateTime          string            `json:"createTime"`
	RegisterTime        string            `json:"registerTime"` // TODO: not converted
	Labels              map[string]string `json:"labels"`
}

func NewSearchInstanceResp() *SearchInstanceResp {
	return &SearchInstanceResp{
		RegisterClustersSet: gxset.NewSet(),
		RegisterClusters:    make([]string, 0),
	}
}

func (r *SearchInstanceResp) FromDataplaneResource(dr *mesh.DataplaneResource) *SearchInstanceResp {
	// TODO: support more fields
	r.Ip = dr.GetIP()
	meta := dr.GetMeta()
	r.Name = meta.GetName()
	r.CreateTime = meta.GetCreationTime().String()
	r.RegisterTime = r.CreateTime // TODO: separate createTime and RegisterTime
	cluster := dr.Spec.Networking.Inbound[0].Tags[legacy.ZoneTag]
	r.RegisterClustersSet.Add(cluster)
	for _, c := range r.RegisterClustersSet.Values() {
		r.RegisterClusters = append(r.RegisterClusters, c.(string))
	}
	r.DeployCluster = cluster
	if r.RegisterTime != "" {
		r.RegisterState = "Registed"
	} else {
		r.RegisterState = "UnRegisted"
	}
	// label conversion
	r.Labels = meta.GetLabels()
	// spec conversion
	spec := dr.Spec
	{
		statusValue := spec.Extensions[coremodel.ExtensionsPodPhaseKey]
		if v, ok := spec.Extensions[coremodel.ExtensionsPodStatusKey]; ok {
			statusValue = v
		}
		if v, ok := spec.Extensions[coremodel.ExtensionsContainerStatusReasonKey]; ok {
			statusValue = v
		}
		r.DeployState = statusValue
		r.WorkloadName = spec.Extensions[coremodel.ExtensionsWorkLoadKey]
		// name field source is different between universal and k8s mode
		r.AppName = spec.Extensions[meshproto.Application]
		if r.AppName == "" {
			for _, inbound := range spec.Networking.Inbound {
				r.AppName = inbound.Tags[legacy.AppTag]
			}
		}
	}
	return r
}

type State struct {
	Label string `json:"label"`
	Level string `json:"level"`
	Tip   string `json:"tip"`
	Value string `json:"value"`
}

type InstanceDetailResp struct {
	RpcPort          int               `json:"rpcPort"`
	Ip               string            `json:"ip"`
	AppName          string            `json:"appName"`
	WorkloadName     string            `json:"workloadName"`
	Labels           map[string]string `json:"labels"`
	CreateTime       string            `json:"createTime"`
	ReadyTime        string            `json:"readyTime"`
	RegisterTime     string            `json:"registerTime"`
	RegisterClusters []string          `json:"registerClusters"`
	DeployCluster    string            `json:"deployCluster"`
	DeployState      string            `json:"deployState"`
	RegisterState    string            `json:"registerState"`
	Node             string            `json:"node"`
	Image            string            `json:"image"`
	Probes           ProbeStruct       `json:"probes"`
	Tags             map[string]string `json:"tags"`
}

type ProbeStruct struct {
	StartupProbe   StartupProbe   `json:"startupProbe"`
	ReadinessProbe ReadinessProbe `json:"readinessProbe"`
	LivenessProbe  LivenessProbe  `json:"livenessProbe"`
}

type StartupProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
}
type ReadinessProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
}
type LivenessProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
}

func (r *InstanceDetailResp) FromInstanceDetail(id *InstanceDetail) *InstanceDetailResp {
	r.AppName = id.AppName
	r.RpcPort = id.RpcPort
	r.Ip = id.Ip
	r.WorkloadName = id.WorkloadName
	r.Labels = id.Labels
	r.CreateTime = id.CreateTime
	r.ReadyTime = id.ReadyTime
	r.RegisterTime = id.RegisterTime
	r.RegisterClusters = id.RegisterClusters.Values()
	r.DeployCluster = id.DeployCluster
	r.DeployCluster = id.DeployCluster
	r.DeployState = id.DeployState
	r.Node = id.Node
	r.Image = id.Image
	r.Tags = id.Tags
	r.RegisterState = id.RegisterState
	r.Probes = id.Probes
	return r
}

type InstanceDetail struct {
	RpcPort          int
	Ip               string
	AppName          string
	WorkloadName     string
	Labels           map[string]string
	CreateTime       string
	ReadyTime        string
	RegisterTime     string
	RegisterState    string
	RegisterClusters Set
	DeployCluster    string
	DeployState      string
	Node             string
	Image            string
	Tags             map[string]string
	Probes           ProbeStruct
}

func NewInstanceDetail() *InstanceDetail {
	return &InstanceDetail{
		RpcPort:          -1,
		Ip:               "",
		AppName:          "",
		WorkloadName:     "",
		Labels:           nil,
		CreateTime:       "",
		ReadyTime:        "",
		RegisterTime:     "",
		RegisterClusters: NewSet(),
		DeployCluster:    "",
		Node:             "",
		Image:            "",
	}
}

func (a *InstanceDetail) Merge(dataplane *mesh.DataplaneResource) {
	// TODO: support more fields
	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.mergeInbound(inbound)
	}
	meta := dataplane.Meta
	a.mergeMeta(meta)
	extensions := dataplane.Spec.Extensions
	a.mergeExtensions(extensions)
	probes := dataplane.Spec.Probes
	a.mergeProbes(probes)

	a.Ip = dataplane.GetIP()
	if a.RegisterTime != "" {
		a.RegisterState = "Registed"
	} else {
		a.RegisterState = "UnRegisted"
	}
}

func (a *InstanceDetail) mergeInbound(inbound *legacy.Dataplane_Networking_Inbound) {
	a.RpcPort = int(inbound.Port)
	a.RegisterClusters.Add(inbound.Tags[legacy.ZoneTag])
	for _, deployCluster := range a.RegisterClusters.Values() {
		a.DeployCluster = deployCluster // TODO: separate deployCluster and registerCluster
	}
	a.Tags = inbound.Tags
	if a.AppName == "" {
		a.AppName = inbound.Tags[legacy.AppTag]
	}
}

func (a *InstanceDetail) mergeExtensions(extensions map[string]string) {
	image := extensions[coremodel.ExtensionsImageKey]
	a.Image = image
	if a.AppName == "" {
		a.AppName = extensions[meshproto.Application]
	}
	a.WorkloadName = extensions[coremodel.ExtensionsWorkLoadKey]
	a.DeployState = extensions[coremodel.ExtensionsPodPhaseKey]
	a.Node = extensions[coremodel.ExtensionsNodeNameKey]
}

func (a *InstanceDetail) mergeMeta(meta coremodel.ResourceMeta) {
	a.CreateTime = meta.GetCreationTime().String()
	a.RegisterTime = meta.GetModificationTime().String() // Not sure if it's the right field
	a.ReadyTime = a.RegisterTime
	// TODO: separate createTime , RegisterTime and ReadyTime
	a.Labels = meta.GetLabels()
}

func (a *InstanceDetail) mergeProbes(probes *legacy.Dataplane_Probes) {
	if probes == nil {
		return
	}
	portStartup := probes.Endpoints[0].InboundPort
	portReadiness := probes.Endpoints[1].InboundPort
	portLiveness := probes.Endpoints[2].InboundPort
	a.Probes = ProbeStruct{
		StartupProbe: StartupProbe{
			Type: "HTTP", // TODO: support more scheme

			Port: int(portStartup),
			Open: true,
		},
		ReadinessProbe: ReadinessProbe{
			Type: "HTTP", // TODO: support more scheme
			Port: int(portReadiness),
			Open: true,
		},
		LivenessProbe: LivenessProbe{
			Type: "HTTP", // TODO: support more scheme
			Port: int(portLiveness),
			Open: true,
		},
	}
}
