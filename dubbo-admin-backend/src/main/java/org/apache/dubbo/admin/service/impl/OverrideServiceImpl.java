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
import org.apache.dubbo.admin.service.OverrideService;
import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Override;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * IbatisOverrideDAO.java
 *
 */
@Component
public class OverrideServiceImpl extends AbstractService implements OverrideService {

    public void saveOverride(Override override) {
        URL url = getUrlFromOverride(override);
        registry.register(url);
    }

    public void updateOverride(Override override) {
        String hash = override.getHash();
        if (hash == null) {
            throw new IllegalStateException("no override id");
        }
        URL oldOverride = findOverrideUrl(hash);
        if (oldOverride == null) {
            throw new IllegalStateException("Route was changed!");
        }
        URL newOverride = getUrlFromOverride(override);

        registry.unregister(oldOverride);
        registry.register(newOverride);

    }

    public void deleteOverride(String id) {
        URL oldOverride = findOverrideUrl(id);
        if (oldOverride == null) {
            throw new IllegalStateException("Route was changed!");
        }
        registry.unregister(oldOverride);
    }

    public void enableOverride(String id) {
        if (id == null) {
            throw new IllegalStateException("no override id");
        }

        URL oldOverride = findOverrideUrl(id);
        if (oldOverride == null) {
            throw new IllegalStateException("Override was changed!");
        }
        if (oldOverride.getParameter("enabled", true)) {
            return;
        }

        URL newOverride = oldOverride.addParameter("enabled", true);
        registry.unregister(oldOverride);
        registry.register(newOverride);

    }

    public void disableOverride(String id) {
        if (id == null) {
            throw new IllegalStateException("no override id");
        }

        URL oldProvider = findOverrideUrl(id);
        if (oldProvider == null) {
            throw new IllegalStateException("Override was changed!");
        }
        if (!oldProvider.getParameter("enabled", true)) {
            return;
        }

        URL newProvider = oldProvider.addParameter("enabled", false);
        registry.unregister(oldProvider);
        registry.register(newProvider);

    }

    private Map<String, URL> findOverrideUrl(String service, String address, String application) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONFIGURATORS_CATEGORY);
        if (service != null && service.length() > 0) {
            filter.put(SyncUtils.SERVICE_FILTER_KEY, service);
        }
        if (address != null && address.length() > 0) {
            filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);
        }
        if (application != null && application.length() > 0) {
            filter.put(Constants.APPLICATION_KEY, application);
        }
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<Override> findByAddress(String address) {
        return SyncUtils.url2OverrideList(findOverrideUrl(null, address, null));
    }

    public List<Override> findByServiceAndAddress(String service, String address) {
        return SyncUtils.url2OverrideList(findOverrideUrl(service, address, null));
    }

    public List<Override> findByApplication(String application) {
        return SyncUtils.url2OverrideList(findOverrideUrl(null, null, application));
    }

    public List<Override> findByService(String service) {
        return SyncUtils.url2OverrideList(findOverrideUrl(service, null, null));
    }

    public List<Override> findByServiceAndApplication(String service, String application) {
        return SyncUtils.url2OverrideList(findOverrideUrl(service, null, application));
    }

    public List<Override> findAll() {
        return SyncUtils.url2OverrideList(findOverrideUrl(null, null, null));
    }

    private Pair<String, URL> findOverrideUrlPair(String id) {
        return SyncUtils.filterFromCategory(getRegistryCache(), Constants.CONFIGURATORS_CATEGORY, id);
    }

    public Override findById(String id) {
        return SyncUtils.url2Override(findOverrideUrlPair(id));
    }

    private URL getUrlFromOverride(Override override) {
        return override.toUrl();
    }

    URL findOverrideUrl(String id) {
        return getUrlFromOverride(findById(id));
    }

}
