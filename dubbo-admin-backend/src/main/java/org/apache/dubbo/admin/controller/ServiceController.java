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

import java.util.ArrayList;
import java.util.List;
import java.util.Map;


@RestController
@RequestMapping("/api/{env}/service")
public class ServiceController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @RequestMapping(method = RequestMethod.GET)
    public List<ServiceDTO> searchService(@RequestParam String pattern,
                                   @RequestParam(required = false) String filter) {

        List<Provider> allProviders = providerService.findAll();

        List<ServiceDTO> result = new ArrayList<>();
        for (Provider provider : allProviders) {
            Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
            ServiceDTO s = new ServiceDTO();
            if (filter == null || filter.length() == 0) {
                s.setAppName(provider.getApplication());
                s.setService(provider.getService());
                s.setGroup(map.get(Constants.GROUP_KEY));
                s.setVersion(map.get(Constants.VERSION_KEY));
                result.add(s);
            } else {
                filter = filter.toLowerCase();
                String key = null;
                switch (pattern) {
                    case "application":
                        key = provider.getApplication().toLowerCase();
                        break;
                    case "service name":
                        key = provider.getService().toLowerCase();
                        break;
                    case "IP":
                        key = provider.getService().toLowerCase();
                        break;
                }
                if (key.contains(filter)) {
                    result.add(createService(provider, map));
                }
            }
        }
        return result;
    }

    @RequestMapping("/{app}/{service}")
    public ServiceDetailDTO serviceDetail(@PathVariable String app, @PathVariable String service) {
        List<Provider> providers = providerService.findByAppandService(app, service);

        List<Consumer> consumers = consumerService.findByAppandService(app, service);

        ServiceDetailDTO serviceDetailDTO = new ServiceDetailDTO();
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        return serviceDetailDTO;
    }

    private ServiceDTO createService(Provider provider, Map<String, String> map) {
        ServiceDTO serviceDTO = new ServiceDTO();
        serviceDTO.setAppName(provider.getApplication());
        serviceDTO.setService(provider.getService());
        serviceDTO.setGroup(map.get(Constants.GROUP_KEY));
        serviceDTO.setVersion(map.get(Constants.VERSION_KEY));
        return serviceDTO;
    }

}
