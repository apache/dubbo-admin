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

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.dto.BalancingDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.LoadBalance;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.apache.dubbo.admin.registry.common.util.OverrideUtils;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.util.StringUtils;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import java.util.ArrayList;
import java.util.List;

import static org.apache.dubbo.admin.registry.common.util.OverrideUtils.overrideToLoadBalance;

@RestController
@RequestMapping("/api/{env}/rules/balancing")
public class LoadBalanceController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createLoadbalance(@RequestBody BalancingDTO balancingDTO, @PathVariable String env) throws ParamValidationException {
        String serviceName = balancingDTO.getService();
        if (StringUtils.isEmpty(serviceName)) {
            throw new ParamValidationException("serviceName is Empty!");
        }
        LoadBalance loadBalance = new LoadBalance();
        loadBalance.setService(serviceName);
        loadBalance.setMethod(formatMethodName(balancingDTO.getMethodName()));
        loadBalance.setStrategy(balancingDTO.getStrategy());
        overrideService.saveOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateLoadbalance(@PathVariable String id, @RequestBody BalancingDTO balancingDTO, @PathVariable String env) throws ParamValidationException {
        Override override = overrideService.findById(id);
        if (override == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        LoadBalance old = overrideToLoadBalance(override);
        LoadBalance loadBalance = new LoadBalance();
        loadBalance.setStrategy(balancingDTO.getStrategy());
        loadBalance.setMethod(formatMethodName(balancingDTO.getMethodName()));
        loadBalance.setService(old.getService());
        loadBalance.setHash(id);
        overrideService.updateOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<BalancingDTO> searchLoadbalances(@RequestParam(required = false) String service, @PathVariable String env) {
        List<Override> overrides;
        if (service == null || service.length() == 0) {
            overrides = overrideService.findAll();
           //TODO throw Exception
        } else {
            overrides = overrideService.findByService(service);
        }
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

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public BalancingDTO detailLoadBalance(@PathVariable String id, @PathVariable String env) throws ParamValidationException {
        Override override =  overrideService.findById(id);
        if (override == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }

        LoadBalance loadBalance = OverrideUtils.overrideToLoadBalance(override);
        BalancingDTO balancingDTO = new BalancingDTO();
        balancingDTO.setService(loadBalance.getService());
        balancingDTO.setMethodName(loadBalance.getMethod());
        balancingDTO.setStrategy(loadBalance.getStrategy());
        return balancingDTO;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteLoadBalance(@PathVariable String id, @PathVariable String env) {
        if (id == null) {
            throw new IllegalArgumentException("Argument of id is null!");
        }
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
