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
import org.apache.dubbo.admin.model.domain.ConditionRoute;
import org.apache.dubbo.admin.model.dto.ConditionRouteDTO;
import org.apache.dubbo.admin.service.RouteService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

@RestController
@RequestMapping("/api/{env}/rules/route/condition")
public class ConditionRoutesController {

    private final RouteService routeService;

    @Autowired
    public ConditionRoutesController(RouteService routeService) {
        this.routeService = routeService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createRule(@RequestBody ConditionRouteDTO routeDTO, @PathVariable String env) {
        String serviceName = routeDTO.getService();
        String app = routeDTO.getApplication();
        if (StringUtils.isEmpty(serviceName) && StringUtils.isEmpty(app)) {
            throw new ParamValidationException("serviceName and app is Empty!");
        }
        ConditionRoute conditionRoute = convertRouteDTOtoRoute(routeDTO);
        routeService.createConditionRoute(conditionRoute);
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateRule(@PathVariable String id, @RequestBody ConditionRouteDTO routeDTO, @PathVariable String env) {
        if (routeService.findConditionRoute(id) == null) {
           //throw exception
        }
        ConditionRoute newConditionRoute = convertRouteDTOtoRoute(routeDTO);
        routeService.updateConditionRoute(newConditionRoute);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<ConditionRouteDTO> searchRoutes(@RequestParam(required = false) String application,
                                                @RequestParam(required = false) String serviceName, @PathVariable String env) {
        ConditionRoute conditionRoute = null;
        List<ConditionRouteDTO> result = new ArrayList<>();
        if (StringUtils.isNotEmpty(application)) {
            conditionRoute = routeService.findConditionRoute(application);
        }
        if (StringUtils.isNotEmpty(serviceName)) {
            conditionRoute = routeService.findConditionRoute(serviceName);
        }
        ConditionRouteDTO routeDTO = convertRouteToRouteDTO(conditionRoute);
        if (routeDTO != null) {
            result.add(routeDTO);
        }
        return result;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public ConditionRouteDTO detailRoute(@PathVariable String id, @PathVariable String env) {
        ConditionRoute conditionRoute = routeService.findConditionRoute(id);
        if (conditionRoute == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        ConditionRouteDTO routeDTO = convertRouteToRouteDTO(conditionRoute);
        return routeDTO;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteRoute(@PathVariable String id, @PathVariable String env) {
        routeService.deleteConditionRoute(id);
        return true;
    }

    @RequestMapping(value = "/enable/{id}", method = RequestMethod.PUT)
    public boolean enableRoute(@PathVariable String id, @PathVariable String env) {
        routeService.enableConditionRoute(id);
        return true;
    }

    @RequestMapping(value = "/disable/{id}", method = RequestMethod.PUT)
    public boolean disableRoute(@PathVariable String id, @PathVariable String env) {
        routeService.disableConditionRoute(id);
        return true;
    }

    private String parseCondition(String[] conditions) {
        StringBuilder when = new StringBuilder();
        StringBuilder then = new StringBuilder();
        for (String condition : conditions) {
            condition = condition.trim();
            if (condition.contains("=>")) {
                String[] array = condition.split("=>", 2);
                String consumer = array[0].trim();
                String provider = array[1].trim();
                if (consumer != "") {
                    if (when.length() != 0) {
                        when.append(" & ").append(consumer);
                    } else {
                        when.append(consumer);
                    }
                }
                if (provider != "") {
                    if (then.length() != 0) {
                        then.append(" & ").append(provider);
                    } else {
                        then.append(provider);
                    }
                }
            }
        }
        return (when.append(" => ").append(then)).toString();
    }

    private ConditionRoute convertRouteDTOtoRoute(ConditionRouteDTO routeDTO) {
        if (routeDTO == null) {
            return null;
        }
        ConditionRoute conditionRoute = new ConditionRoute();
        conditionRoute.setConditions(routeDTO.getConditions());
        conditionRoute.setEnabled(routeDTO.isEnabled());
        conditionRoute.setForce(routeDTO.isForce());
        conditionRoute.setRuntime(routeDTO.isRuntime());
        conditionRoute.setPriority(routeDTO.getPriority());
        if (StringUtils.isNotEmpty(routeDTO.getApplication())) {
            conditionRoute.setScope("application");
            conditionRoute.setKey(routeDTO.getApplication());
        } else {
            conditionRoute.setScope("service");
            conditionRoute.setKey(routeDTO.getService());
        }
        return conditionRoute;
    }

    private ConditionRouteDTO convertRouteToRouteDTO(ConditionRoute conditionRoute) {
        if (conditionRoute == null) {
            return null;
        }
        ConditionRouteDTO routeDTO = new ConditionRouteDTO();
        routeDTO.setConditions(conditionRoute.getConditions());
        routeDTO.setEnabled(conditionRoute.isEnabled());
        routeDTO.setForce(conditionRoute.isForce());
        routeDTO.setPriority(conditionRoute.getPriority());
        routeDTO.setRuntime(conditionRoute.isRuntime());
        if (conditionRoute.getScope().equals("application")) {
            routeDTO.setApplication(conditionRoute.getKey());
        } else {
            routeDTO.setService(conditionRoute.getKey());
        }
        return routeDTO;
    }
}