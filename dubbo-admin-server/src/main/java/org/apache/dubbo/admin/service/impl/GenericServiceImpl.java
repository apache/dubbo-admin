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
import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.rpc.service.GenericService;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;

@Component
public class GenericServiceImpl {
    private ApplicationConfig applicationConfig;
    private final Registry registry;

    public GenericServiceImpl(Registry registry) {
        this.registry = registry;
    }

    @PostConstruct
    public void init() {
        RegistryConfig registryConfig = buildRegistryConfig(registry);

        applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
        applicationConfig.setRegistry(registryConfig);
    }

    private RegistryConfig buildRegistryConfig(Registry registry) {
        URL fromUrl = registry.getUrl();

        RegistryConfig config = new RegistryConfig();
        config.setGroup(fromUrl.getParameter("group"));

        URL address = URL.valueOf(fromUrl.getProtocol() + "://" + fromUrl.getAddress());
        if (fromUrl.hasParameter(Constants.NAMESPACE_KEY)) {
            address = address.addParameter(Constants.NAMESPACE_KEY, fromUrl.getParameter(Constants.NAMESPACE_KEY));
        }

        config.setAddress(address.toString());
        return config;
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {

        ReferenceConfig<GenericService> reference = new ReferenceConfig<>();
        String group = Tool.getGroup(service);
        String version = Tool.getVersion(service);
        String intf = Tool.getInterface(service);
        reference.setGeneric(true);
        reference.setApplication(applicationConfig);
        reference.setInterface(intf);
        reference.setVersion(version);
        reference.setGroup(group);
        //Keep it consistent with the ConfigManager cache
        reference.setSticky(false);
        try {
            removeGenericSymbol(parameterTypes);
            GenericService genericService = reference.get();
            return genericService.$invoke(method, parameterTypes, params);
        } finally {
            reference.destroy();
        }
    }

    /**
     * remove generic from parameterTypes
     *
     * @param parameterTypes
     */
    private void removeGenericSymbol(String[] parameterTypes) {
        for (int i = 0; i < parameterTypes.length; i++) {
            int index = parameterTypes[i].indexOf("<");
            if (index > -1) {
                parameterTypes[i] = parameterTypes[i].substring(0, index);
            }
        }
    }
}
