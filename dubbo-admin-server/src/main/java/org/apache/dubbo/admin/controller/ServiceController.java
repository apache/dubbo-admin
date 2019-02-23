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

package org.apache.dubbo.admin.controller;

import com.google.gson.Gson;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.ServiceDTO;
import org.apache.dubbo.admin.model.dto.ServiceDetailDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Map;
import java.util.Set;

@RestController
@RequestMapping("/api/{env}")
public class ServiceController {

    private final ProviderService providerService;
    private final ConsumerService consumerService;
    private final Gson gson;

    @Autowired
    public ServiceController(ProviderService providerService, ConsumerService consumerService) {
        this.providerService = providerService;
        this.consumerService = consumerService;
        this.gson = new Gson();
    }

    @RequestMapping( value = "/service", method = RequestMethod.GET)
    public Set<ServiceDTO> searchService(@RequestParam String pattern,
                                         @RequestParam String filter,@PathVariable String env) {
        return providerService.getServiceDTOS(pattern, filter, env);
    }

    @RequestMapping(value = "/service/{service}", method = RequestMethod.GET)
    public ServiceDetailDTO serviceDetail(@PathVariable String service, @PathVariable String env) {
        service = service.replace(Constants.ANY_VALUE, Constants.PATH_SEPARATOR);
        String group = null;
        String version = null;
        String interfaze = service;
        int i = interfaze.indexOf("/");
        if (i >= 0) {
            group = interfaze.substring(0, i);
            interfaze = interfaze.substring(i + 1);
        }
        i = interfaze.lastIndexOf(":");
        if (i >= 0) {
            version = interfaze.substring(i + 1);
            interfaze = interfaze.substring(0, i);
        }
        List<Provider> providers = providerService.findByService(service);

        List<Consumer> consumers = consumerService.findByService(service);

        String application = null;
        if (providers != null && providers.size() > 0) {
            application = providers.get(0).getApplication();
        }
        MetadataIdentifier identifier = new MetadataIdentifier(interfaze, version, group, Constants.PROVIDER_SIDE, application);
        String metadata = providerService.getProviderMetaData(identifier);
        ServiceDetailDTO serviceDetailDTO = new ServiceDetailDTO();
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        if (metadata != null) {
            FullServiceDefinition serviceDefinition = gson.fromJson(metadata, FullServiceDefinition.class);
            serviceDetailDTO.setMetadata(serviceDefinition);
        }
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        serviceDetailDTO.setService(service);
        serviceDetailDTO.setApplication(application);
        return serviceDetailDTO;
    }

    @RequestMapping(value = "/services", method = RequestMethod.GET)
    public Set<String> allServices(@PathVariable String env) {
        return providerService.findServices();
    }

    @RequestMapping(value = "/applications", method = RequestMethod.GET)
    public Set<String> allApplications(@PathVariable String env) {
        return providerService.findApplications();
    }
}
