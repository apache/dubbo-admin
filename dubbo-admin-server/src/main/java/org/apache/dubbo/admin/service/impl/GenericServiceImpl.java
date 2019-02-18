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
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.config.context.ConfigManager;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.rpc.service.GenericService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class GenericServiceImpl {

    private static final Map<String, ReferenceConfig<GenericService>> referenceConfigMap = new ConcurrentHashMap<>();

    @Autowired
    private Registry registry;

    private static volatile ApplicationConfig applicationConfig;

    public void initApplicationConfig() {
        if (applicationConfig == null) {
            synchronized (GenericServiceImpl.class){
                if (applicationConfig == null) {
                    RegistryConfig registryConfig = new RegistryConfig();
                    registryConfig.setAddress(registry.getUrl().getProtocol() + "://" + registry.getUrl().getAddress());
                    registryConfig.setGroup(registry.getUrl().getParameter(org.apache.dubbo.common.Constants.GROUP_KEY, Constants.DEFAULT_ROOT));
                    applicationConfig = new ApplicationConfig();
                    applicationConfig.setName("dubbo-admin");
                    applicationConfig.setRegistry(registryConfig);
                }
            }
        }
    }

    public ReferenceConfig<GenericService> getReferenceConfig(String service) {
        ReferenceConfig<GenericService> reference = referenceConfigMap.get(service);
        if (reference == null) {
            referenceConfigMap.computeIfAbsent(service, key -> {
                ReferenceConfig<GenericService> genericServiceReferenceConfig = new ReferenceConfig<>();
                genericServiceReferenceConfig.setGeneric(true);
                if (ConfigManager.getInstance().getApplication().isPresent()) {
                    genericServiceReferenceConfig.setApplication(ConfigManager.getInstance().getApplication().get());
                } else {
                    initApplicationConfig();
                    genericServiceReferenceConfig.setApplication(applicationConfig);
                }
                int i;
                if ((i = key.indexOf(Constants.PATH_SEPARATOR)) > -1) {
                    genericServiceReferenceConfig.setGroup(key.substring(0, i));
                    key = key.substring(i + 1);
                }
                if ((i = key.indexOf(':')) > -1) {
                    genericServiceReferenceConfig.setVersion(key.substring(i + 1));
                    key = key.substring(0, i);
                }
                genericServiceReferenceConfig.setInterface(key);
                return genericServiceReferenceConfig;
            });
        }
        return referenceConfigMap.get(service);
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {
        ReferenceConfig<GenericService> reference = getReferenceConfig(service);
        GenericService genericService = reference.get();
        return genericService.$invoke(method, parameterTypes, params);
    }
}
