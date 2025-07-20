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

package manifest

const (
	// OwningResourceName represents the name of the owner to which the resource relates.
	OwningResourceName = "install.operator.dubbo.io/owning-resource"
	// OwningResourceNamespace represents the namespace of the owner to which the resource relates.
	OwningResourceNamespace = "install.operator.dubbo.io/owning-resource-namespace"
	// DubboComponentLabel indicates which Dubbo component a resource belongs to.
	DubboComponentLabel = "operator.dubbo.io/component"
	// OwningResourceNotPruned indicates that the resource should not be pruned during reconciliation cycles,
	// note this will not prevent the resource from being deleted if the owning resource is deleted.
	OwningResourceNotPruned = "install.operator.dubbo.io/owning-resource-not-pruned"
)
