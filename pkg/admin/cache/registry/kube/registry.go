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

package kube

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-admin/pkg/admin/cache/registry"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient/client"
)

func init() {
	registry.AddRegistry("kube", func(u *common.URL) (registry.AdminRegistry, error) {
		return NewRegistry()
	})
}

type Registry struct {
	client *client.KubeClient
}

func NewRegistry() (*Registry, error) {
	return nil, nil
}

func (kr *Registry) Delegate() dubboRegistry.Registry {
	return nil
}

func (kr *Registry) Subscribe(listener registry.AdminNotifyListener) error {
	return nil
}

func (kr *Registry) Destroy() error {
	return nil
}
