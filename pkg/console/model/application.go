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
	"strconv"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/config/app"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ApplicationDetailReq struct {
	AppName string `form:"appName"`
	Mesh    string `form:"mesh"`
}

type ApplicationDetailResp struct {
	AppName          string   `json:"appName"`
	AppTypes         []string `json:"appTypes"`
	DeployClusters   []string `json:"deployClusters"`
	DubboPorts       []string `json:"dubboPorts"`
	DubboVersions    []string `json:"dubboVersions"`
	Images           []string `json:"images"`
	RegisterClusters []string `json:"registerClusters"`
	RegisterModes    []string `json:"registerModes"`
	RPCProtocols     []string `json:"rpcProtocols"`
	SerialProtocols  []string `json:"serialProtocols"`
	Workloads        []string `json:"workloads"`
}

func (r *ApplicationDetailResp) FromApplicationDetail(ad *ApplicationDetail) *ApplicationDetailResp {
	r.AppTypes = ad.AppTypes.Values()
	r.DeployClusters = ad.DeployClusters.Values()
	r.DubboPorts = ad.DubboPorts.Values()
	r.DubboVersions = ad.DubboVersions.Values()
	r.Images = ad.Images.Values()
	r.RegisterClusters = ad.RegisterClusters.Values()
	r.RegisterModes = ad.RegisterModes.Values()
	r.RPCProtocols = ad.RPCProtocols.Values()
	r.SerialProtocols = ad.SerialProtocols.Values()
	r.Workloads = ad.Workloads.Values()
	return r
}

type ApplicationDetail struct {
	AppTypes         Set
	DeployClusters   Set
	DubboPorts       Set
	DubboVersions    Set
	Images           Set
	RegisterClusters Set
	RegisterModes    Set
	RPCProtocols     Set
	SerialProtocols  Set
	Workloads        Set
}

func NewApplicationDetail() *ApplicationDetail {
	return &ApplicationDetail{
		AppTypes:         NewSet(),
		DeployClusters:   NewSet(),
		DubboPorts:       NewSet(),
		DubboVersions:    NewSet(),
		Images:           NewSet(),
		RegisterClusters: NewSet(),
		RegisterModes:    NewSet(),
		RPCProtocols:     NewSet(),
		SerialProtocols:  NewSet(),
		Workloads:        NewSet(),
	}
}
func (a *ApplicationDetail) MergeInstance(instanceRes *meshresource.InstanceResource, cfg app.AdminConfig) {
	instance := instanceRes.Spec
	if instance.WorkloadType == constants.StatefulSet {
		a.AppTypes.Add(constants.Stateful)
	} else {
		a.AppTypes.Add(constants.Stateless)
	}
	a.DubboPorts.Add(strconv.FormatInt(instance.RpcPort, 10))
	a.DubboVersions.Add(instance.ReleaseVersion)
	a.Images.Add(instance.Image)
	if d := cfg.FindDiscovery(instanceRes.Mesh); d != nil {
		a.RegisterClusters.Add(instanceRes.Mesh)
	}
	if cfg.Engine != nil && cfg.Engine.ID == instance.SourceEngine {
		a.DeployClusters.Add(cfg.Engine.Name)
	}
	a.RegisterModes.Add(constants.Application)
	a.RPCProtocols.Add(instance.Protocol)
	if strutil.IsNotBlank(instance.Serialization) {
		a.SerialProtocols.Add(instance.Serialization)
	} else if strutil.IsNotBlank(instance.PreferSerialization) {
		a.SerialProtocols.Add(instance.PreferSerialization)
	}
	a.Workloads.Add(instance.WorkloadName)
}

type ApplicationTabInstanceInfoReq struct {
	coremodel.PageReq

	AppName string `form:"appName"`
	Mesh    string `form:"mesh"`
}

func NewApplicationTabInstanceInfoReq() *ApplicationTabInstanceInfoReq {
	return &ApplicationTabInstanceInfoReq{
		PageReq: coremodel.PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type AppInstanceInfoResp struct {
	AppName         string                 `json:"appName"`
	CreateTime      string                 `json:"createTime"`
	LifecycleState  InstanceLifecycleState `json:"lifecycleState"`
	DeployState     InstanceDeployState    `json:"deployState"`
	DeployClusters  string                 `json:"deployClusters"`
	IP              string                 `json:"ip"`
	Labels          map[string]string      `json:"labels"`
	Name            string                 `json:"name"`
	RegisterCluster string                 `json:"registerCluster"`
	RegisterState   InstanceRegisterState  `json:"registerState"`
	RegisterTime    string                 `json:"registerTime"`
	WorkloadName    string                 `json:"workloadName"`
}

type ApplicationServiceReq struct {
	AppName string `json:"appName"`
}

type ApplicationServiceResp struct {
	ConsumeNum int64 `json:"consumeNum"`
	ProvideNum int64 `json:"provideNum"`
}

type ApplicationServiceFormReq struct {
	coremodel.PageReq

	AppName     string `form:"appName"`
	ServiceName string `form:"serviceName"`
	Side        string `form:"side"`
	Mesh        string `form:"mesh"`
}

func NewApplicationServiceFormReq() *ApplicationServiceFormReq {
	return &ApplicationServiceFormReq{
		PageReq: coremodel.PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ApplicationSearchReq struct {
	coremodel.PageReq

	AppName  string `form:"appName" json:"appName"`
	Keywords string `form:"keywords" json:"keywords"`
	Mesh     string `form:"mesh" json:"mesh"`
}

func NewApplicationSearchReq() *ApplicationSearchReq {
	return &ApplicationSearchReq{
		PageReq: coremodel.PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ApplicationGraphReq struct {
	coremodel.PageReq

	AppName  string `form:"appName" json:"appName"`
	Keywords string `form:"keywords" json:"keywords"`
	Mesh     string `form:"mesh" json:"mesh"`
}

func NewApplicationGraphReq() *ApplicationGraphReq {
	return &ApplicationGraphReq{
		PageReq: coremodel.PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ApplicationSearchResp struct {
	AppName          string   `json:"appName"`
	DeployClusters   []string `json:"deployClusters"`
	InstanceCount    int64    `json:"instanceCount"`
	RegistryClusters []string `json:"registryClusters"`
}

type FlowWeightSet struct {
	Weight int32        `json:"weight"`
	Scope  []ParamMatch `json:"scope,omitempty"`
}

type GraySet struct {
	EnvName string       `json:"name,omitempty"`
	Scope   []ParamMatch `json:"scope,omitempty"`
}

type AppAccessLogConfigResp struct {
	AccessLog bool `json:"operatorLog"`
}

type AppFlowWeightConfigResp struct {
	FlowWeightSets []FlowWeightSet `json:"flowWeightSets"`
}

type AppGrayConfigResp struct {
	GraySets []GraySet `json:"graySets"`
}
