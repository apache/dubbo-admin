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

package org.apache.dubbo.admin.registry.mapping;

import org.apache.dubbo.common.ProtocolServiceKey;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.registry.client.ServiceDiscovery;
import org.apache.dubbo.registry.client.event.listener.ServiceInstancesChangedListener;

import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

public class AdminServiceInstancesChangedListener extends ServiceInstancesChangedListener {

    private AddressChangeListener addressChangeListener;

    private Map<ProtocolServiceKey, List<ServiceInstancesChangedListener.ProtocolServiceKeyWithUrls>> oldServiceUrls;

    public AdminServiceInstancesChangedListener(Set<String> serviceNames, ServiceDiscovery serviceDiscovery, AddressChangeListener addressChangeListener) {
        super(serviceNames, serviceDiscovery);
        this.addressChangeListener = addressChangeListener;
        oldServiceUrls = new HashMap<>();
    }

    protected void notifyAddressChanged() {
        Map<ProtocolServiceKey, List<ProtocolServiceKeyWithUrls>> protocolServiceUrls = serviceUrls.values().stream()
                .flatMap(Collection::stream)
                .collect(Collectors.groupingBy(ProtocolServiceKeyWithUrls::getProtocolServiceKey));

        oldServiceUrls.keySet().stream()
                .filter(protocolServiceKey -> !protocolServiceUrls.containsKey(protocolServiceKey))
                .forEach(protocolServiceKey -> addressChangeListener.notifyAddressChanged(protocolServiceKey.toString(), new ArrayList<>()));

        protocolServiceUrls
                .forEach((protocolServiceKey, urls) -> addressChangeListener.notifyAddressChanged(protocolServiceKey.toString(), extractUrls(urls)));

        oldServiceUrls = protocolServiceUrls;
    }

    private List<URL> extractUrls(List<ServiceInstancesChangedListener.ProtocolServiceKeyWithUrls> keyUrls) {
        return keyUrls.stream()
                .flatMap((protocolServiceKeyWithUrls) -> protocolServiceKeyWithUrls.getUrls().stream())
                .collect(Collectors.toList());
    }
}
