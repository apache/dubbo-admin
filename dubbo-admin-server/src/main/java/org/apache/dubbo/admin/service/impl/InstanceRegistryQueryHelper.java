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

package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.domain.RegistrySource;
import org.apache.dubbo.common.URLBuilder;
import org.apache.dubbo.common.constants.CommonConstants;
import org.apache.dubbo.common.url.component.ServiceConfigURL;
import org.apache.dubbo.common.utils.CollectionUtils;
import org.apache.dubbo.metadata.MetadataInfo;
import org.apache.dubbo.registry.client.InstanceAddressURL;
import org.apache.dubbo.registry.client.ServiceInstance;
import org.apache.dubbo.rpc.RpcContext;

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import org.springframework.stereotype.Component;

import java.util.Collection;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentMap;
import java.util.stream.Collectors;

@Component
public class InstanceRegistryQueryHelper {

    private final InstanceRegistryCache instanceRegistryCache;

    public InstanceRegistryQueryHelper(InstanceRegistryCache instanceRegistryCache) {
        this.instanceRegistryCache = instanceRegistryCache;
    }


    public Set<String> findServices() {
        Set<String> services = Sets.newHashSet();
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null) {
            return services;
        }
        appInterfaceMap.values().forEach(serviceUrlMap ->
                serviceUrlMap.forEach((service, urls) -> {
                    if (CollectionUtils.isNotEmpty(urls)) {
                        services.add(service);
                    }
                }));
        return services;
    }

    public Set<String> findApplications() {
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null) {
            return Sets.newHashSet();
        }
        return Sets.newHashSet(appInterfaceMap.keySet());
    }

    public List<Provider> findByService(String serviceName) {
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null) {
            return Lists.newArrayList();
        }
        List<InstanceAddressURL> providerUrls = Lists.newArrayList();
        appInterfaceMap.values().forEach(serviceUrlMap ->
                serviceUrlMap.forEach((service, urls) -> {
                    if (CollectionUtils.isNotEmpty(urls) && service.equals(serviceName)) {
                        providerUrls.addAll(urls);
                    }
                }));
        return urlsToProviderList(providerUrls).stream()
                .filter(provider -> provider.getService().equals(serviceName))
                .collect(Collectors.toList());
    }

    public List<Consumer> findAllConsumer() {
        return instanceRegistryCache.getSubscribedCache().values().stream()
                .flatMap(m -> m.values().stream())
                .flatMap(Collection::stream)
                .map(m -> new Pair<>(m.toFullString(), m))
                .map(SyncUtils::url2Consumer)
                .collect(Collectors.toList());
    }

    public List<Consumer> findConsumerByService(String serviceName) {
        return instanceRegistryCache.getSubscribedCache().values().stream().filter(m -> m.containsKey(serviceName))
                .flatMap(m -> m.get(serviceName).stream())
                .map(m -> new Pair<>(m.toFullString(), m))
                .map(SyncUtils::url2Consumer)
                .collect(Collectors.toList());
    }

    public List<Provider> findByAddress(String providerAddress) {
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null) {
            return Lists.newArrayList();
        }
        List<InstanceAddressURL> providerUrls = Lists.newArrayList();
        appInterfaceMap.values().forEach(serviceUrlMap ->
                serviceUrlMap.forEach((service, urls) -> {
                    if (CollectionUtils.isNotEmpty(urls)) {
                        urls.forEach(url -> {
                            if ((url.getInstance().getHost().equals(providerAddress))) {
                                providerUrls.add(url);
                            }
                        });
                    }
                }));
        return urlsToProviderList(providerUrls);
    }

    public List<Provider> findByApplication(String application) {
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null || appInterfaceMap.get(application) == null) {
            return Lists.newArrayList();
        }
        List<InstanceAddressURL> providerUrls = Lists.newArrayList();
        appInterfaceMap.get(application).forEach((service, urls) -> providerUrls.addAll(urls));
        return urlsToProviderList(providerUrls);
    }

    public String findVersionInApplication(String application) {
        ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appInterfaceMap = instanceRegistryCache.get(Constants.PROVIDERS_CATEGORY);
        if (appInterfaceMap == null || appInterfaceMap.get(application) == null) {
            return null;
        }
        Map<String, List<InstanceAddressURL>> urlsMap = appInterfaceMap.get(application);
        for (Map.Entry<String, List<InstanceAddressURL>> entry : urlsMap.entrySet()) {
            List<InstanceAddressURL> urls = entry.getValue();
            if (CollectionUtils.isNotEmpty(urls)) {
                return urls.get(0).getParameter(Constants.SPECIFICATION_VERSION_KEY, "3.0.0");
            }
        }
        return null;
    }

    private List<Provider> urlsToProviderList(List<InstanceAddressURL> urls) {
        List<Provider> providers = Lists.newArrayList();
        urls.stream().distinct().forEach(url -> {
            ServiceInstance instance = url.getInstance();
            MetadataInfo metadataInfo = url.getMetadataInfo();

            metadataInfo.getServices().forEach((serviceKey, serviceInfo) -> {
                // build consumer url

                ServiceConfigURL consumerUrl = new URLBuilder()
                        .setProtocol(serviceInfo.getProtocol())
                        .setPath(serviceInfo.getPath())
                        .addParameter("group", serviceInfo.getGroup())
                        .addParameter("version", serviceInfo.getVersion())
                        .build();
                RpcContext.getServiceContext().setConsumerUrl(consumerUrl);
                Provider p = new Provider();
                String service = serviceInfo.getServiceKey();
                p.setService(service);
                p.setAddress(url.getAddress());
                p.setApplication(instance.getServiceName());
                p.setUrl(url.toFullString());
                p.setDynamic(url.getParameter("dynamic", true));
                p.setEnabled(url.getParameter(Constants.ENABLED_KEY, true));
                p.setSerialization(url.getParameter(org.apache.dubbo.remoting.Constants.SERIALIZATION_KEY, "hessian2"));
                p.setTimeout(url.getParameter(CommonConstants.TIMEOUT_KEY, CommonConstants.DEFAULT_TIMEOUT));
                p.setWeight(url.getParameter(Constants.WEIGHT_KEY, Constants.DEFAULT_WEIGHT));
                p.setUsername(url.getParameter("owner"));
                p.setRegistrySource(RegistrySource.INSTANCE);
                providers.add(p);

                RpcContext.getServiceContext().setConsumerUrl(null);
            });
        });
        return providers;
    }
}
