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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type clientConfig struct {
	restConfig rest.Config
}

func NewClientConfigForRestConfig(restConfig *rest.Config) clientcmd.ClientConfig {
	return &clientConfig{
		restConfig: *restConfig,
	}
}

func (c clientConfig) RawConfig() (api.Config, error) {
	cfg := api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Preferences:    api.Preferences{},
		Clusters:       map[string]*api.Cluster{},
		AuthInfos:      map[string]*api.AuthInfo{},
		Contexts:       map[string]*api.Context{},
		CurrentContext: "",
	}
	return cfg, nil
}

func (c clientConfig) ClientConfig() (*rest.Config, error) {
	return c.copyRestConfig(), nil
}

func (c *clientConfig) copyRestConfig() *rest.Config {
	out := c.restConfig
	return &out
}

func (c clientConfig) Namespace() (string, bool, error) {
	return "default", false, nil
}

func (c clientConfig) ConfigAccess() clientcmd.ConfigAccess {
	return nil
}
