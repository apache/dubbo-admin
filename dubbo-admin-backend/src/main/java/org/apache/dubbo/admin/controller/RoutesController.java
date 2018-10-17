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

import org.apache.dubbo.admin.dto.BaseDTO;
import org.apache.dubbo.admin.dto.RouteDTO;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.apache.dubbo.admin.util.MD5Util;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;

@RestController
@RequestMapping("/api/{env}/rules/route")
public class RoutesController {

    @Autowired
    private RouteService routeService;
    @Autowired
    private ProviderService providerService;

    @RequestMapping(method = RequestMethod.POST)
    public boolean createRule(@RequestBody RouteDTO routeDTO, @PathVariable String env) {
        String serviceName = routeDTO.getService();
        String app = routeDTO.getApp();
        if (serviceName == null && app == null) {

        }
        if (serviceName != null) {
            //2.6
            String version = null;
            String service = serviceName;
            if (serviceName.contains(":") && !serviceName.endsWith(":")) {
                version = serviceName.split(":")[1];
                service = serviceName.split(":")[0];
            }

            String[] conditions = routeDTO.getConditions();
            String rule = parseCondition(conditions);
            Route route = new Route();
            route.setService(service);
            route.setVersion(version);
            route.setEnabled(routeDTO.isEnabled()); route.setForce(routeDTO.isForce());
            route.setGroup(routeDTO.getGroup());
            route.setDynamic(routeDTO.isDynamic());
            route.setRuntime(routeDTO.isRuntime());
            route.setPriority(routeDTO.getPriority());
            route.setRule(rule);
            routeService.createRoute(route);

        } else {
            //new feature in 2.7
        }
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateRule(@PathVariable String id, @RequestBody RouteDTO routeDTO, @PathVariable String env) {
        Route route = routeService.findRoute(id);
        if (route == null) {
            //TODO Exception
        }
        String[] conditions = routeDTO.getConditions();
        String rule = parseCondition(conditions);
        Route newRoute = new Route();
        newRoute.setService(route.getService());
        newRoute.setVersion(route.getVersion());
        newRoute.setEnabled(routeDTO.isEnabled());
        newRoute.setForce(routeDTO.isForce());
        newRoute.setGroup(routeDTO.getGroup());
        newRoute.setDynamic(routeDTO.isDynamic());
        newRoute.setRuntime(routeDTO.isRuntime());
        newRoute.setPriority(routeDTO.getPriority());
        newRoute.setRule(rule);
        routeService.updateRoute(newRoute);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<RouteDTO> searchRoutes(@RequestParam(required = false) String app,
                                    @RequestParam(required = false) String service, @PathVariable String env) {
        List<Route> routes;
        if (app != null) {
           // app scope in 2.7
        }
        if (service != null) {
            routes = routeService.findByService(service);
        } else {
            routes = routeService.findAll();
        }
        List<RouteDTO> routeDTOS = new ArrayList<>();
        for (Route route : routes) {
            RouteDTO routeDTO = new RouteDTO();
            routeDTO.setDynamic(route.isDynamic());
            routeDTO.setConditions(new String[]{route.getRule()});
            routeDTO.setEnabled(route.isEnabled());
            routeDTO.setForce(route.isForce());
            routeDTO.setGroup(route.getGroup());
            routeDTO.setPriority(route.getPriority());
            routeDTO.setRuntime(route.isRuntime());
            routeDTO.setService(route.getService());
            routeDTO.setId(route.getHash());
            routeDTOS.add(routeDTO);
        }
        return routeDTOS;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public RouteDTO detailRoute(@PathVariable String id, @PathVariable String env) {
        Route route = routeService.findRoute(id);
        if (route == null) {
            // TODO throw exception
        }
        RouteDTO routeDTO = new RouteDTO();
        routeDTO.setDynamic(route.isDynamic());
        routeDTO.setConditions(new String[]{route.getRule()});
        routeDTO.setEnabled(route.isEnabled());
        routeDTO.setForce(route.isForce());
        routeDTO.setGroup(route.getGroup());
        routeDTO.setPriority(route.getPriority());
        routeDTO.setRuntime(route.isRuntime());
        routeDTO.setService(route.getService());
        routeDTO.setId(route.getHash());
        return routeDTO;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteRoute(@PathVariable String id, @PathVariable String env) {
        routeService.deleteRoute(id);
        return true;
    }

    @RequestMapping(value = "/enable/{id}", method = RequestMethod.PUT)
    public boolean enableRoute(@PathVariable String id, @PathVariable String env) {

        routeService.enableRoute(id);
        return true;
    }

    @RequestMapping(value = "/disable/{id}", method = RequestMethod.PUT)
    public boolean disableRoute(@PathVariable String id, @PathVariable String env) {

        routeService.disableRoute(id);
        return true;
    }

    private String parseCondition(String[] conditions) {
        StringBuilder when = new StringBuilder();
        StringBuilder then = new StringBuilder();
        for (String condition : conditions) {
            condition = condition.trim();
            if (condition.contains("=>") && !condition.endsWith("=>")) {
                String consumer = condition.split("=>")[0].trim();
                String provider = condition.split("=>")[1].trim();
                if (when.length() != 0) {
                    when.append(" & ").append(consumer);
                } else {
                    when.append(consumer);
                }

                if (then.length() != 0) {
                    then.append(" & ").append(provider);
                } else {
                    then.append(provider);
                }
            }
        }
        return (when.append(" => ").append(then)).toString();
    }

    public static void main(String[] args) {
        String a = "fwjiojie =>";
        a.split("=>", 2);
    }

}