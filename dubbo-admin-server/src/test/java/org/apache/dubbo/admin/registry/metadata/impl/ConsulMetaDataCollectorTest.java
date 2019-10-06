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

package org.apache.dubbo.admin.registry.metadata.impl;


import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import com.pszymczyk.consul.ConsulProcess;
import com.pszymczyk.consul.ConsulStarterBuilder;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.metadata.definition.ServiceDefinitionBuilder;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.definition.model.MethodDefinition;
import org.apache.dubbo.metadata.definition.util.ClassUtils;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.junit.After;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;

import java.io.IOException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import static org.apache.dubbo.common.constants.CommonConstants.CONSUMER_SIDE;
import static org.apache.dubbo.common.constants.CommonConstants.PROVIDER_SIDE;

public class ConsulMetaDataCollectorTest {

    private final Gson gson = new Gson();
    private ConsulProcess consul;
    private ConsulMetaDataCollector consulMetaDataCollector;

    @Before
    public void setUp() {
        consul = ConsulStarterBuilder.consulStarter().build().start();
        consulMetaDataCollector = new ConsulMetaDataCollector();
        consulMetaDataCollector.setUrl(URL.valueOf("consul://127.0.0.1:" + consul.getHttpPort()));
        consulMetaDataCollector.init();
    }

    @After
    public void tearDown() {
        consul.close();
        consulMetaDataCollector = null;
    }

    @Test
    public void testGetProviderMetaData() {
        MetadataIdentifier identifier = buildIdentifier(true);

        Map<String, String> params = new HashMap<>();
        params.put("key1", "value1");
        params.put("key2", "true");

        FullServiceDefinition definition = ServiceDefinitionBuilder.buildFullDefinition(ServiceA.class, params);

        String metadata = gson.toJson(definition);
        consulMetaDataCollector.getClient().setKVValue(identifier.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY), metadata);

        String providerMetaData = consulMetaDataCollector.getProviderMetaData(identifier);
        Assert.assertEquals(metadata, providerMetaData);

        FullServiceDefinition retDef = gson.fromJson(providerMetaData, FullServiceDefinition.class);
        Assert.assertEquals(ServiceA.class.getCanonicalName(), retDef.getCanonicalName());
        Assert.assertEquals(ClassUtils.getCodeSource(ServiceA.class), retDef.getCodeSource());
        Assert.assertEquals(params, retDef.getParameters());

        //method def assertions
        Assert.assertNotNull(retDef.getMethods());
        Assert.assertEquals(3, retDef.getMethods().size());
        List<String> methodNames = retDef.getMethods().stream().map(MethodDefinition::getName).sorted().collect(Collectors.toList());
        Assert.assertEquals("method1", methodNames.get(0));
        Assert.assertEquals("method2", methodNames.get(1));
        Assert.assertEquals("method3", methodNames.get(2));
    }


    @Test
    public void testGetConsumerMetaData() {
        MetadataIdentifier identifier = buildIdentifier(false);

        Map<String, String> consumerParams = new HashMap<>();
        consumerParams.put("k1", "v1");
        consumerParams.put("k2", "1");
        consumerParams.put("k3", "true");
        String metadata = gson.toJson(consumerParams);
        consulMetaDataCollector.getClient().setKVValue(identifier.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY), metadata);

        String consumerMetaData = consulMetaDataCollector.getConsumerMetaData(identifier);
        Map<String, String> retParams = gson.fromJson(consumerMetaData, new TypeToken<Map<String, String>>() {
        }.getType());
        Assert.assertEquals(consumerParams, retParams);
    }


    private MetadataIdentifier buildIdentifier(boolean isProducer) {
        MetadataIdentifier identifier = new MetadataIdentifier();
        identifier.setApplication(String.format("metadata-%s-test", isProducer ? "provider" : "consumer"));
        identifier.setGroup("group");
        identifier.setServiceInterface(ServiceA.class.getName());
        identifier.setSide(isProducer ? PROVIDER_SIDE : CONSUMER_SIDE);
        identifier.setVersion("1.0.0");
        return identifier;
    }


    interface ServiceA {
        void method1();

        void method2() throws IOException;

        String method3();
    }

}
