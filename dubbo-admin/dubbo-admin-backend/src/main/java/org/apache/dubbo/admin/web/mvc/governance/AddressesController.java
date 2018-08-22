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
package org.apache.dubbo.admin.web.mvc.governance;

import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;
import org.apache.dubbo.admin.web.mvc.BaseController;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.util.List;

/**
 * ProvidersController.
 * URI: /services/$service/providers
 *
 */
@Controller
@RequestMapping("/governance/addresses")
public class AddressesController extends BaseController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @Autowired
    ServicesController servicesController;


    @RequestMapping("")
    public String index(@RequestParam String ip, HttpServletRequest request, HttpServletResponse response, Model model) {
        prepare(request, response, model, "index", "addresses");
        List<Provider> providers = null;
        List<Consumer> consumers = null;
        providers = providerService.findByAddress(ip);
        consumers = consumerService.findByAddress(ip);

        model.addAttribute("providers", providers);
        model.addAttribute("consumers", consumers);
        return "";
    }

}
