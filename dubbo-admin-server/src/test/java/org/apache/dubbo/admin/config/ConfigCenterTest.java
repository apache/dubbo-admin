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

package org.apache.dubbo.admin.config;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNotNull;

import java.io.IOException;

import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.curator.test.TestingServer;
import org.apache.dubbo.admin.configcenter.ConfigCenterConfig;
import org.apache.dubbo.admin.registry.RegistryConfig;
import org.apache.dubbo.admin.registry.config.GovernanceConfigurationConfig;
import org.apache.dubbo.admin.registry.metadata.MetaDataCollector;
import org.apache.dubbo.admin.registry.metadata.MetaDataCollectorConfig;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.NetUtils;
import org.apache.dubbo.configcenter.DynamicConfiguration;
import org.apache.dubbo.registry.Registry;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.InjectMocks;
import org.springframework.test.context.junit4.SpringJUnit4ClassRunner;


@RunWith(SpringJUnit4ClassRunner.class)
public class ConfigCenterTest {

    private String zkAddress;
    private TestingServer zkServer;
    private CuratorFramework zkClient;

    @Before
    public void setup() throws Exception {
        int zkServerPort = NetUtils.getAvailablePort();
        zkAddress = "zookeeper://127.0.0.1:" + zkServerPort;
        zkServer = new TestingServer(zkServerPort, true);
        zkClient = CuratorFrameworkFactory.builder().connectString(zkServer.getConnectString()).retryPolicy(new ExponentialBackoffRetry(1000, 3)).build();
        zkClient.start();
        
    }

    @After
    public void tearDown() throws IOException {
        zkServer.close();
        zkServer.stop();
    }

    @InjectMocks
    private ConfigCenterConfig configCenterConfig;
    @InjectMocks
    private RegistryConfig registryConfig;
    @InjectMocks
    private GovernanceConfigurationConfig governanceConfigurationConfig;
    @InjectMocks
    private MetaDataCollectorConfig metaDataCollectorConfig;
    
    private DynamicConfiguration dynamicConfiguration;

    public void getDynamicConfiguration() throws Exception {
        zkClient.createContainers("/dubbo/config/dubbo/dubbo.properties");
        zkClient.setData().forPath("/dubbo/config/dubbo/dubbo.properties", "dubbo.registry.address=zookeeper://test-registry.com:2182".getBytes());
        // mock @value inject
        configCenterConfig.setAddress(zkAddress);
        configCenterConfig.setGroup("dubbo");
        configCenterConfig.setUsername("username");
        configCenterConfig.setPassword("password");
        dynamicConfiguration = configCenterConfig.getDynamicConfiguration();

        String configfile = configCenterConfig.getConfigFile();
        String group = configCenterConfig.getGroup();
        assertEquals(dynamicConfiguration.getProperties(configfile, group), "dubbo.registry.address=zookeeper://test-registry.com:2182");
    }
    
    @Test
    public void testDynamicConfiguration() throws Exception {
        
        getDynamicConfiguration();
        String configfile = configCenterConfig.getConfigFile();
        dynamicConfiguration.addListener(configfile, event -> {
            Registry registry = registryConfig.configCenterRegistry(dynamicConfiguration);
            URL registryUrl = registry.getUrl();
            assertNotNull(registryUrl);
            assertEquals("127.0.0.1", registryUrl.getHost());
        });
        
        dynamicConfiguration.addListener(configfile, event -> {
            MetaDataCollector metaDataCollector = metaDataCollectorConfig.configCenterMetaDataCollector(dynamicConfiguration);
            URL metadataUrl = metaDataCollector.getUrl();
            assertEquals("127.0.0.1", metadataUrl.getHost());
        });
        zkClient.setData().forPath("/dubbo/config/dubbo/dubbo.properties", getPropertiesContent().getBytes());
        
        Thread.sleep(2000);

        // config is empty
//        zkClient.setData().forPath("/dubbo/config/dubbo/dubbo.properties", "".getBytes());
//        registryConfig.setAddress(null);
//        registry = registryConfig.configCenterRegistry(dynamicConfiguration);
//        assertNotNull(registry.getUrl());
    }
    
    @Test
    public void testStaticConfiguration() {

        registryConfig.setAddress(zkAddress);
        Registry registry = registryConfig.registry();
        URL registryUrl = registry.getUrl();
        assertNotNull(registryUrl);
        assertEquals("127.0.0.1", registryUrl.getHost());
        
        metaDataCollectorConfig.setAddress(zkAddress);
        MetaDataCollector metaDataCollector = metaDataCollectorConfig.metaDataCollector();
        URL metadataUrl = metaDataCollector.getUrl();
        assertEquals("127.0.0.1", metadataUrl.getHost());
        
    }

    
    private String getPropertiesContent() {
        return "dubbo.registry.address=" + zkAddress + "\n" +
        "dubbo.metadata-report.address=" + zkAddress + "\n";
    }
}
