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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	mesh "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ApplicationDetailReq struct {
	AppName string `form:"appName"`
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

func (a *ApplicationDetail) MergeMetaData(metadata *mesh.MetaDataResource) {
	a.mergeServiceInfo(metadata)
}

func (a *ApplicationDetail) mergeServiceInfo(metadata *mesh.MetaDataResource) {
	for _, serviceInfo := range metadata.Spec.Services {
		a.DubboVersions.Add(fmt.Sprintf("dubbo %s", serviceInfo.Params[constants.ReleaseKey]))
		a.RPCProtocols.Add(serviceInfo.Protocol)
		a.SerialProtocols.Add(serviceInfo.Params[constants.SerializationKey])

	}
}

func (a *ApplicationDetail) MergeDataplane(dataplane *mesh.DataplaneResource) {
	if work, ok := dataplane.Spec.Extensions[constants.WorkLoadKey]; ok &&
		regexp.MustCompile(`^.*-\d+$`).MatchString(work) {
		a.AppTypes.Add(constants.Stateful)
	} else {
		a.AppTypes.Add(constants.Stateless)
	}

	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.mergeInbound(inbound)
	}
	extensions := dataplane.Spec.Extensions
	a.mergeExtensions(extensions)
}

func (a *ApplicationDetail) mergeInbound(inbound *legacy.Dataplane_Networking_Inbound) {
	a.DubboPorts.Add(strconv.Itoa(int(inbound.Port)))
	a.DeployClusters.Add(inbound.Tags[legacy.ZoneTag])
}

func (a *ApplicationDetail) mergeExtensions(extensions map[string]string) {
	a.Images.Add(extensions[coremodel.ExtensionsImageKey])
	a.Workloads.Add(extensions[coremodel.ExtensionsWorkLoadKey])
}

func (a *ApplicationDetail) GetRegistry(ctx consolectx.Context) {
	// TODO

}

// todo Application instance info

type ApplicationTabInstanceInfoReq struct {
	AppName string `form:"appName"`
	PageReq
}

