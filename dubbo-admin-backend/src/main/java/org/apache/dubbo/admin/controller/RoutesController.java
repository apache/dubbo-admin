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

import org.apache.dubbo.admin.dto.RouteDTO;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/routes")
public class RoutesController {

    @Autowired
    private RouteService routeService;
    @Autowired
    private ProviderService providerService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createRule(@RequestBody RouteDTO routeDTO) {
        String serviceName = routeDTO.getServiceName();
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
        Long id = routeDTO.getId();
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
            newRoute.setId(id);
            routeService.updateRoute(newRoute);
        }
        return true;
    }

    @RequestMapping(value = "/search", method = RequestMethod.POST)
    public List<Route> allRoutes(@RequestBody Map<String, String> params) {
        String app = params.get("app");
        String serviceName = params.get("serviceName");
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
        //no support for findAll and findByaddress
        return routes;
    }

    @RequestMapping("/detail")
    public Route routeDetail(@RequestParam long id) {
        Route route = routeService.findRoute(id);
        if (route == null) {
            // TODO throw exception
        }
        return route;
    }

    @RequestMapping(value = "/delete", method = RequestMethod.POST)
    public boolean deleteRoute(@RequestBody Map<String, Long> params) {
        Long id = params.get("id");
        routeService.deleteRoute(id);
        return true;
    }

    @RequestMapping(value = "/edit", method = RequestMethod.POST)
    public Route editRule(@RequestBody Map<String, Long> params) {
        Long id = params.get("id");
        Route route = routeService.findRoute(id);
        if (route == null) {
            // TODO throw exception
        }
        return route;
    }

    @RequestMapping(value = "/changeStatus", method = RequestMethod.POST)
    public boolean enableRoute(@RequestBody Map<String, Object> params) {
        boolean enabled = (boolean)params.get("enabled");

        long id = Long.parseLong(params.get("id").toString());
        if (enabled) {
            routeService.disableRoute(id);
        } else {
            routeService.enableRoute(id);
        }
        return true;
    }

    private Object getParameter(Map<String, Object> result, String key, Object defaultValue) {
        if (result.get(key) != null) {
            return result.get(key);
        }
        return defaultValue;
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