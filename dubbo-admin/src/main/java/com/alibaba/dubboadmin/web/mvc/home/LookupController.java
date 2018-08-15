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
package com.alibaba.dubboadmin.web.mvc.home;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.alibaba.dubbo.common.URL;
import com.alibaba.dubbo.registry.RegistryService;
import com.alibaba.dubboadmin.governance.service.ConsumerService;

import org.springframework.beans.factory.annotation.Autowired;

public class LookupController extends RestfulController {

    @Autowired
    ConsumerService consumerDAO;

    @Autowired
    private RegistryService registryService;

    public ResultController doExecute(Map<String, Object> context) throws Exception {
        String inf = request.getParameter("interface");
        if (inf == null || inf.isEmpty()) {
            throw new IllegalArgumentException("please give me the interface");
        }
        String group = null;
        if (inf.contains("/")) {
            int idx = inf.indexOf('/');
            group = inf.substring(idx);
            inf = inf.substring(idx + 1, inf.length());
        }
        String version = null;
        if (inf.contains(":")) {
            int idx = inf.lastIndexOf(':');
            version = inf.substring(idx + 1, inf.length());
            inf = inf.substring(idx);
        }

        String parameters = request.getParameter("parameters");
        String url = "subscribe://" + operatorAddress + "/" + request.getParameter("interface");
        if (parameters != null && parameters.trim().length() > 0) {
            url += parameters.trim();
        }

        URL u = URL.valueOf(url);
        if (group != null) {
            u.addParameter("group", group);
        }

        if (version != null) u.addParameter("version", version);

        List<URL> lookup = registryService.lookup(u);

        Map<String, Map<String, String>> serviceUrl = new HashMap<String, Map<String, String>>();
        Map<String, String> urls = new HashMap<String, String>();
        serviceUrl.put(request.getParameter("interface").trim(), urls);

        for (URL u2 : lookup) {
            urls.put(u2.toIdentityString(), u2.toParameterString());
        }

        ResultController resultController = new ResultController();
        resultController.setMessage(serviceUrl);
        return resultController;
    }

}
