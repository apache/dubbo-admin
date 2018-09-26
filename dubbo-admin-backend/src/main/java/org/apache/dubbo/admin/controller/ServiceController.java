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

import org.apache.dubbo.admin.domain.ServiceDO;
import org.apache.dubbo.admin.domain.ServiceDetailDO;
import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.utils.StringUtils;
import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;


@RestController
@RequestMapping("/api/service")
public class ServiceController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @RequestMapping(value = "/search", method = RequestMethod.POST)
    public List<ServiceDO> search(@RequestBody Map<String, String> params) {

        String pattern = params.get("pattern");
        String filter = params.get("filter");

        List<Provider> allProviders = providerService.findAll();

        List<ServiceDO> result = new ArrayList<>();
        if (pattern.equals("application")) {
            for (Provider provider : allProviders) {
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                String app = map.get(Constants.APPLICATION_KEY);
                if (app.toLowerCase().contains(filter)) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(app);
                    s.setServiceName(provider.getService());
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }
            }

        } else if (pattern.equals("service name")) {
            for (Provider provider : allProviders) {
                String service = provider.getService();
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                if (service.toLowerCase().contains(filter.toLowerCase())) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(map.get(Constants.APPLICATION_KEY));
                    s.setServiceName(service);
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }
            }

        } else if (pattern.equals("IP")) {
            for (Provider provider : allProviders) {
                String address = provider.getAddress();
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                if (address.contains(filter)) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(map.get(Constants.APPLICATION_KEY));
                    s.setServiceName(provider.getService());
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }

            }
        }
        return result;
    }

    @RequestMapping("/detail")
    public ServiceDetailDO serviceDetail(@RequestParam String app, @RequestParam String service) {
        List<Provider> providers = providerService.findByAppandService(app, service);

        List<Consumer> consumers = consumerService.findByAppandService(app, service);

        ServiceDetailDO serviceDetailDO = new ServiceDetailDO();
        serviceDetailDO.setConsumers(consumers);
        serviceDetailDO.setProviders(providers);
        return serviceDetailDO;
    }

}
