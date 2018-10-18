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

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.dto.AccessDTO;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.apache.dubbo.admin.registry.common.route.RouteRule;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Resource;
import java.text.ParseException;
import java.util.*;

@RestController
@RequestMapping("/api/{env}/rules/access")
public class AccessesController {
    private static final Logger logger = LoggerFactory.getLogger(AccessesController.class);

    @Resource
    private RouteService routeService;

    @RequestMapping(method = RequestMethod.GET)
    public List<AccessDTO> searchAccess(@RequestParam(required = false) String service, @PathVariable String env) throws ParseException {
        List<AccessDTO> result = new ArrayList<>();
        List<Route> routes = new ArrayList<>();
        if (StringUtils.isNotBlank(service)) {
            Route route = routeService.getBlackwhitelistRouteByService(service.trim());
            if (route != null) {
                routes.add(route);
            }
        } else {
            routes = routeService.findAllForceRoute();
        }

        for (Route route : routes) {
            // Match WhiteBlackList Route
            if (route.getName().endsWith(AccessDTO.KEY_BLACK_WHITE_LIST)) {
                AccessDTO accessDTO = new AccessDTO();
                accessDTO.setId(route.getHash());
                accessDTO.setService(route.getService());
                Map<String, RouteRule.MatchPair> when = RouteRule.parseRule(route.getMatchRule());
                for (String key : when.keySet()) {
                    accessDTO.setWhitelist(when.get(key).getUnmatches());
                    accessDTO.setBlacklist(when.get(key).getMatches());
                }
                result.add(accessDTO);
            }
        }
throw new ParseException("222",3);
//        return result;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public AccessDTO detailAccess(@PathVariable String id, @PathVariable String env) throws ParseException {
        Route route = routeService.findRoute(id);
        if (route.getName().endsWith(AccessDTO.KEY_BLACK_WHITE_LIST)) {
            AccessDTO accessDTO = new AccessDTO();
            accessDTO.setId(route.getHash());
            accessDTO.setService(route.getService());
            Map<String, RouteRule.MatchPair> when = RouteRule.parseRule(route.getMatchRule());
            for (String key : when.keySet()) {
                accessDTO.setWhitelist(when.get(key).getUnmatches());
                accessDTO.setBlacklist(when.get(key).getMatches());
            }
            return accessDTO;
        } else {
            return null;
        }
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public void deleteAccess(@PathVariable String id, @PathVariable String env) {
        routeService.deleteRoute(id);
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public void createAccess(@RequestBody AccessDTO accessDTO, @PathVariable String env) {
        if (StringUtils.isBlank(accessDTO.getService())) {
            throw new ParamValidationException("Service is required.");
        }
        if (accessDTO.getBlacklist() == null && accessDTO.getWhitelist() == null) {
            throw new ParamValidationException("One of Blacklist/Whitelist is required.");
        }

        Route route = routeService.getBlackwhitelistRouteByService(accessDTO.getService());

        if (route != null) {
            throw new ParamValidationException(accessDTO.getService() + " is existed.");
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

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public void updateAccess(@PathVariable String id, @RequestBody AccessDTO accessDTO, @PathVariable String env) {
        Route route = routeService.findRoute(id);
        if (Objects.isNull(route)) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
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
