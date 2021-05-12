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

package org.apache.dubbo.admin.service;

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.common.URL;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentMap;

public interface SideService {

    List<String> findServicesByApplication(String application);

    Map<String, URL> findUrlByAppendService(String application, String serviceName);

    ConcurrentMap<String, ConcurrentMap<String, Map<String, URL>>> getRegistryCache();

    default Map<String, URL> findUrlByAppendService(String application, String serviceName, String category) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, category);
        filter.put(Constants.APPLICATION, application);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, serviceName);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    default String findServiceVersion(String serviceName, String application) {
        String version = null;
        Map<String, URL> result = findUrlByAppendService(application, serviceName);
        if (result != null && result.size() > 0) {
            URL url = result.values().stream().findFirst().get();
            if (url.getParameter(Constants.SPECIFICATION_VERSION_KEY) != null) {
                version = url.getParameter(Constants.SPECIFICATION_VERSION_KEY);
            }
        }
        return version;
    }

    default String findVersionInApplication(String application) {
        List<String> services = findServicesByApplication(application);
        if (services == null || services.size() == 0) {
            throw new ParamValidationException("there is no service for application: " + application);
        }
        return findServiceVersion(services.get(0), application);
    }

    default List<String> findServicesByApplication(String application, String category) {
        List<String> ret = new ArrayList<String>();

        ConcurrentMap<String, Map<String, URL>> urls = getRegistryCache().get(category);
        if (urls == null || application == null || application.length() == 0) {
            return ret;
        }

        for (Map.Entry<String, Map<String, URL>> e1 : urls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION))) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

}
