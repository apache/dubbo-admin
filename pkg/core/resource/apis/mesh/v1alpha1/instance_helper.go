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

package v1alpha1

import (
	"fmt"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/constants"
)

func BuildInstanceResName(appName string, ip string, rpcPort int64) string {
	return fmt.Sprintf("%s%s:%d", appName, ip, rpcPort)
}

// FromRPCInstance create an instance resource from a rpc instance resource
func FromRPCInstance(rpcInstanceRes *RPCInstanceResource) *InstanceResource {
	resName := BuildInstanceResName(rpcInstanceRes.Spec.AppName, rpcInstanceRes.Spec.Ip, rpcInstanceRes.Spec.Port)
	instanceRes := NewInstanceResourceWithAttributes(resName, rpcInstanceRes.Mesh)
	instanceRes.Spec.Name = rpcInstanceRes.Spec.Name
	instanceRes.Spec.AppName = rpcInstanceRes.Spec.AppName
	instanceRes.Spec.Ip = rpcInstanceRes.Spec.Ip
	instanceRes.Spec.RpcPort = rpcInstanceRes.Spec.Port
	MergeRPCInstanceIntoInstance(rpcInstanceRes, instanceRes)
	return instanceRes
}

// MergeRPCInstanceIntoInstance merge rpc instance resource into instance resource
func MergeRPCInstanceIntoInstance(
	rpcInstanceRes *RPCInstanceResource,
	instanceRes *InstanceResource) {
	instanceRes.Spec.ReleaseVersion = rpcInstanceRes.Spec.ReleaseVersion
	instanceRes.Spec.RegisterTime = rpcInstanceRes.Spec.RegisterTime
	instanceRes.Spec.UnregisterTime = rpcInstanceRes.Spec.UnregisterTime
	instanceRes.Spec.Protocol = rpcInstanceRes.Spec.Protocol
	instanceRes.Spec.Serialization = rpcInstanceRes.Spec.Serialization
	instanceRes.Spec.PreferSerialization = rpcInstanceRes.Spec.PreferSerialization
	instanceRes.Spec.Tags = rpcInstanceRes.Spec.Tags
}

// FromRuntimeInstance create an instance resource from a runtime instance resource
func FromRuntimeInstance(rtInstanceRes *RuntimeInstanceResource) *InstanceResource {
	resName := BuildInstanceResName(rtInstanceRes.Spec.AppName, rtInstanceRes.Spec.Ip, rtInstanceRes.Spec.RpcPort)
	instanceRes := NewInstanceResourceWithAttributes(resName, rtInstanceRes.Mesh)
	instanceRes.Spec.Name = resName
	instanceRes.Spec.AppName = rtInstanceRes.Spec.AppName
	instanceRes.Spec.Ip = rtInstanceRes.Spec.Ip
	instanceRes.Spec.RpcPort = rtInstanceRes.Spec.RpcPort
	MergeRuntimeInstanceIntoInstance(rtInstanceRes, instanceRes)
	return instanceRes
}

// MergeRuntimeInstanceIntoInstance merge runtime instance resource into instance resource
func MergeRuntimeInstanceIntoInstance(
	rtInstanceRes *RuntimeInstanceResource,
	instanceRes *InstanceResource) {
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
	instanceRes.Spec.SourceEngine = rtInstanceRes.Spec.SourceEngine
}

func ClearRPCInstanceFromInstance(instanceRes *InstanceResource) {
	if instanceRes == nil || instanceRes.Spec == nil {
		return
	}
	instanceRes.Spec.ReleaseVersion = ""
	instanceRes.Spec.RegisterTime = ""
	instanceRes.Spec.UnregisterTime = time.Now().Format(constants.TimeFormatStr)
	instanceRes.Spec.Protocol = ""
	instanceRes.Spec.Serialization = ""
	instanceRes.Spec.PreferSerialization = ""
	instanceRes.Spec.Tags = nil
}

func ClearRuntimeInstanceFromInstance(instanceRes *InstanceResource) {
	if instanceRes == nil || instanceRes.Spec == nil {
		return
	}
	instanceRes.Labels = nil
	instanceRes.Spec.Image = ""
	instanceRes.Spec.CreateTime = ""
	instanceRes.Spec.StartTime = ""
	instanceRes.Spec.ReadyTime = ""
	instanceRes.Spec.DeployState = ""
	instanceRes.Spec.WorkloadType = ""
	instanceRes.Spec.WorkloadName = ""
	instanceRes.Spec.Node = ""
	instanceRes.Spec.Probes = nil
	instanceRes.Spec.Conditions = nil
	instanceRes.Spec.SourceEngine = ""
}

func HasRuntimeInstanceSource(instanceRes *InstanceResource) bool {
	if instanceRes == nil || instanceRes.Spec == nil {
		return false
	}
	return instanceRes.Spec.SourceEngine != "" ||
		instanceRes.Spec.DeployState != "" ||
		instanceRes.Spec.WorkloadName != "" ||
		instanceRes.Spec.Node != "" ||
		instanceRes.Spec.Image != "" ||
		instanceRes.Spec.StartTime != "" ||
		instanceRes.Spec.ReadyTime != "" ||
		len(instanceRes.Spec.Conditions) > 0
}

func HasRPCInstanceSource(instanceRes *InstanceResource) bool {
	if instanceRes == nil || instanceRes.Spec == nil {
		return false
	}
	return instanceRes.Spec.RegisterTime != ""
}
