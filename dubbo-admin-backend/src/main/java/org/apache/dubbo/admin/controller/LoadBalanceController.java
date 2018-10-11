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
import org.apache.dubbo.admin.dto.BaseDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.LoadBalance;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.apache.dubbo.admin.registry.common.util.OverrideUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;

import static org.apache.dubbo.admin.registry.common.util.OverrideUtils.overrideToLoadBalance;

@RestController
@RequestMapping("/api/balancing")
public class LoadBalanceController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createLoadbalance(@RequestBody BalancingDTO balancingDTO) {
        String serviceName = balancingDTO.getService();
        if (serviceName == null || serviceName.length() == 0) {
            //TODO throw exception
        }
        LoadBalance loadBalance = new LoadBalance();
        loadBalance.setService(serviceName);
        loadBalance.setMethod(formatMethodName(balancingDTO.getMethodName()));
        loadBalance.setStrategy(balancingDTO.getStrategy());
        overrideService.saveOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public boolean updateLoadbalance(@RequestBody BalancingDTO balancingDTO) {
        String id = balancingDTO.getId();
        Override override = overrideService.findById(id);
        if (override == null) {
            //TODO throw exception
        }
        LoadBalance old = overrideToLoadBalance(override);
        LoadBalance loadBalance = new LoadBalance();
        loadBalance.setStrategy(balancingDTO.getStrategy());
        loadBalance.setMethod(formatMethodName(balancingDTO.getMethodName()));
        loadBalance.setService(old.getService());
        loadBalance.setId(old.getId());
        overrideService.updateOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/search", method = RequestMethod.GET)
    public List<BalancingDTO> allLoadbalances(@RequestParam String serviceName) {
        if (serviceName == null || serviceName.length() == 0) {
           //TODO throw Exception
        }
        List<Override> overrides = overrideService.findByService(serviceName);
        List<BalancingDTO> loadBalances = new ArrayList<>();
        if (overrides != null) {
            for (Override override : overrides) {
                LoadBalance l = overrideToLoadBalance(override);
                if (l != null) {
                    BalancingDTO balancingDTO = new BalancingDTO();
                    balancingDTO.setService(l.getService());
                    balancingDTO.setMethodName(l.getMethod());
                    balancingDTO.setStrategy(l.getStrategy());
                    balancingDTO.setId(l.getHash());
                    loadBalances.add(balancingDTO);
                }
            }
        }
        return loadBalances;
    }

    @RequestMapping("/detail")
    public BalancingDTO detail(@RequestParam String id) {
        Override override =  overrideService.findById(id);
        if (override == null) {
            //TODO throw exception
        }

        LoadBalance loadBalance = OverrideUtils.overrideToLoadBalance(override);
        BalancingDTO balancingDTO = new BalancingDTO();
        balancingDTO.setService(loadBalance.getService());
        balancingDTO.setMethodName(loadBalance.getMethod());
        balancingDTO.setStrategy(loadBalance.getStrategy());
        return balancingDTO;
    }

    @RequestMapping(value  = "/delete", method = RequestMethod.POST)
    public boolean delete(@RequestBody BaseDTO baseDTO) {
        String id = baseDTO.getId();
        overrideService.deleteOverride(id);
        return true;
    }

    private String formatMethodName(String method) {
        if (method.equals("0")) {
            return "*";
        }
        return method;
    }
}
