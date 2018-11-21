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
package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.common.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.admin.model.dto.AccessDTO;
import org.apache.dubbo.admin.service.RouteService;
import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Route;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * IbatisRouteService
 *
 */
@Component
public class RouteServiceImpl extends AbstractService implements RouteService {

    public void createRoute(Route route) {
        registry.register(route.toUrl());
    }

    public void updateRoute(Route route) {
        String hash = route.getHash();
        if (hash == null) {
            throw new IllegalStateException("no route hash");
        }
        URL oldRoute = findRouteUrl(hash);
        if (oldRoute == null) {
            throw new IllegalStateException("Route was changed!");
        }

        registry.unregister(oldRoute);
        registry.register(route.toUrl());
    }

    public void deleteRoute(String id) {
        URL oldRoute = findRouteUrl(id);
        if (oldRoute == null) {
            throw new IllegalStateException("Route was changed!");
        }
        registry.unregister(oldRoute);
    }

    public void enableRoute(String id) {
        if (id == null) {
            throw new IllegalStateException("no route id");
        }

        URL oldRoute = findRouteUrl(id);
        if (oldRoute == null) {
            throw new IllegalStateException("Route was changed!");
        }
        if (oldRoute.getParameter("enabled", true)) {
            return;
        }

        registry.unregister(oldRoute);
        URL newRoute = oldRoute.addParameter("enabled", true);
        registry.register(newRoute);

    }

    public void disableRoute(String id) {
        if (id == null) {
            throw new IllegalStateException("no route id");
        }

        URL oldRoute = findRouteUrl(id);
        if (oldRoute == null) {
            throw new IllegalStateException("Route was changed!");
        }
        if (!oldRoute.getParameter("enabled", true)) {
            return;
        }

        URL newRoute = oldRoute.addParameter("enabled", false);
        registry.unregister(oldRoute);
        registry.register(newRoute);

    }

    public List<Route> findAll() {
        return SyncUtils.url2RouteList(findAllUrl());
    }

    private Map<String, URL> findAllUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.ROUTERS_CATEGORY);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public Route findRoute(String id) {
        return SyncUtils.url2Route(findRouteUrlPair(id));
    }

    public Pair<String, URL> findRouteUrlPair(String id) {
        return SyncUtils.filterFromCategory(getRegistryCache(), Constants.ROUTERS_CATEGORY, id);
    }

    private URL findRouteUrl(String id) {
        return findRoute(id).toUrl();
    }

    private Map<String, URL> findRouteUrl(String service, String address, boolean force) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.ROUTERS_CATEGORY);
        if (service != null && service.length() > 0) {
            filter.put(SyncUtils.SERVICE_FILTER_KEY, service);
        }
        if (address != null && address.length() > 0) {
            filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);
        }
        if (force) {
            filter.put("force", "true");
        }
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<Route> findByService(String serviceName) {
        return SyncUtils.url2RouteList(findRouteUrl(serviceName, null, false));
    }

    public List<Route> findByAddress(String address) {
        return SyncUtils.url2RouteList(findRouteUrl(null, address, false));
    }

    public List<Route> findByServiceAndAddress(String service, String address) {
        return SyncUtils.url2RouteList(findRouteUrl(service, address, false));
    }

    public List<Route> findForceRouteByService(String service) {
        return SyncUtils.url2RouteList(findRouteUrl(service, null, true));
    }

    public List<Route> findForceRouteByAddress(String address) {
        return SyncUtils.url2RouteList(findRouteUrl(null, address, true));
    }

    public List<Route> findForceRouteByServiceAndAddress(String service, String address) {
        return SyncUtils.url2RouteList(findRouteUrl(service, address, true));
    }

    public List<Route> findAllForceRoute() {
        return SyncUtils.url2RouteList(findRouteUrl(null, null, true));
    }

    public Route getBlackwhitelistRouteByService(String service) {
        List<Route> routes = SyncUtils.url2RouteList(findRouteUrl(service, null, true));
        for (Route route : routes) {
            if (route.getName().endsWith(AccessDTO.KEY_BLACK_WHITE_LIST)) {
                return route;
            }
        }
        return null;
    }

}
