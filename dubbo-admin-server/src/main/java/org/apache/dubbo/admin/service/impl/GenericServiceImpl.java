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

import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.config.RegistryConfig;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.rpc.service.GenericService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;

@Component
public class GenericServiceImpl {

    private ReferenceConfig<GenericService> reference;

    @Autowired
    private Registry registry;

    @PostConstruct
    public void init() {
        reference = new ReferenceConfig<>();
        reference.setGeneric(true);

        RegistryConfig registryConfig = new RegistryConfig();
        registryConfig.setAddress(registry.getUrl().getProtocol() + "://" + registry.getUrl().getAddress());

        ApplicationConfig applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
        applicationConfig.setRegistry(registryConfig);

        reference.setApplication(applicationConfig);
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {

        reference.setInterface(service);
        GenericService genericService = reference.get();
        return genericService.$invoke(method, parameterTypes, params);
    }

    public static void main(String[] args) {
        GenericServiceImpl genericService = new GenericServiceImpl();
        genericService.init();
        genericService.invoke("org.apache.dubbo.demo.api.DemoService", "sayHello", new String[]{"java.lang.String"}, new Object[]{"hello"});
    }
}
