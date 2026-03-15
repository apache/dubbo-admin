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
	"github.com/duke-git/lancet/v2/strutil"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/config/app"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type SearchInstanceReq struct {
	coremodel.PageReq

	AppName  string `form:"appName"`
	Keywords string `form:"keywords"`
	Mesh     string `form:"mesh"`
}

func NewSearchInstanceReq() *SearchInstanceReq {
	return &SearchInstanceReq{
		PageReq: coremodel.PageReq{PageSize: 15},
	}
}

type InstanceDetailReq struct {
	InstanceName string `form:"instanceName"`
	Mesh         string `form:"mesh"`
}

type SearchPaginationResult struct {
	List     any                  `json:"list"`
	PageInfo coremodel.Pagination `json:"pageInfo"`
}

func NewSearchPaginationResult() *SearchPaginationResult {
	return &SearchPaginationResult{}
}

type SearchInstanceResp struct {
	Ip               string            `json:"ip"`
	Name             string            `json:"name"`
	WorkloadName     string            `json:"workloadName"`
	AppName          string            `json:"appName"`
	LifecycleState   string            `json:"lifecycleState"`
	DeployState      string            `json:"deployState"`
	DeployCluster    string            `json:"deployCluster"`
	RegisterState    string            `json:"registerState"`
	RegisterClusters []string          `json:"registerClusters"`
	CreateTime       string            `json:"createTime"`
	RegisterTime     string            `json:"registerTime"`
	Labels           map[string]string `json:"labels"`
}

func NewSearchInstanceResp() *SearchInstanceResp {
	return &SearchInstanceResp{
		RegisterClusters: make([]string, 0),
	}
}

func (r *SearchInstanceResp) FromInstanceResource(instanceResource *meshresource.InstanceResource, cfg app.AdminConfig) *SearchInstanceResp {
	instance := instanceResource.Spec
	r.Ip = instance.Ip
	r.Name = instance.Name
	r.CreateTime = instance.CreateTime
	r.RegisterTime = instance.RegisterTime
	if d := cfg.FindDiscovery(instanceResource.Mesh); d != nil {
		r.RegisterClusters = []string{d.Name}
	}
	if cfg.Engine != nil && cfg.Engine.ID == instance.SourceEngine {
		r.DeployCluster = cfg.Engine.Name
	}
	r.RegisterState = DeriveInstanceRegisterState(instance)
	r.Labels = instance.Tags
	r.DeployState = DeriveInstanceDeployState(instance)
	r.LifecycleState = DeriveInstanceLifecycleState(instance, r.DeployState, r.RegisterState)
	r.WorkloadName = instance.WorkloadName
	r.AppName = instance.AppName
	return r
}

type State struct {
	Label string `json:"label"`
	Level string `json:"level"`
	Tip   string `json:"tip"`
	Value string `json:"value"`
}

type InstanceDetailResp struct {
	RpcPort          int64             `json:"rpcPort"`
	Ip               string            `json:"ip"`
	AppName          string            `json:"appName"`
	WorkloadName     string            `json:"workloadName"`
	Labels           map[string]string `json:"labels"`
	CreateTime       string            `json:"createTime"`
	ReadyTime        string            `json:"readyTime"`
	RegisterTime     string            `json:"registerTime"`
	RegisterClusters []string          `json:"registerClusters"`
	DeployCluster    string            `json:"deployCluster"`
	LifecycleState   string            `json:"lifecycleState"`
	DeployState      string            `json:"deployState"`
	RegisterState    string            `json:"registerState"`
	Node             string            `json:"node"`
	Image            string            `json:"image"`
	Probes           ProbeStruct       `json:"probes"`
	Tags             map[string]string `json:"tags"`
}

const (
	StartupProbeType   = "startup"
	ReadinessProbeType = "readiness"
	LivenessProbeType  = "liveness"
)

type ProbeStruct struct {
	StartupProbe   Probe `json:"startupProbe"`
	ReadinessProbe Probe `json:"readinessProbe"`
	LivenessProbe  Probe `json:"livenessProbe"`
}

type Probe struct {
	Type string `json:"type"`
	Port int32  `json:"port"`
	Open bool   `json:"open"`
}

func FromInstanceResource(res *meshresource.InstanceResource, cfg app.AdminConfig) *InstanceDetailResp {
	r := &InstanceDetailResp{}
	instance := res.Spec
	r.RpcPort = instance.RpcPort
	r.Ip = instance.Ip
	r.AppName = instance.AppName
	r.WorkloadName = instance.WorkloadName
	r.Labels = instance.Tags
	r.CreateTime = instance.CreateTime
	r.ReadyTime = instance.ReadyTime
	r.RegisterTime = instance.RegisterTime
	if d := cfg.FindDiscovery(res.Mesh); d != nil {
		r.RegisterClusters = []string{d.Name}
	}
	if cfg.Engine.ID == res.Spec.SourceEngine {
		r.DeployCluster = cfg.Engine.Name
	}
	r.DeployState = DeriveInstanceDeployState(instance)
	r.RegisterState = DeriveInstanceRegisterState(instance)
	r.LifecycleState = DeriveInstanceLifecycleState(instance, r.DeployState, r.RegisterState)
	r.Node = instance.Node
	r.Image = instance.Image
	r.Probes = ProbeStruct{}
	for _, p := range instance.Probes {
		switch p.Type {
		case StartupProbeType:
			r.Probes.StartupProbe = Probe{
				Type: StartupProbeType,
				Port: p.Port,
				Open: true,
			}
		case ReadinessProbeType:
			r.Probes.ReadinessProbe = Probe{
				Type: ReadinessProbeType,
				Port: p.Port,
				Open: true,
			}
		case LivenessProbeType:
			r.Probes.LivenessProbe = Probe{
				Type: LivenessProbeType,
				Port: p.Port,
				Open: true,
			}
		}

	}
	return r
}

func DeriveInstanceDeployState(instance *meshproto.Instance) string {
	if instance == nil || strutil.IsBlank(instance.DeployState) {
		return "Unknown"
	}
	switch instance.DeployState {
	case "Running":
		if !isPodReady(instance) {
			return "Starting"
		}
		return "Running"
	default:
		return instance.DeployState
	}
}

func DeriveInstanceRegisterState(instance *meshproto.Instance) string {
	if instance == nil || strutil.IsBlank(instance.RegisterTime) {
		return "UnRegistered"
	}
	return "Registered"
}

func DeriveInstanceLifecycleState(instance *meshproto.Instance, deployState string, registerState string) string {
	switch deployState {
	case "Crashing", "Failed", "Unknown", "Succeeded":
		return "Error"
	case "Terminating":
		return "Terminating"
	}

	if registerState == "Registered" {
		if deployState == "Running" {
			return "Serving"
		}
		return "Error"
	}

	if deployState == "Running" && strutil.IsNotBlank(instance.UnregisterTime) {
		return "Draining"
	}

	switch deployState {
	case "Pending", "Starting", "Running":
		return "Starting"
	default:
		return "Unknown"
	}
}

func isPodReady(instance *meshproto.Instance) bool {
	for _, condition := range instance.Conditions {
		if condition == nil {
			continue
		}
		if condition.Type == "Ready" {
			return condition.Status == "True"
		}
	}
	return false
}
