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

import javax.annotation.PostConstruct;

@Component
public class GenericServiceImpl {
    private ApplicationConfig applicationConfig;
    @Autowired
    private Registry registry;

    @PostConstruct
    public void init() {
        RegistryConfig registryConfig = new RegistryConfig();
        registryConfig.setAddress(registry.getUrl().getProtocol() + "://" + registry.getUrl().getAddress());
        registryConfig.setGroup(registry.getUrl().getParameter(org.apache.dubbo.common.Constants.GROUP_KEY, Constants.DEFAULT_ROOT));
        applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
        applicationConfig.setRegistry(registryConfig);

    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {
        ReferenceConfig<GenericService> reference = new ReferenceConfig<>();
        reference.setGeneric(true);
        if (ConfigManager.getInstance().getApplication().isPresent()) {
            reference.setApplication(ConfigManager.getInstance().getApplication().get());
        } else {
            reference.setApplication(applicationConfig);
        }
        int i;
        if ((i = service.indexOf(Constants.PATH_SEPARATOR)) > -1) {
            reference.setGroup(service.substring(0, i));
            service = service.substring(i + 1);
        }
        if ((i = service.indexOf(':')) > -1) {
            reference.setVersion(service.substring(i + 1));
            service = service.substring(0, i);
        }
        reference.setInterface(service);
        GenericService genericService = reference.get();
        return genericService.$invoke(method, parameterTypes, params);
    }
}