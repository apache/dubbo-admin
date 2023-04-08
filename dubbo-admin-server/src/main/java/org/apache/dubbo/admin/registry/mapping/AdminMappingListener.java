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

package org.apache.dubbo.admin.registry.mapping;

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.service.impl.InstanceRegistryCache;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.CollectionUtils;
import org.apache.dubbo.common.utils.NetUtils;
import org.apache.dubbo.metadata.MappingChangedEvent;
import org.apache.dubbo.metadata.MappingListener;
import org.apache.dubbo.registry.client.InstanceAddressURL;
import org.apache.dubbo.registry.client.ServiceDiscovery;
import org.apache.dubbo.registry.client.ServiceInstance;
import org.apache.dubbo.registry.client.event.ServiceInstancesChangedEvent;
import org.apache.dubbo.registry.client.event.listener.ServiceInstancesChangedListener;

import com.google.common.collect.Sets;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.stream.Collectors;

public class AdminMappingListener implements MappingListener {

    private static final URL CONSUMER_URL = new URL(Constants.ADMIN_PROTOCOL, NetUtils.getLocalHost(), 0, "",
            Constants.INTERFACE_KEY, Constants.ANY_VALUE,
            Constants.GROUP_KEY, Constants.ANY_VALUE,
            Constants.VERSION_KEY, Constants.ANY_VALUE,
            Constants.CLASSIFIER_KEY, Constants.ANY_VALUE,
            Constants.CATEGORY_KEY, Constants.PROVIDERS_CATEGORY + ","
            + Constants.CONSUMERS_CATEGORY + ","
            + Constants.ROUTERS_CATEGORY + ","
            + Constants.CONFIGURATORS_CATEGORY,
            Constants.ENABLED_KEY, Constants.ANY_VALUE,
            Constants.CHECK_KEY, String.valueOf(false));

    /* app - listener */
    private final Map<String, ServiceInstancesChangedListener> serviceListeners = new ConcurrentHashMap<>();

    private final ServiceDiscovery serviceDiscovery;

    private final InstanceRegistryCache instanceRegistryCache;

    public AdminMappingListener(ServiceDiscovery serviceDiscovery, InstanceRegistryCache instanceRegistryCache) {
        this.serviceDiscovery = serviceDiscovery;
        this.instanceRegistryCache = instanceRegistryCache;
    }

    @Override
    public void onEvent(MappingChangedEvent event) {
        Set<String> apps = event.getApps();
        if (CollectionUtils.isEmpty(apps)) {
            return;
        }
        for (String serviceName : apps) {
            ServiceInstancesChangedListener serviceInstancesChangedListener = serviceListeners.get(serviceName);
            if (serviceInstancesChangedListener == null) {
                synchronized (this) {
                    serviceInstancesChangedListener = serviceListeners.get(serviceName);
                    if (serviceInstancesChangedListener == null) {
                        AddressChangeListener addressChangeListener = new DefaultAddressChangeListener(serviceName, instanceRegistryCache);
                        serviceInstancesChangedListener = new AdminServiceInstancesChangedListener(Sets.newHashSet(serviceName), serviceDiscovery, addressChangeListener);
                        serviceInstancesChangedListener.setUrl(CONSUMER_URL);
                        List<ServiceInstance> serviceInstances = serviceDiscovery.getInstances(serviceName);
                        if (CollectionUtils.isNotEmpty(serviceInstances)) {
                            serviceInstancesChangedListener.onEvent(new ServiceInstancesChangedEvent(serviceName, serviceInstances));
                        }
                        serviceListeners.put(serviceName, serviceInstancesChangedListener);
                        serviceInstancesChangedListener.setUrl(CONSUMER_URL);
                        serviceDiscovery.addServiceInstancesChangedListener(serviceInstancesChangedListener);
                    }
                }
            }
        }
    }

    private static class DefaultAddressChangeListener implements AddressChangeListener {

        private String serviceName;

        private InstanceRegistryCache instanceRegistryCache;

        public DefaultAddressChangeListener(String serviceName, InstanceRegistryCache instanceRegistryCache) {
            this.serviceName = serviceName;
            this.instanceRegistryCache = instanceRegistryCache;
        }

        @Override
        public void notifyAddressChanged(String protocolServiceKey, List<URL> urls) {
            String serviceKey = removeProtocol(protocolServiceKey);
            ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> appServiceMap = instanceRegistryCache.computeIfAbsent(Constants.PROVIDERS_CATEGORY, key -> new ConcurrentHashMap<>());
            Map<String, List<InstanceAddressURL>> serviceMap = appServiceMap.computeIfAbsent(serviceName, key -> new ConcurrentHashMap<>());
            if (CollectionUtils.isEmpty(urls)) {
                serviceMap.remove(serviceKey);
            } else {
                List<InstanceAddressURL> instanceAddressUrls = urls.stream().map(url -> (InstanceAddressURL) url).collect(Collectors.toList());
                serviceMap.put(serviceKey, instanceAddressUrls);
            }
            instanceRegistryCache.refreshConsumer(false);
        }

        private String removeProtocol(String protocolServiceKey) {
            int index = protocolServiceKey.lastIndexOf(":");
            if (index == -1) {
                return protocolServiceKey;
            }
            return protocolServiceKey.substring(0, index);
        }
    }

    @Override
    public void stop() {
        // ignore
    }

}
