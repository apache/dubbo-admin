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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.registry.mapping.ServiceMapping;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.CollectionUtils;
import org.apache.dubbo.common.utils.ConcurrentHashSet;
import org.apache.dubbo.metadata.MappingChangedEvent;
import org.apache.dubbo.metadata.MappingListener;
import org.apache.dubbo.remoting.zookeeper.ZookeeperClient;
import org.apache.dubbo.remoting.zookeeper.ZookeeperTransporter;
import org.apache.dubbo.rpc.model.ApplicationModel;

import java.util.List;
import java.util.Set;

import static org.apache.dubbo.metadata.ServiceNameMapping.getAppNames;


public class ZookeeperServiceMapping implements ServiceMapping {

    private ZookeeperClient zkClient;

    private final static String MAPPING_PATH = Constants.PATH_SEPARATOR + Constants.DEFAULT_ROOT + Constants.PATH_SEPARATOR + Constants.DEFAULT_MAPPING_GROUP;

    private final Set<MappingListener> listeners = new ConcurrentHashSet<>();

    private final Set<String> anyServices = new ConcurrentHashSet<>();

    @Override
    public void init(URL url) {
        ZookeeperTransporter zookeeperTransporter = ZookeeperTransporter.getExtension(ApplicationModel.defaultModel());
        zkClient = zookeeperTransporter.connect(url);
        listenerAll();
    }

    @Override
    public void listenerAll() {
        zkClient.create(MAPPING_PATH, false);
        List<String> services = zkClient.addChildListener(MAPPING_PATH, (path, currentChildList) -> {
            for (String child : currentChildList) {
                if (anyServices.add(child)) {
                    notifyMappingChangedEvent(child);
                }
            }
        });
        if (CollectionUtils.isNotEmpty(services)) {
            for (String service : services) {
                if (anyServices.add(service)) {
                    notifyMappingChangedEvent(service);
                }
            }
        }
    }

    private void notifyMappingChangedEvent(String service) {
        if (service.equals(Constants.CONFIGURATORS_CATEGORY) || service.equals(Constants.CONSUMERS_CATEGORY)
                || service.equals(Constants.PROVIDERS_CATEGORY) || service.equals(Constants.ROUTERS_CATEGORY)) {
            return;
        }
        String servicePath = MAPPING_PATH + Constants.PATH_SEPARATOR + service;
        String content = zkClient.getContent(servicePath);
        if (content != null) {
            Set<String> apps = getAppNames(content);
            MappingChangedEvent event = new MappingChangedEvent(service, apps);
            for (MappingListener listener : listeners) {
                listener.onEvent(event);
            }
        }
    }

    @Override
    public void addMappingListener(MappingListener listener) {
        listeners.add(listener);
    }

}
