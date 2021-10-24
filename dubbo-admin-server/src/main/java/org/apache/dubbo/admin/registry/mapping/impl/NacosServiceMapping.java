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

package org.apache.dubbo.admin.registry.mapping.impl;

import org.apache.dubbo.admin.registry.mapping.ServiceMapping;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.common.utils.ConcurrentHashSet;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.metadata.MappingChangedEvent;
import org.apache.dubbo.metadata.MappingListener;
import org.apache.dubbo.registry.nacos.NacosNamingServiceWrapper;
import org.apache.dubbo.registry.nacos.util.NacosNamingServiceUtils;

import com.alibaba.nacos.api.common.Constants;
import com.alibaba.nacos.api.exception.NacosException;
import com.alibaba.nacos.api.naming.pojo.ListView;
import com.google.common.collect.Sets;

import java.util.Arrays;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

import static com.alibaba.nacos.api.PropertyKeyConst.NAMING_LOAD_CACHE_AT_START;
import static org.apache.dubbo.common.constants.RegistryConstants.CONFIGURATORS_CATEGORY;
import static org.apache.dubbo.common.constants.RegistryConstants.CONSUMERS_CATEGORY;
import static org.apache.dubbo.common.constants.RegistryConstants.PROVIDERS_CATEGORY;
import static org.apache.dubbo.common.constants.RegistryConstants.ROUTERS_CATEGORY;

/**
 * Nacos not support batch listen config feature. Therefore, regularly query the service list instead of notification
 */
public class NacosServiceMapping implements ServiceMapping {

    /**
     * All 2.x supported categories
     */
    private static final List<String> ALL_SUPPORTED_CATEGORIES = Arrays.asList(
            PROVIDERS_CATEGORY,
            CONSUMERS_CATEGORY,
            ROUTERS_CATEGORY,
            CONFIGURATORS_CATEGORY
    );

    /**
     * The separator for service name
     * Change a constant to be configurable, it's designed for Windows file name that is compatible with old
     * Nacos binary release(< 0.6.1)
     */
    private static final String SERVICE_NAME_SEPARATOR = System.getProperty("nacos.service.name.separator", ":");

    private static final long LOOKUP_INTERVAL = Long.getLong("nacos.service.names.lookup.interval", 30);

    private ScheduledExecutorService scheduledExecutorService;

    private final Set<MappingListener> listeners = new ConcurrentHashSet<>();

    private static final int PAGINATION_SIZE = 100;

    private NacosNamingServiceWrapper namingService;

    private Set<String> anyServices = new HashSet<>();

    private static final Logger LOGGER = LoggerFactory.getLogger(NacosServiceMapping.class);

    @Override
    public void init(URL url) {
        url.addParameter(NAMING_LOAD_CACHE_AT_START, "false");
        namingService = NacosNamingServiceUtils.createNamingService(url);
        scheduledExecutorService = Executors.newSingleThreadScheduledExecutor();
        listenerAll();
    }

    @Override
    public void listenerAll() {

        try {
            anyServices = getAllServiceNames().stream().filter(this::filterApplication).collect(Collectors.toSet());
        } catch (Exception e) {
            LOGGER.error("Get nacos all services fail ", e);
        }
        for (String service : anyServices) {
            notifyMappingChangedEvent(service);
        }
        scheduledExecutorService.scheduleAtFixedRate(() -> {
            try {
                Set<String> serviceNames = getAllServiceNames();
                for (String serviceName : serviceNames) {
                    if (filterApplication(serviceName) && anyServices.add(serviceName)) {
                        notifyMappingChangedEvent(serviceName);
                    }
                }
            } catch (Exception e) {
                LOGGER.error("Get nacos all services fail ", e);
            }

        }, LOOKUP_INTERVAL, LOOKUP_INTERVAL, TimeUnit.SECONDS);
    }

    private Set<String> getAllServiceNames() throws NacosException {

        Set<String> serviceNames = new HashSet<>();
        int pageIndex = 1;
        ListView<String> listView = namingService.getServicesOfServer(pageIndex, PAGINATION_SIZE,
                Constants.DEFAULT_GROUP);
        // First page data
        List<String> firstPageData = listView.getData();
        // Append first page into list
        serviceNames.addAll(firstPageData);
        // the total count
        int count = listView.getCount();
        // the number of pages
        int pageNumbers = count / PAGINATION_SIZE;
        int remainder = count % PAGINATION_SIZE;
        // remain
        if (remainder > 0) {
            pageNumbers += 1;
        }
        // If more than 1 page
        while (pageIndex < pageNumbers) {
            listView = namingService.getServicesOfServer(++pageIndex, PAGINATION_SIZE, Constants.DEFAULT_GROUP);
            serviceNames.addAll(listView.getData());
        }

        return serviceNames;
    }

    private boolean filterApplication(String serviceName) {
        if (StringUtils.isBlank(serviceName)) {
            return false;
        }
        for (String category : ALL_SUPPORTED_CATEGORIES) {
            String prefix = category + SERVICE_NAME_SEPARATOR;
            if (serviceName.startsWith(prefix)) {
                return false;
            }
        }
        return true;
    }

    private void notifyMappingChangedEvent(String service) {
        MappingChangedEvent event = new MappingChangedEvent(null, Sets.newHashSet(service));
        for (MappingListener listener : listeners) {
            listener.onEvent(event);
        }
    }


    @Override
    public void addMappingListener(MappingListener listener) {
        listeners.add(listener);
    }

}
