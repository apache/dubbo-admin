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

import org.apache.dubbo.admin.service.RegistryCache;
import org.apache.dubbo.registry.client.InstanceAddressURL;

import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.function.Function;

/**
 * instance registry url {@link InstanceAddressURL} cache
 * key --> category,value --> ConcurrentMap<appName, Map<serviceKey, List<InstanceAddressURL>>>
 */
@Component
public class InstanceRegistryCache implements RegistryCache<String, ConcurrentMap<String, Map<String, List<InstanceAddressURL>>>> {

    private final ConcurrentMap<String, ConcurrentMap<String, Map<String, List<InstanceAddressURL>>>> registryCache = new ConcurrentHashMap<>();

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
}
