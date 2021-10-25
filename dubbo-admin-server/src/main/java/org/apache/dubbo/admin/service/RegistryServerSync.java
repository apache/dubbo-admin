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
package org.apache.dubbo.admin.service;

import org.apache.dubbo.admin.common.util.CoderUtil;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.admin.service.impl.InterfaceRegistryCache;
import org.apache.dubbo.common.BaseServiceMetadata;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.common.utils.NetUtils;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.registry.NotifyListener;
import org.apache.dubbo.registry.Registry;

import org.springframework.beans.factory.DisposableBean;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;

@Component
public class RegistryServerSync implements DisposableBean, NotifyListener {

    private static final Logger logger = LoggerFactory.getLogger(RegistryServerSync.class);

    private static final URL SUBSCRIBE = new URL(Constants.ADMIN_PROTOCOL, NetUtils.getLocalHost(), 0, "",
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

    /**
     * Make sure ID never changed when the same url notified many times
     */
    private final ConcurrentHashMap<String, String> URL_IDS_MAPPER = new ConcurrentHashMap<>();

    /**
     * ConcurrentMap<category, ConcurrentMap<servicename, Map<MD5, URL>>>
     * registryCache
     */
    @Autowired
    private Registry registry;

    @Autowired
    private InterfaceRegistryCache interfaceRegistryCache;

    @EventListener(classes = ApplicationReadyEvent.class)
    public void startSubscribe() {
        logger.info("Init Dubbo Admin Sync Cache...");
        registry.subscribe(SUBSCRIBE, this);
    }

    @Override
    public void destroy() throws Exception {
        registry.unsubscribe(SUBSCRIBE, this);
    }

    // Notification of any service with any type (override、subscribe、route、provider) is full.
    @Override
    public void notify(List<URL> urls) {
        if (urls == null || urls.isEmpty()) {
            return;
        }
        // Map<category, Map<servicename, Map<Long, URL>>>
        final Map<String, Map<String, Map<String, URL>>> categories = new HashMap<>();
        String interfaceName = null;
        for (URL url : urls) {
            String category = url.getUrlParam().getParameter(Constants.CATEGORY_KEY);
            if (category == null) {
                category = Constants.PROVIDERS_CATEGORY;
            }
            // NOTE: group and version in empty protocol is *
            if (Constants.EMPTY_PROTOCOL.equalsIgnoreCase(url.getProtocol())) {
                ConcurrentMap<String, Map<String, URL>> services = interfaceRegistryCache.get(category);
                if (services != null) {
                    String group = url.getUrlParam().getParameter(Constants.GROUP_KEY);
                    String version = url.getUrlParam().getParameter(Constants.VERSION_KEY);
                    // NOTE: group and version in empty protocol is *
                    if (!Constants.ANY_VALUE.equals(group) && !Constants.ANY_VALUE.equals(version)) {
                        services.remove(getServiceInterface(url));
                    } else {
                        for (Map.Entry<String, Map<String, URL>> serviceEntry : services.entrySet()) {
                            String service = serviceEntry.getKey();
                            if (Tool.getInterface(service).equals(getServiceInterface(url))
                                    && (Constants.ANY_VALUE.equals(group) || StringUtils.isEquals(group, Tool.getGroup(service)))
                                    && (Constants.ANY_VALUE.equals(version) || StringUtils.isEquals(version, Tool.getVersion(service)))) {
                                services.remove(service);
                            }
                        }
                    }
                }
            } else {
                if (StringUtils.isEmpty(interfaceName)) {
                    interfaceName = getServiceInterface(url);
                }
                Map<String, Map<String, URL>> services = categories.get(category);
                if (services == null) {
                    services = new HashMap<>();
                    categories.put(category, services);
                }
                String group = url.getUrlParam().getParameter(Constants.GROUP_KEY);
                String version = url.getUrlParam().getParameter(Constants.VERSION_KEY);
                String service = BaseServiceMetadata.buildServiceKey(getServiceInterface(url), group, version);
                Map<String, URL> ids = services.get(service);
                if (ids == null) {
                    ids = new HashMap<>();
                    services.put(service, ids);
                }

                // Make sure we use the same ID for the same URL
                if (URL_IDS_MAPPER.containsKey(url.toFullString())) {
                    ids.put(URL_IDS_MAPPER.get(url.toFullString()), url);
                } else {
                    String md5 = CoderUtil.MD5_16bit(url.toFullString());
                    ids.put(md5, url);
                    URL_IDS_MAPPER.putIfAbsent(url.toFullString(), md5);
                }
            }
        }
        if (categories.size() == 0) {
            return;
        }
        for (Map.Entry<String, Map<String, Map<String, URL>>> categoryEntry : categories.entrySet()) {
            String category = categoryEntry.getKey();
            ConcurrentMap<String, Map<String, URL>> services = interfaceRegistryCache.get(category);
            if (services == null) {
                services = new ConcurrentHashMap<>();
                interfaceRegistryCache.put(category, services);
            } else {// Fix map can not be cleared when service is unregistered: when a unique “group/service:version” service is unregistered, but we still have the same services with different version or group, so empty protocols can not be invoked.
                Set<String> keys = new HashSet<>(services.keySet());
                for (String key : keys) {
                    if (Tool.getInterface(key).equals(interfaceName) && !categoryEntry.getValue().containsKey(key)) {
                        services.remove(key);
                    }
                }
            }
            services.putAll(categoryEntry.getValue());
        }
    }

    private String getServiceInterface(URL url) {
        String serviceInterface = url.getServiceInterface();
        if (StringUtils.isBlank(serviceInterface) || Constants.ANY_VALUE.equals(serviceInterface)) {
            serviceInterface = url.getPath();
        }
        return serviceInterface;
    }

}

