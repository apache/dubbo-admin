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

import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.RelationDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.MetricsService;
import org.apache.dubbo.admin.service.ProviderService;

import org.apache.dubbo.common.utils.CollectionUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@Component
public class MetricsServiceImpl implements MetricsService {

    @Autowired
    private ConsumerService consumerService;
    @Autowired
    private ProviderService providerService;

    @Override
    public RelationDTO getApplicationRelation() {

        List<Consumer> consumerList = consumerService.findAll();
        List<Provider> providerList = providerService.findAll();

        int index = 0;
        // collect all service
        Set<String> serviceSet = new HashSet<>();

        // collect consumer's nodes map <application, node>
        Map<String, RelationDTO.Node> consumerNodeMap = new HashMap<>();
        // collect consumer's service and applications map <service, set<application>>
        Map<String, Set<String>> consumerServiceApplicationMap = new HashMap<>();
        for (Consumer consumer : consumerList) {
            String application = consumer.getApplication();
            if (!consumerNodeMap.keySet().contains(application)) {
                RelationDTO.Node node = new RelationDTO.Node(index, application, RelationDTO.CONSUMER_CATEGORIES.getIndex());
                consumerNodeMap.put(application, node);
                index++;
            }
            String service = consumer.getService();
            serviceSet.add(service);
            consumerServiceApplicationMap.computeIfAbsent(service, s -> new HashSet<>());
            consumerServiceApplicationMap.get(service).add(application);
        }
        // collect provider's nodes
        Map<String, RelationDTO.Node> providerNodeMap = new HashMap<>();
        // collect provider's service and applications map <service, set<application>>
        Map<String, Set<String>> providerServiceApplicationMap = new HashMap<>();
        for (Provider provider : providerList) {
            String application = provider.getApplication();
            if (!providerNodeMap.keySet().contains(application)) {
                RelationDTO.Node node = new RelationDTO.Node(index, application, RelationDTO.PROVIDER_CATEGORIES.getIndex());
                providerNodeMap.put(application, node);
                index++;
            }
            String service = provider.getService();
            serviceSet.add(service);
            providerServiceApplicationMap.computeIfAbsent(service, s -> new HashSet<>());
            providerServiceApplicationMap.get(service).add(application);
        }
        // merge provider's nodes and consumer's nodes
        Map<String, RelationDTO.Node> nodeMap = new HashMap<>(consumerNodeMap);
        for (Map.Entry<String, RelationDTO.Node> entry : providerNodeMap.entrySet()) {
            if (nodeMap.get(entry.getKey()) != null) {
                nodeMap.get(entry.getKey()).setCategory(RelationDTO.CONSUMER_AND_PROVIDER_CATEGORIES.getIndex());
            } else {
                nodeMap.put(entry.getKey(), entry.getValue());
            }
        }
        // build link by same service
        Set<RelationDTO.Link> linkSet = new HashSet<>();
        for (String service : serviceSet) {
            Set<String> consumerApplicationSet = consumerServiceApplicationMap.get(service);
            Set<String> providerApplicationSet = providerServiceApplicationMap.get(service);
            if (CollectionUtils.isNotEmpty(consumerApplicationSet) && CollectionUtils.isNotEmpty(providerApplicationSet)) {
                for (String providerApplication : providerApplicationSet) {
                    for (String consumerApplication : consumerApplicationSet) {
                        if (nodeMap.get(consumerApplication) != null && nodeMap.get(providerApplication) != null) {
                            Integer consumerIndex = nodeMap.get(consumerApplication).getIndex();
                            Integer providerIndex = nodeMap.get(providerApplication).getIndex();
                            linkSet.add(new RelationDTO.Link(consumerIndex, providerIndex));
                        }
                    }
                }
            }
        }
        // sort node by index
        List<RelationDTO.Node> nodeList = nodeMap.values().stream().sorted(Comparator.comparingInt(RelationDTO.Node::getIndex)).collect(Collectors.toList());
        return new RelationDTO(nodeList, new ArrayList<>(linkSet));
    }
}
