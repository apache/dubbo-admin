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
@RequestMapping("/api/routes")
public class RoutesController {

    @Autowired
    private RouteService routeService;
    @Autowired
    private ProviderService providerService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createRule(@RequestBody RouteDTO routeDTO) {
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
            for (String condition : conditions) {
                Route route = new Route();
                route.setService(service);
                route.setVersion(version);
                route.setEnabled(routeDTO.isEnabled());
                route.setForce(routeDTO.isForce());
                route.setGroup(routeDTO.getGroup());
                route.setDynamic(routeDTO.isDynamic());
                route.setRuntime(routeDTO.isRuntime());
                route.setPriority(routeDTO.getPriority());
                route.setRule(condition);
                routeService.createRoute(route);
            }

        } else {
            //new feature in 2.7
        }
        return true;
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public boolean updateRule(@RequestBody RouteDTO routeDTO) {
        String id = routeDTO.getId();
        Route route = routeService.findRoute(id);
        if (route == null) {
            //TODO Exception
        }
        String[] conditions = routeDTO.getConditions();
        for (String condition : conditions) {
            Route newRoute = new Route();
            newRoute.setService(route.getService());
            newRoute.setVersion(route.getVersion());
            newRoute.setEnabled(routeDTO.isEnabled());
            newRoute.setForce(routeDTO.isForce());
            newRoute.setGroup(routeDTO.getGroup());
            newRoute.setDynamic(routeDTO.isDynamic());
            newRoute.setRuntime(routeDTO.isRuntime());
            newRoute.setPriority(routeDTO.getPriority());
            newRoute.setRule(condition);
            routeService.updateRoute(newRoute);
        }
        return true;
    }

    @RequestMapping(value = "/search", method = RequestMethod.GET)
    public List<RouteDTO> allRoutes(@RequestParam(required = false) String app,
                                    @RequestParam(required = false) String serviceName) {
        List<Route> routes = null;
        if (app != null) {
           // app scope in 2.7
        }
        if (serviceName != null) {
            routes = routeService.findByService(serviceName);
        }
        if (serviceName == null && app == null) {
            // TODO throw exception
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
            routeDTO.setId(MD5Util.MD5_16bit(route.toUrl().toFullString()));
            routeDTOS.add(routeDTO);
        }
        //no support for findAll or findByaddress
        return routeDTOS;
    }

    @RequestMapping("/detail")
    public RouteDTO routeDetail(@RequestParam String id) {
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

    @RequestMapping(value = "/delete", method = RequestMethod.POST)
    public boolean deleteRoute(@RequestBody BaseDTO baseDTO) {
        String id = baseDTO.getId();
        routeService.deleteRoute(id);
        return true;
    }

    @RequestMapping(value = "/enable", method = RequestMethod.POST)
    public boolean enableRoute(@RequestBody BaseDTO baseDTO) {

        String id = baseDTO.getId();
        routeService.enableRoute(id);
        return true;
    }

    @RequestMapping(value = "/disable", method = RequestMethod.POST)
    public boolean disableRoute(@RequestBody BaseDTO baseDTO) {

        String id = baseDTO.getId();
        routeService.disableRoute(id);
        return true;
    }

    public static void main(String[] args) {
        String yaml =
                "enable: true\n" +
                        "priority: 0\n" +
                        "runtime: true\n" +
                        "category: routers\n" +
                        "dynamic: true\n" +
                        "conditions:\n" +
                        "  - '=> host != 172.22.3.91'\n" +
                        "  - 'host != 10.20.153.10,10.20.153.11 =>'\n" +
                        "  - 'host = 10.20.153.10,10.20.153.11 =>'\n" +
                        "  - 'application != kylin => host != 172.22.3.95,172.22.3.96'\n" +
                        "  - 'method = find*,list*,get*,is* => host = 172.22.3.94,172.22.3.95,172.22.3.96'";
    }

}