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

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.model.dto.BalancingDTO;
import org.apache.dubbo.admin.service.OverrideService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;


@RestController
@RequestMapping("/api/{env}/rules/balancing")
public class LoadBalanceController {

    private final OverrideService overrideService;

    @Autowired
    public LoadBalanceController(OverrideService overrideService) {
        this.overrideService = overrideService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createLoadbalance(@RequestBody BalancingDTO balancingDTO, @PathVariable String env) throws ParamValidationException {
        if (StringUtils.isBlank(balancingDTO.getService()) && StringUtils.isBlank(balancingDTO.getApplication())) {
            throw new ParamValidationException("Either Service or application is required.");
        }
        overrideService.saveBalance(balancingDTO);
//        String serviceName = balancingDTO.getService();
//        if (StringUtils.isEmpty(serviceName)) {
//            throw new ParamValidationException("serviceName is Empty!");
//        }
//        LoadBalance loadBalance = new LoadBalance();
//        loadBalance.setService(serviceName);
//        loadBalance.setMethod(formatMethodName(balancingDTO.getMethodName()));
//        loadBalance.setStrategy(balancingDTO.getStrategy());
//        overrideService.saveOverride(OverrideUtils.loadBalanceToOverride(loadBalance));
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateLoadbalance(@PathVariable String id, @RequestBody BalancingDTO balancingDTO, @PathVariable String env) throws ParamValidationException {
        if (id == null) {
            throw new ParamValidationException("Unknown ID!");
        }
        id = id.replace("*", "/");
        BalancingDTO balancing = overrideService.findBalance(id);
        if (balancing == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }

        overrideService.saveBalance(balancingDTO);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<BalancingDTO> searchLoadbalances(@RequestParam(required = false) String service,
                                                 @RequestParam(required = false) String application,
                                                 @PathVariable String env) {

        if (StringUtils.isBlank(service) && StringUtils.isBlank(application)) {
            throw new ParamValidationException("Either service or application is required");
        }
        BalancingDTO balancingDTO;
        if (StringUtils.isNotBlank(application)) {
            balancingDTO = overrideService.findBalance(application);
        } else {
            balancingDTO = overrideService.findBalance(service);
        }
        List<BalancingDTO> balancingDTOS = new ArrayList<>();
        if (balancingDTO != null) {
            balancingDTOS.add(balancingDTO);
        }
        return balancingDTOS;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public BalancingDTO detailLoadBalance(@PathVariable String id, @PathVariable String env) throws ParamValidationException {
        id = id.replace("*", "/");
        BalancingDTO balancingDTO = overrideService.findBalance(id);
        if (balancingDTO == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        return balancingDTO;

//        LoadBalance loadBalance = OverrideUtils.overrideToLoadBalance(override);
//        BalancingDTO balancingDTO = new BalancingDTO();
//        balancingDTO.setService(loadBalance.getService());
//        balancingDTO.setMethodName(loadBalance.getMethod());
//        balancingDTO.setStrategy(loadBalance.getStrategy());
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteLoadBalance(@PathVariable String id, @PathVariable String env) {
        id = id.replace("*", "/");
        if (id == null) {
            throw new IllegalArgumentException("Argument of id is null!");
        }
        overrideService.deleteBalance(id);
        return true;
    }


}
