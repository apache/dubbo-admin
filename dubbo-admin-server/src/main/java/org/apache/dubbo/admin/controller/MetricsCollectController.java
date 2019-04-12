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

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.MetricDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.admin.service.impl.MetrcisCollectServiceImpl;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;
import java.util.HashMap;
import java.util.List;
import java.util.ArrayList;
import java.util.Set;
import java.util.Iterator;



@RestController
@RequestMapping("/api/{env}/metrics")
public class MetricsCollectController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @RequestMapping(method = RequestMethod.POST)
    public String metricsCollect(@RequestParam String group) {
        MetrcisCollectServiceImpl service = new MetrcisCollectServiceImpl();
        service.setUrl("dubbo://127.0.0.1:20880?scope=remote&cache=true");

        return service.invoke(group).toString();
    }

    private String getOnePortMessage(String group, String ip, String port, String protocol) {
        MetrcisCollectServiceImpl metrcisCollectService = new MetrcisCollectServiceImpl();
        metrcisCollectService.setUrl(protocol + "://" + ip + ":" + port +"?scope=remote&cache=true");
        String res = metrcisCollectService.invoke(group).toString();
        return res;
    }

    @RequestMapping( value = "/ipAddr", method = RequestMethod.GET)
    public List<MetricDTO> searchService(@RequestParam String ip, @RequestParam String group) {

        System.out.println(ip);
        Map<String, String> configMap = new HashMap<String, String>();
        addMetricsConfigToMap(configMap);

//         default value
        if (configMap.size() <= 0) {
            configMap.put("20880", "dubbo");
        }
        List<MetricDTO> metricDTOS = new ArrayList<>();
        for (String port : configMap.keySet()) {
            String protocol = configMap.get(port);
            String res = getOnePortMessage(group, ip, port, protocol);
            metricDTOS.addAll(new Gson().fromJson(res, new TypeToken<List<MetricDTO>>(){}.getType()));
        }

        return metricDTOS;
    }

    protected void addMetricsConfigToMap(Map<String, String> configMap) {
        Set<String> services = providerService.findServices();
        services.addAll(consumerService.findServices());
        Iterator<String> it = services.iterator();
        while (it.hasNext()) {
            String service = it.next();
            List<Provider>  providers = providerService.findByService(service);
            List<Consumer> consumers = consumerService.findByService(service);
            String providerApplication = null;
            String consumerApplication = null;
            providerApplication = providers.get(0).getApplication();
            consumerApplication = consumers.get(0).getApplication();

            MetadataIdentifier providerMetadataIdentifier = new MetadataIdentifier(service
                    ,null,null, Constants.PROVIDER_SIDE ,providerApplication);
            MetadataIdentifier consumerMetadataIdentifier = new MetadataIdentifier(service
                    ,null,null, Constants.CONSUMER_SIDE ,consumerApplication);

            if (consumerApplication != null) {
                String consumerMetadata = consumerService.getConsumerMetadata(consumerMetadataIdentifier);
                Map<String, String> consumerParameters = new Gson().fromJson(consumerMetadata, Map.class);
                configMap.put(consumerParameters.get(Constants.METRICS_PORT), consumerParameters.get(Constants.METRICS_PROTOCOL));
            }

            if (providerApplication != null) {
                String providerMetaData = providerService.getProviderMetaData(providerMetadataIdentifier);
                FullServiceDefinition providerServiceDefinition = new Gson().fromJson(providerMetaData, FullServiceDefinition.class);
                Map<String, String> parameters = providerServiceDefinition.getParameters();
                configMap.put(parameters.get(Constants.METRICS_PORT), parameters.get(Constants.METRICS_PROTOCOL));
            }
        }
        configMap.remove(null);
    }
}