func NewApplicationTabInstanceInfoReq() *ApplicationTabInstanceInfoReq {
	return &ApplicationTabInstanceInfoReq{
		PageReq: PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ApplicationTabInstanceInfoResp struct {
	AppName         string            `json:"appName"`
	CreateTime      string            `json:"createTime"`
	DeployState     string            `json:"deployState"`
	DeployClusters  string            `json:"deployClusters"`
	IP              string            `json:"ip"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	RegisterCluster string            `json:"registerCluster"`
	RegisterState   string            `json:"registerState"`
	RegisterTime    string            `json:"registerTime"`
	WorkloadName    string            `json:"workloadName"`
}

type ByApplicationInstanceName []*ApplicationTabInstanceInfoResp

func (a ByApplicationInstanceName) Len() int { return len(a) }

func (a ByApplicationInstanceName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

func (a ByApplicationInstanceName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a *ApplicationTabInstanceInfoResp) FromDataplaneResource(dataplane *mesh.DataplaneResource) *ApplicationTabInstanceInfoResp {
	// TODO: support more fields
	extensions := dataplane.Spec.Extensions
	a.mergeExtension(extensions)
	a.mergeMainDataplane(dataplane)
	return a
}

// nolint
func (a *ApplicationTabInstanceInfoResp) mergeMainDataplane(dataplane *mesh.DataplaneResource) {
	a.AppName = dataplane.GetMeta().GetLabels()[v1alpha1.Application]
	a.CreateTime = dataplane.Meta.GetCreationTime().String()
	a.IP = dataplane.Spec.Networking.Address
	a.DeployClusters = dataplane.Spec.Networking.Inbound[0].Tags[legacy.ZoneTag]
	a.Labels = dataplane.GetMeta().GetLabels()
	a.Name = dataplane.Meta.GetName()
	a.RegisterTime = a.CreateTime
	if a.RegisterTime != "" {
		a.RegisterState = "Registed"
	} else {
		a.RegisterState = "UnRegisted"
	}
}

func (a *ApplicationTabInstanceInfoResp) mergeExtension(extensions map[string]string) {
	a.WorkloadName = extensions[coremodel.ExtensionsWorkLoadKey]
	a.DeployState = extensions[coremodel.ExtensionsPodPhaseKey]
}

func (a *ApplicationTabInstanceInfoResp) GetRegistry(ctx consolectx.Context) {
	// TODO

}

// Todo Application Service

type ApplicationServiceReq struct {
	AppName string `json:"appName"`
}

type ApplicationServiceResp struct {
	ConsumeNum int64 `json:"consumeNum"`
	ProvideNum int64 `json:"provideNum"`
}

type ApplicationServiceFormReq struct {
	AppName string `form:"appName"`
	Side    string `form:"side"`
	PageReq
}

func NewApplicationServiceFormReq() *ApplicationServiceFormReq {
	return &ApplicationServiceFormReq{
		PageReq: PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ApplicationServiceFormResp struct {
	ServiceName   string         `json:"serviceName"`
	VersionGroups []versionGroup `json:"versionGroups"`
}

type ByAppServiceFormName []*ApplicationServiceFormResp

func (a ByAppServiceFormName) Len() int { return len(a) }

func (a ByAppServiceFormName) Less(i, j int) bool {
	return a[i].ServiceName < a[j].ServiceName
}

func (a ByAppServiceFormName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type versionGroup struct {
	Group   string `json:"group"`
	Version string `json:"version"`
}

func NewApplicationServiceFormResp() *ApplicationServiceFormResp {
	return &ApplicationServiceFormResp{
		ServiceName:   "",
		VersionGroups: nil,
	}
}

func (a *ApplicationServiceFormResp) FromApplicationServiceForm(form *ApplicationServiceForm) error {
	a.ServiceName = form.ServiceName
	versionGroupList := make([]versionGroup, 0)
	for _, gv := range form.VersionGroups.Values() {
		var versionGroupInfo versionGroup
		if err := json.Unmarshal([]byte(gv), &versionGroupInfo); err != nil {
			return err
		}
		versionGroupList = append(versionGroupList, versionGroupInfo)
	}
	a.VersionGroups = versionGroupList
	return nil
}

type ApplicationServiceForm struct {
	ServiceName   string
	VersionGroups Set
}

func NewApplicationServiceForm(serviceName string) *ApplicationServiceForm {
	return &ApplicationServiceForm{
		ServiceName:   serviceName,
		VersionGroups: NewSet(),
	}
}

func (a *ApplicationServiceForm) FromServiceInfo(serviceInfo *v1alpha1.ServiceInfo) error {
	versionGroupInfo := versionGroup{
		Group:   serviceInfo.Group,
		Version: serviceInfo.Version,
	}
	bytes, err := json.Marshal(versionGroupInfo)
	if err != nil {
		return err
	}
	a.VersionGroups.Add(string(bytes))
	return nil
}

type ApplicationSearchReq struct {
	AppName  string `form:"appName" json:"appName"`
	Keywords string `form:"keywords" json:"keywords"`
	PageReq
}

func NewApplicationSearchReq() *ApplicationSearchReq {
	return &ApplicationSearchReq{
		PageReq: PageReq{
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

func (a *ApplicationSearchResp) FromApplicationSearch(applicationSearch *ApplicationSearch) *ApplicationSearchResp {
	a.RegistryClusters = applicationSearch.RegistryClusters.Values()
	a.InstanceCount = applicationSearch.InstanceCount
	a.DeployClusters = applicationSearch.DeployClusters.Values()
	return a
}

type ByAppName []*ApplicationSearchResp

func (a ByAppName) Len() int { return len(a) }

func (a ByAppName) Less(i, j int) bool {
	return a[i].AppName < a[j].AppName
}

func (a ByAppName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Todo Application Search

type ApplicationSearch struct {
	AppName          string
	DeployClusters   Set
	InstanceCount    int64
	RegistryClusters Set
}

func NewApplicationSearch(appName string) *ApplicationSearch {
	return &ApplicationSearch{
		AppName:          appName,
		DeployClusters:   NewSet(),
		InstanceCount:    0,
		RegistryClusters: NewSet(),
	}
}

func (a *ApplicationSearch) MergeDataplane(dataplane *mesh.DataplaneResource) {
	a.InstanceCount++

	// merge inbounds
	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.DeployClusters.Add(inbound.Tags[legacy.ZoneTag])
	}
}

func (a *ApplicationSearch) GetRegistry(ctx consolectx.Context) {

	// TODO
}

type FlowWeightSet struct {
	Weight int32        `json:"weight"`
	Scope  []ParamMatch `json:"scope,omitempty"`
}

type GraySet struct {
	EnvName string       `json:"name,omitempty"`
	Scope   []ParamMatch `json:"scope,omitempty"`
}
