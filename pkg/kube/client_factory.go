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
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	diskcached "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/discovery/cached/memory"

	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/apache/dubbo-admin/pkg/common/lazy"

	// "k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// clientFactory partially implements the kubectl util.Factory, which is provides access to various k8s clients.
// The full Factory can be built with MakeKubeFactory.
// This split is to avoid huge dependencies.
type clientFactory struct {
	clientConfig    clientcmd.ClientConfig
	expander        lazy.Lazy[meta.RESTMapper]
	mapper          lazy.Lazy[meta.ResettableRESTMapper]
	discoveryClient lazy.Lazy[discovery.CachedDiscoveryInterface]
}

// newClientFactory creates a new util.Factory from the given clientcmd.ClientConfig.
func newClientFactory(clientConfig clientcmd.ClientConfig, diskCache bool) *clientFactory {
	cf := &clientFactory{
		clientConfig: clientConfig,
	}
	cf.discoveryClient = lazy.NewWithRetry(func() (discovery.CachedDiscoveryInterface, error) {
		restConfig, err := cf.ToRestConfig()
		if err != nil {
			return nil, err
		}
		// Setup cached discoveryengine. CLIs uses disk cache, controllers use memory cache.
		if diskCache {
			cacheDir := filepath.Join(homedir.HomeDir(), ".kube", "cache")
			httpCacheDir := filepath.Join(cacheDir, "http")
			discoveryCacheDir := computeDiscoverCacheDir(filepath.Join(cacheDir, "discoveryengine"), restConfig.Host)
			return diskcached.NewCachedDiscoveryClientForConfig(restConfig, discoveryCacheDir, httpCacheDir, 6*time.Hour)
		}
		d, err := discovery.NewDiscoveryClientForConfig(restConfig)
		if err != nil {
			return nil, err
		}
		return memory.NewMemCacheClient(d), nil
	})
	cf.mapper = lazy.NewWithRetry(func() (meta.ResettableRESTMapper, error) {
		discoveryClient, err := cf.ToDiscoveryClient()
		if err != nil {
			return nil, err
		}
		return restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient), nil
	})
	cf.expander = lazy.NewWithRetry(func() (meta.RESTMapper, error) {
		discoveryClient, err := cf.discoveryClient.Get()
		if err != nil {
			return nil, err
		}
		mapper, err := cf.mapper.Get()
		if err != nil {
			return nil, err
		}
		return restmapper.NewShortcutExpander(mapper, discoveryClient, func(string) {}), nil
	})
	return cf
}

func (c *clientFactory) RestClient() (*rest.RESTClient, error) {
	clientConfig, err := c.ToRestConfig()
	if err != nil {
		return nil, err
	}
	return rest.RESTClientFor(clientConfig)
}

func (c *clientFactory) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return c.discoveryClient.Get()
}

func (c *clientFactory) ToRestConfig() (*rest.Config, error) {
	restConfig, err := c.clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(restConfig), nil
}

// overlyCautiousIllegalFileCharacters matches characters that *might* not be supported.  Windows is really restrictive, so this is really restrictive
var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/.)]`)

func computeDiscoverCacheDir(dir, host string) string {
	schemelesshost := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	safehost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelesshost, "_")
	return filepath.Join(dir, safehost)
}
