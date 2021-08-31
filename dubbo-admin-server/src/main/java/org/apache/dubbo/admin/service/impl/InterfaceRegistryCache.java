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
import org.apache.dubbo.common.URL;

import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

/**
 * interface registry url cache
 * key --> category,value --> ConcurrentMap<serviceKey, Map<hash, URL>>
 */
@Component
public class InterfaceRegistryCache implements RegistryCache<String, ConcurrentMap<String, Map<String, URL>>> {

    private final ConcurrentMap<String, ConcurrentMap<String, Map<String, URL>>> registryCache = new ConcurrentHashMap<>();

    @Override
    public void put(String key, ConcurrentMap<String, Map<String, URL>> value) {
        registryCache.put(key, value);
    }

    @Override
    public ConcurrentMap<String, Map<String, URL>> get(String key) {
        return registryCache.get(key);
    }

    public ConcurrentMap<String, ConcurrentMap<String, Map<String, URL>>> getRegistryCache() {
        return registryCache;
    }
}
