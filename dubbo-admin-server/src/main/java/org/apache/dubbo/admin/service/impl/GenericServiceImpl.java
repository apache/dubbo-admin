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

import java.util.List;

import javax.annotation.PostConstruct;

import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.rpc.service.GenericService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

@Component
public class GenericServiceImpl {
    private ProviderService providerService;
    private ApplicationConfig applicationConfig;
    
    @Autowired
    public GenericServiceImpl(ProviderService providerService) {
        this.providerService = providerService;
    }

    @PostConstruct
    public void init() {
        applicationConfig = new ApplicationConfig();
        applicationConfig.setName("dubbo-admin");
    }

    public Object invoke(String service, String method, String[] parameterTypes, Object[] params) {
    	List<Provider> providers = providerService.findByService(service);

        ReferenceConfig<GenericService> reference = new ReferenceConfig<>();
        String group = Tool.getGroup(service);
        String version = Tool.getVersion(service);
        String interfaze = Tool.getInterface(service);
        reference.setGeneric(true);
        reference.setApplication(applicationConfig);
        reference.setInterface(interfaze);
        reference.setVersion(version);
        reference.setGroup(group);
        reference.setUrl(providers.get(0).getUrl()+"?generic=true&timeout=3000");
        GenericService genericService = reference.get();

        return genericService.$invoke(method, parameterTypes, params);
    }
}
