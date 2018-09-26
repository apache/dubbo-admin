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

import org.apache.dubbo.admin.dto.BalancingDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.LoadBalance;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.apache.dubbo.admin.registry.common.util.OverrideUtils;
import org.apache.dubbo.admin.util.YamlUtil;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/balancing")
public class LoadBalanceController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createLoadbalance(@RequestBody BalancingDTO balancingDTO) {
        String serviceName = balancingDTO.getServiceName();
        String rule = balancingDTO.getRule();
        if (serviceName == null || serviceName.length() == 0) {
            //TODO throw exception
        }

        Map<String, Object> result = YamlUtil.loadString(rule);
        LoadBalance loadBalance = generateLoadbalance(result);
        loadBalance.setService(serviceName);
        overrideService.saveOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public boolean updateLoadbalance(@RequestBody BalancingDTO balancingDTO) {
        Long id = balancingDTO.getId();
        String rule = balancingDTO.getRule();
        Override override = overrideService.findById(id);
        if (override == null) {
            //TODO throw exception
        }
        LoadBalance old = OverrideUtils.overrideToLoadBalance(override);
        Map<String, Object> result = YamlUtil.loadString(rule);
        LoadBalance loadBalance = generateLoadbalance(result);
        loadBalance.setService(old.getService());
        loadBalance.setId(old.getId());
        overrideService.updateOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/search", method = RequestMethod.POST)
    public List<LoadBalance> allLoadbalances(@RequestBody Map<String, String> params) {
        String serviceName = params.get(params);
        if (serviceName == null || serviceName.length() == 0) {
           //TODO throw Exception
        }
        List<Override> overrides = overrideService.findByService(serviceName);
        List<LoadBalance> loadBalances = new ArrayList<>();
        if (overrides != null) {
            for (Override override : overrides) {
                LoadBalance l = OverrideUtils.overrideToLoadBalance(override);
                if (l != null) {
                    loadBalances.add(l);
                }
            }
        }
        return loadBalances;
    }

    @RequestMapping("/detail")
    public LoadBalance detail(@RequestParam Long id) {
        Override override =  overrideService.findById(id);
        if (override == null) {
            //TODO throw exception
        }
        return OverrideUtils.overrideToLoadBalance(override);
    }

    @RequestMapping(value  = "/delete", method = RequestMethod.POST)
    public boolean delete(@RequestBody Map<String, Long> params) {
        Long id = params.get("id");
        overrideService.deleteOverride(id);
        return true;
    }

    private LoadBalance generateLoadbalance(Map<String, Object> yaml) {
        LoadBalance loadBalance = new LoadBalance();
        String methodName;
        if (yaml.get("methodName").equals(0)) {
            methodName = "*";
        } else {
            methodName = (String)yaml.get("methodName");
        }
        String strategy = (String)yaml.get("strategy");
        loadBalance.setMethod(methodName);
        loadBalance.setStrategy(strategy);
        return loadBalance;
    }
}
