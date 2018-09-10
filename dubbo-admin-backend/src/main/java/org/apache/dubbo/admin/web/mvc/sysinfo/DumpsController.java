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
package org.apache.dubbo.admin.web.mvc.sysinfo;

import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.web.mvc.BaseController;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.RequestMapping;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.util.ArrayList;
import java.util.List;

@Controller
@RequestMapping("/sysinfo/dumps")
public class DumpsController extends BaseController {

    @Autowired
    ProviderService providerDAO;

    @Autowired
    ConsumerService consumerDAO;

    @Autowired
    HttpServletResponse response;

    @RequestMapping("")
    public String index(HttpServletRequest request, HttpServletResponse response, Model model) {
        prepare(request, response, model, "index", "dumps");
        model.addAttribute("noProviderServices", getNoProviders());
        model.addAttribute("services", providerDAO.findServices());
        model.addAttribute("providers", providerDAO.findAll());
        model.addAttribute("consumers", consumerDAO.findAll());
        return "sysinfo/screen/dumps/index";
    }

    private List<String> getNoProviders() {
        List<String> providerServices = providerDAO.findServices();
        List<String> consumerServices = consumerDAO.findServices();
        List<String> noProviderServices = new ArrayList<String>();
        if (consumerServices != null) {
            noProviderServices.addAll(consumerServices);
            noProviderServices.removeAll(providerServices);
        }
        return noProviderServices;
    }
}
