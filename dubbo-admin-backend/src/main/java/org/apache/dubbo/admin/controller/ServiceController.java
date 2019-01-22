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
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.ServiceDTO;
import org.apache.dubbo.admin.model.dto.ServiceDetailDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.*;

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
        List<Provider> providers = providerService.findByService(service);

        List<Consumer> consumers = consumerService.findByService(service);

        Map<String, String> info = ConvertUtil.serviceName2Map(service);
        String application = null;
        if (providers != null && providers.size() > 0) {
            application = providers.get(0).getApplication();
        }
        MetadataIdentifier identifier = new MetadataIdentifier(info.get(Constants.INTERFACE_KEY),
                                                                      info.get(Constants.VERSION_KEY),
                                                                      info.get(Constants.GROUP_KEY), Constants.PROVIDER_SIDE, application);
        String metadata = providerService.getProviderMetaData(identifier);
        ServiceDetailDTO serviceDetailDTO = new ServiceDetailDTO();
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        if (metadata != null) {
            FullServiceDefinition serviceDefinition = gson.fromJson(metadata, FullServiceDefinition.class);
            serviceDetailDTO.setMetadata(serviceDefinition);
        }
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
