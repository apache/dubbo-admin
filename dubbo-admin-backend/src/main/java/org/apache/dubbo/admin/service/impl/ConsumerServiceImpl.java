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

import org.apache.dubbo.admin.common.util.Pair;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.common.Constants;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentMap;

@Component
public class ConsumerServiceImpl extends AbstractService implements ConsumerService {

    public List<Consumer> findByService(String service) {
        return SyncUtils.url2ConsumerList(findConsumerUrlByService(service));
    }

    public Consumer findConsumer(String id) {
        return SyncUtils.url2Consumer(findConsumerUrl(id));
    }

    private Pair<String, URL> findConsumerUrl(String id) {
        return SyncUtils.filterFromCategory(getRegistryCache(), Constants.CONSUMERS_CATEGORY, id);
    }

    public List<Consumer> findAll() {
        return SyncUtils.url2ConsumerList(findAllConsumerUrl());
    }

    @Override
    public String getConsumerMetadata(MetadataIdentifier consumerIdentifier) {
        return metaDataCollector.getConsumerMetaData(consumerIdentifier);
    }

    private Map<String, URL> findAllConsumerUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<String> findAddresses() {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (null == consumerUrls) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : consumerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getAddress();
                if (app != null) ret.add(app);
            }
        }

        return ret;
    }

    public List<String> findAddressesByApplication(String application) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);

        if(consumerUrls == null)
            return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : consumerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION_KEY))) {
                    String addr = u.getAddress();
                    if (addr != null) ret.add(addr);
                }
            }
        }

        return ret;
    }

    public List<String> findAddressesByService(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (null == consumerUrls) return ret;

        for (Map.Entry<String, URL> e2 : consumerUrls.get(service).entrySet()) {
            URL u = e2.getValue();
            String app = u.getAddress();
            if (app != null) ret.add(app);
        }

        return ret;
    }

    public List<Consumer> findByAddress(String consumerAddress) {
        return SyncUtils.url2ConsumerList(findConsumerUrlByAddress(consumerAddress));
    }

    public List<String> findServicesByAddress(String address) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (consumerUrls == null || address == null || address.length() == 0) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : consumerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (address.equals(u.getAddress())) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

    private Map<String, URL> findConsumerUrlByAddress(String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<String> findApplications() {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (consumerUrls == null) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : consumerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                String app = u.getParameter(Constants.APPLICATION_KEY);
                if (app != null) ret.add(app);
            }
        }

        return ret;
    }

    public List<String> findApplicationsByServiceName(String service) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (consumerUrls == null) return ret;

        Map<String, URL> value = consumerUrls.get(service);
        if (value == null) {
            return ret;
        }
        for (Map.Entry<String, URL> e2 : value.entrySet()) {
            URL u = e2.getValue();
            String app = u.getParameter(Constants.APPLICATION_KEY);
            if (app != null) ret.add(app);
        }

        return ret;
    }

    public List<Consumer> findByApplication(String application) {
        return SyncUtils.url2ConsumerList(findConsumerUrlByApplication(application));
    }

    public List<Consumer> findByAppandService(String app, String serviceName) {
        return SyncUtils.url2ConsumerList(findConsumerUrlByAppandService(app, serviceName));
    }

    private Map<String, URL> findConsumerUrlByAppandService(String application, String service) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(Constants.APPLICATION_KEY, application);
        filter.put(Constants.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    private Map<String, URL> findConsumerUrlByApplication(String application) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(Constants.APPLICATION_KEY, application);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

    public List<String> findServicesByApplication(String application) {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (consumerUrls == null || application == null || application.length() == 0) return ret;

        for (Map.Entry<String, Map<String, URL>> e1 : consumerUrls.entrySet()) {
            Map<String, URL> value = e1.getValue();
            for (Map.Entry<String, URL> e2 : value.entrySet()) {
                URL u = e2.getValue();
                if (application.equals(u.getParameter(Constants.APPLICATION_KEY))) {
                    ret.add(e1.getKey());
                    break;
                }
            }
        }

        return ret;
    }

    public List<String> findServices() {
        List<String> ret = new ArrayList<String>();
        ConcurrentMap<String, Map<String, URL>> consumerUrls = getRegistryCache().get(Constants.CONSUMERS_CATEGORY);
        if (consumerUrls != null) ret.addAll(consumerUrls.keySet());
        return ret;
    }

    public Map<String, URL> findConsumerUrlByService(String service) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getRegistryCache(), filter);
    }

}
