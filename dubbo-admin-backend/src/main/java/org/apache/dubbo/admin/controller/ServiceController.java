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

import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.utils.StringUtils;
import org.apache.dubbo.admin.dto.ServiceDTO;
import org.apache.dubbo.admin.dto.ServiceDetailDTO;
import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.*;


@RestController
@RequestMapping("/api/{env}/service")
public class ServiceController {

    private final ProviderService providerService;
    private final ConsumerService consumerService;

    @Autowired
    public ServiceController(ProviderService providerService, ConsumerService consumerService) {
        this.providerService = providerService;
        this.consumerService = consumerService;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<ServiceDTO> searchService(@RequestParam String pattern,
                                          @RequestParam(required = false) String filter) {

        List<Provider> allProviders = providerService.findAll();
        Set<String> serviceUrl = new HashSet<>();

        List<ServiceDTO> result = new ArrayList<>();
        for (Provider provider : allProviders) {
            Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
            String app = provider.getApplication();
            String service = map.get(Constants.INTERFACE_KEY);
            String group = map.get(Constants.GROUP_KEY);
            String version = map.get(Constants.VERSION_KEY);
            String url = app + service + group + version;
            if (serviceUrl.contains(url)) {
                continue;
            }
            ServiceDTO s = new ServiceDTO();
            if (org.apache.commons.lang3.StringUtils.isEmpty(filter)) {
                s.setAppName(app);
                s.setService(service);
                s.setGroup(group);
                s.setVersion(version);
                result.add(s);
                serviceUrl.add(url);
            } else {
                filter = filter.toLowerCase();
                String key = null;
                switch (pattern) {
                    case "application":
                        key = provider.getApplication().toLowerCase();
                        break;
                    case "serviceName":
                        key = provider.getService().toLowerCase();
                        break;
                    case "ip":
                        key = provider.getService().toLowerCase();
                        break;
                }
                if (key != null && key.contains(filter)) {
                    result.add(createService(provider, map));
                    serviceUrl.add(url);
                }
            }
        }
        return result;
    }

    @RequestMapping(value = "/{service}", method = RequestMethod.GET)
    public ServiceDetailDTO serviceDetail(@PathVariable String service) {
        service = service.replace("*", "/");
        List<Provider> providers = providerService.findByService(service);

        List<Consumer> consumers = consumerService.findByService(service);

        ServiceDetailDTO serviceDetailDTO = new ServiceDetailDTO();
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        return serviceDetailDTO;
    }

    private ServiceDTO createService(Provider provider, Map<String, String> map) {
        ServiceDTO serviceDTO = new ServiceDTO();
        serviceDTO.setAppName(provider.getApplication());
        String service = map.get(Constants.INTERFACE_KEY);
        String group = map.get(Constants.GROUP_KEY);
        String version = map.get(Constants.VERSION_KEY);
        serviceDTO.setService(service);
        serviceDTO.setGroup(group);
        serviceDTO.setVersion(version);
        return serviceDTO;
    }

}
