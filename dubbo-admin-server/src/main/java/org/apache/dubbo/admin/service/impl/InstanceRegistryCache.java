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
import org.apache.dubbo.admin.service.RegistryCache;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.NamedThreadFactory;
import org.apache.dubbo.metadata.MetadataService;
import org.apache.dubbo.registry.client.InstanceAddressURL;
import org.apache.dubbo.registry.client.metadata.MetadataUtils;
import org.apache.dubbo.rpc.service.Destroyable;

import org.springframework.stereotype.Component;

import java.util.Collection;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Function;
import java.util.stream.Collectors;

/**
 * instance registry url {@link InstanceAddressURL} cache
 * key --> category,value --> ConcurrentMap<appName, Map<serviceKey, List<InstanceAddressURL>>>
 */
@Component
public class InstanceRegistryCache implements RegistryCache<String, ConcurrentMap<String, Map<String, List<InstanceAddressURL>>>> {

    private final ConcurrentMap<String, ConcurrentMap<String, Map<String, List<InstanceAddressURL>>>> registryCache = new ConcurrentHashMap<>();

    private final Map<String, Map<String, List<URL>>> subscribedCache = new ConcurrentHashMap<>();

    private final AtomicBoolean startRefresh = new AtomicBoolean(false);

    @Override
    public void put(String key, ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> value) {
        registryCache.put(key, value);
    }

    @Override
    public ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> get(String key) {
        return registryCache.get(key);
    }

    @Override
    public ConcurrentMap<String, Map<String, List<InstanceAddressURL>>> computeIfAbsent(String key,
                                                                                        Function<? super String, ? extends ConcurrentMap<String, Map<String, List<InstanceAddressURL>>>> mappingFunction) {
        return registryCache.computeIfAbsent(key, mappingFunction);
    }

    public Map<String, Map<String, List<URL>>> getSubscribedCache() {
        return subscribedCache;
    }

    public synchronized void refreshConsumer(boolean refreshAll) {
        if (startRefresh.compareAndSet(false, true)) {
            ScheduledExecutorService executorService = new ScheduledThreadPoolExecutor(1, new NamedThreadFactory("Consumer-Refresh"));
            executorService.scheduleAtFixedRate(() -> refreshConsumer(true), 60, 60, TimeUnit.MINUTES);
        }

        Map<String, Map<String, List<URL>>> origin;

        if (refreshAll) {
            origin = new ConcurrentHashMap<>();
        } else {
            origin = subscribedCache;
        }

        Map<String, List<InstanceAddressURL>> providers = get(Constants.PROVIDERS_CATEGORY).values().stream()
                .flatMap((e) -> e.values().stream())
                .flatMap(Collection::stream)
                .collect(Collectors.groupingBy(InstanceAddressURL::getAddress));

        // remove cached
        origin.keySet().forEach(providers::remove);

        for (List<InstanceAddressURL> instanceAddressURLs : providers.values()) {
            MetadataService metadataService = MetadataUtils.referProxy(instanceAddressURLs.get(0).getInstance());
            try {
                Set<String> subscribedURLs = metadataService.getSubscribedURLs();

                Map<String, List<URL>> subscribed = subscribedURLs.stream().map(URL::valueOf).collect(Collectors.groupingBy(URL::getServiceKey));
                origin.put(instanceAddressURLs.get(0).getAddress(), subscribed);
            } catch (Throwable ignored) {

            }
            ((Destroyable) metadataService).$destroy();
        }
    }
}
