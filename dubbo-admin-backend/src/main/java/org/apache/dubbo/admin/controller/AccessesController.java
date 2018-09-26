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

import com.alibaba.dubbo.common.logger.Logger;
import com.alibaba.dubbo.common.logger.LoggerFactory;
import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.dto.AccessDTO;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.apache.dubbo.admin.registry.common.route.RouteRule;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Resource;
import java.text.ParseException;
import java.util.*;

@RestController
@RequestMapping("/api/access")
public class AccessesController {
    private static final Logger logger = LoggerFactory.getLogger(AccessesController.class);

    @Resource
    private RouteService routeService;
    @Resource
    private ProviderService providerService;

    @RequestMapping("/search")
    public List<AccessDTO> searchAccess(@RequestBody(required = false) Map<String, String> params) throws ParseException {
        List<AccessDTO> result = new ArrayList<>();
        List<Route> routes = new ArrayList<>();
        if (StringUtils.isNotBlank(params.get("service"))) {
            Route route = routeService.getBlackwhitelistRouteByService(params.get("service").trim());
            if (route != null) {
                routes.add(route);
            }
        } else {
            //TODO throw exception
        }

        for (Route route : routes) {
            // Match WhiteBlackList Route
            if (route.getName().endsWith(AccessDTO.KEY_BLACK_WHITE_LIST)) {
                AccessDTO accessDTO = new AccessDTO();
                accessDTO.setId(route.getId());
                accessDTO.setService(route.getService());
                Map<String, RouteRule.MatchPair> when = RouteRule.parseRule(route.getMatchRule());
                for (String key : when.keySet()) {
                    accessDTO.setWhitelist(when.get(key).getUnmatches());
                    accessDTO.setBlacklist(when.get(key).getMatches());
                }
                result.add(accessDTO);
            }
        }
        return result;
    }

    @RequestMapping(value = "/delete", method = RequestMethod.POST)
    public void deleteAccess(@RequestBody Map<String, Long> params) {
        if (params.get("id") == null) {
            throw new IllegalArgumentException("Argument of id is null!");
        }
        routeService.deleteRoute(params.get("id"));
    }

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public void createAccess(@RequestBody AccessDTO accessDTO) {
        if (StringUtils.isBlank(accessDTO.getService())) {
            throw new IllegalArgumentException("Service is required.");
        }
        if (accessDTO.getBlacklist() == null && accessDTO.getWhitelist() == null) {
            throw new IllegalArgumentException("One of Blacklist/Whitelist is required.");
        }

        Route route = routeService.getBlackwhitelistRouteByService(accessDTO.getService());

        if (route != null) {
            throw new IllegalArgumentException(accessDTO.getService() + " is existed.");
        }

        route = new Route();
        route.setService(accessDTO.getService());
        route.setForce(true);
        route.setName(accessDTO.getService() + " " + AccessDTO.KEY_BLACK_WHITE_LIST);
        route.setFilterRule("false");
        route.setEnabled(true);

        Map<String, RouteRule.MatchPair> when = new HashMap<>();
        RouteRule.MatchPair matchPair = new RouteRule.MatchPair(new HashSet<>(), new HashSet<>());
        when.put(Route.KEY_CONSUMER_HOST, matchPair);

        if (accessDTO.getWhitelist() != null) {
            matchPair.getUnmatches().addAll(accessDTO.getWhitelist());
        }
        if (accessDTO.getBlacklist() != null) {
            matchPair.getMatches().addAll(accessDTO.getBlacklist());
        }

        StringBuilder sb = new StringBuilder();
        RouteRule.contidionToString(sb, when);
        route.setMatchRule(sb.toString());
        routeService.createRoute(route);
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public void updateAccess(@RequestBody AccessDTO accessDTO) {
        Route route = routeService.findRoute(accessDTO.getId());
        Map<String, RouteRule.MatchPair> when = new HashMap<>();
        RouteRule.MatchPair matchPair = new RouteRule.MatchPair(new HashSet<>(), new HashSet<>());
        when.put(Route.KEY_CONSUMER_HOST, matchPair);

        if (accessDTO.getWhitelist() != null) {
            matchPair.getUnmatches().addAll(accessDTO.getWhitelist());
        }
        if (accessDTO.getBlacklist() != null) {
            matchPair.getMatches().addAll(accessDTO.getBlacklist());
        }

        StringBuilder sb = new StringBuilder();
        RouteRule.contidionToString(sb, when);
        route.setMatchRule(sb.toString());

        routeService.updateRoute(route);
    }
}
