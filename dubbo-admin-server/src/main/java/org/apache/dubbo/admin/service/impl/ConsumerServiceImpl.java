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

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.SyncUtils;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.metadata.report.identifier.MetadataIdentifier;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Component
public class ConsumerServiceImpl extends AbstractService implements ConsumerService {

    @Autowired
    private InstanceRegistryQueryHelper instanceRegistryQueryHelper;

    @Override
    public List<Consumer> findByService(String service) {
        List<Consumer> consumers = SyncUtils.url2ConsumerList(findConsumerUrlByService(service));
        consumers.addAll(instanceRegistryQueryHelper.findConsumerByService(service));
        return consumers;
    }


    @Override
    public List<Consumer> findAll() {
        List<Consumer> consumers = SyncUtils.url2ConsumerList(findAllConsumerUrl());
        consumers.addAll(instanceRegistryQueryHelper.findAllConsumer());
        return consumers;
    }

    @Override
    public String getConsumerMetadata(MetadataIdentifier consumerIdentifier) {
        return metaDataCollector.getConsumerMetaData(consumerIdentifier);
    }

    private Map<String, URL> findAllConsumerUrl() {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }




    @Override
    public List<Consumer> findByAddress(String consumerAddress) {
        return SyncUtils.url2ConsumerList(findConsumerUrlByAddress(consumerAddress));
    }


    private Map<String, URL> findConsumerUrlByAddress(String address) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(SyncUtils.ADDRESS_FILTER_KEY, address);

        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    public Map<String, URL> findConsumerUrlByService(String service) {
        Map<String, String> filter = new HashMap<String, String>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(SyncUtils.SERVICE_FILTER_KEY, service);

        return SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
    }

    @Override
    public String findVersionInApplication(String application) {
        Map<String, String> filter = new HashMap<>();
        filter.put(Constants.CATEGORY_KEY, Constants.CONSUMERS_CATEGORY);
        filter.put(Constants.APPLICATION_KEY, application);
        Map<String, URL> stringURLMap = SyncUtils.filterFromCategory(getInterfaceRegistryCache(), filter);
        if (stringURLMap == null || stringURLMap.isEmpty()) {
            throw new ParamValidationException("there is no consumer for application: " + application);
        }
        String defaultVersion = "2.6";
        URL url = stringURLMap.values().iterator().next();
        String version = url.getParameter(Constants.SPECIFICATION_VERSION_KEY);
        return StringUtils.isBlank(version) ? defaultVersion : version;
    }
}
