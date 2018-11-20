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
import java.util.regex.Matcher;
import java.util.regex.Pattern;


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
    public Set<ServiceDTO> searchService(@RequestParam String pattern,
                                         @RequestParam String filter,@PathVariable String env) {

        return providerService.getServiceDTOS(pattern, filter, env);
    }



    @RequestMapping(value = "/{service}", method = RequestMethod.GET)
    public ServiceDetailDTO serviceDetail(@PathVariable String service, @PathVariable String env) {
        service = service.replace("*", "/");
        List<Provider> providers = providerService.findByService(service);

        List<Consumer> consumers = consumerService.findByService(service);

        ServiceDetailDTO serviceDetailDTO = new ServiceDetailDTO();
        serviceDetailDTO.setConsumers(consumers);
        serviceDetailDTO.setProviders(providers);
        return serviceDetailDTO;
    }
}
