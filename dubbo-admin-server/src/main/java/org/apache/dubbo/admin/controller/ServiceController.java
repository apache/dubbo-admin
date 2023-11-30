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

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.common.exception.VersionValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.Tool;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.ServiceDTO;
import org.apache.dubbo.admin.model.dto.ServiceDetailDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;

import com.google.gson.Gson;
import com.google.gson.JsonParseException;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.report.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageImpl;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;

@Authority(needLogin = true)
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

    @GetMapping("/service")
    public Page<ServiceDTO> searchService(@RequestParam String pattern, @RequestParam String filter, @PathVariable String env, Pageable pageable) {
        final Set<ServiceDTO> serviceDTOS = providerService.getServiceDTOS(pattern, filter, env);

        final int total = serviceDTOS.size();
        final List<ServiceDTO> content = serviceDTOS.stream().skip(pageable.getOffset()).limit(pageable.getPageSize()).collect(Collectors.toList());

        final Page<ServiceDTO> page = new PageImpl<>(content, pageable, total);
        return page;
    }

    @GetMapping("/service/{service}")
    public ServiceDetailDTO serviceDetail(@PathVariable String service, @PathVariable String env) {
        service = service.replace(Constants.ANY_VALUE, Constants.PATH_SEPARATOR);
        String group = Tool.getGroup(service);
        String version = Tool.getVersion(service);
        String interfaze = Tool.getInterface(service);
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
            try {
                // for dubbo version under 2.7, this metadata will represent as IP address, like 10.0.0.1.
                // So the json conversion will fail.
                String release = providerService.findVersionInApplication(application);
                // serialization compatible 2.x version
                if (release.startsWith("2")) {
                    org.apache.dubbo.admin.model.domain.FullServiceDefinition serviceDefinition = gson.fromJson(metadata, org.apache.dubbo.admin.model.domain.FullServiceDefinition.class);
                    serviceDetailDTO.setMetadata(serviceDefinition);
                } else {
                    FullServiceDefinition serviceDefinition = gson.fromJson(metadata, FullServiceDefinition.class);
                    serviceDetailDTO.setMetadata(serviceDefinition);
                }
            } catch (JsonParseException e) {
                throw new VersionValidationException("dubbo 2.6 does not support metadata");
            }
        }
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        serviceDetailDTO.setService(service);
        serviceDetailDTO.setApplication(application);
        return serviceDetailDTO;
    }

    @GetMapping("/services")
    public Set<String> allServices(@PathVariable String env) {
        return new HashSet<>(providerService.findServices());
    }

    @GetMapping("/applications/instance")
    public Set<String> allInstanceServices(@PathVariable String env) {
        return new HashSet<>(providerService.findInstanceApplications());
    }

    @GetMapping("/applications")
    public Set<String> allApplications(@PathVariable String env) {
        return providerService.findApplications();
    }

    @GetMapping("/consumers")
    public Set<String> allConsumers(@PathVariable String env) {
        List<Consumer> consumers = consumerService.findAll();
        return consumers.stream().map(Consumer::getApplication).collect(Collectors.toSet());
    }

    /**
     * Get all provider's address with specific service name.
     *
     * @param service serviceName
     * @param env     environment
     * @return {@link Set}<{@link String}>
     */
    @GetMapping("/provider/addresses")
    public Set<String> providerAddresses(@RequestParam String service, @PathVariable String env) {
        List<Provider> providers = providerService.findByService(service);
        return Optional.ofNullable(providers).orElse(Collections.emptyList()).stream().map(Provider::getAddress).collect(Collectors.toSet());
    }
}
