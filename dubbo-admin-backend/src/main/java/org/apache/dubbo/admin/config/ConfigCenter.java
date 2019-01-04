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

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.exception.ConfigurationException;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.admin.data.metadata.impl.NoOpMetadataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.ExtensionLoader;
import org.apache.dubbo.registry.Registry;
import org.apache.dubbo.registry.RegistryFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.DependsOn;

import java.util.Arrays;


@Configuration
public class ConfigCenter {


    @Value("${dubbo.configcenter:}")
    private String configCenter;

    @Value("${dubbo.registry.address:}")
    private String registryAddress;

    private static String globalConfigPath = "config/dubbo/dubbo.properties";

    @Value("${dubbo.registry.group:}")
    private String group;


    private URL configCenterUrl;
    private URL registryUrl;
    private URL metadataUrl;


    /*
     * generate dynamic configuration client
     */
    @Bean("governanceConfiguration")
    GovernanceConfiguration getDynamicConfiguration() {
        GovernanceConfiguration dynamicConfiguration = null;

        if (StringUtils.isNotEmpty(configCenter)) {
            configCenterUrl = formUrl(configCenter, group);
            dynamicConfiguration = ExtensionLoader.getExtensionLoader(GovernanceConfiguration.class).getExtension(configCenterUrl.getProtocol());
            dynamicConfiguration.setUrl(configCenterUrl);
            dynamicConfiguration.init();
            String config = dynamicConfiguration.getConfig(globalConfigPath);

            if (StringUtils.isNotEmpty(config)) {
                Arrays.stream(config.split("\n")).forEach( s -> {
                    if(s.startsWith(Constants.REGISTRY_ADDRESS)) {
                        registryUrl = formUrl(s.split("=")[1].trim(), group);
                    } else if (s.startsWith(Constants.METADATA_ADDRESS)) {
                        metadataUrl = formUrl(s.split("=")[1].trim(), group);
                    }
                });
            }
        }
        if (dynamicConfiguration == null) {
            if (StringUtils.isNotEmpty(registryAddress)) {
                registryUrl = formUrl(registryAddress, group);
                dynamicConfiguration = ExtensionLoader.getExtensionLoader(GovernanceConfiguration.class).getExtension(registryUrl.getProtocol());
                dynamicConfiguration.setUrl(registryUrl);
                dynamicConfiguration.init();
            } else {
                throw new ConfigurationException("Either configcenter or registry address is needed");
                //throw exception
            }
        }
        return dynamicConfiguration;
    }

    /*
     * generate registry client
     */
    @Bean
    @DependsOn("governanceConfiguration")
    Registry getRegistry() {
        Registry registry = null;
        if (registryUrl == null) {
            if (StringUtils.isBlank(registryAddress)) {
                throw new ConfigurationException("Either configcenter or registry address is needed");
            }
            registryUrl = formUrl(registryAddress, group);
        }
        RegistryFactory registryFactory = ExtensionLoader.getExtensionLoader(RegistryFactory.class).getAdaptiveExtension();
        registry = registryFactory.getRegistry(registryUrl);
        return registry;
    }

    /*
     * generate metadata client
     */
    @Bean
    @DependsOn("governanceConfiguration")
    MetaDataCollector getMetadataCollector() {
        MetaDataCollector metaDataCollector = new NoOpMetadataCollector();
        if (metadataUrl != null) {
            metaDataCollector = ExtensionLoader.getExtensionLoader(MetaDataCollector.class).getExtension(metadataUrl.getProtocol());
            metaDataCollector.setUrl(metadataUrl);
            metaDataCollector.init();
        }
        return metaDataCollector;
    }

    private URL formUrl(String config, String group) {
        URL url = URL.valueOf(config);
        if (StringUtils.isNotEmpty(group)) {
            url.addParameter(org.apache.dubbo.common.Constants.GROUP_KEY, group);
        }
        return url;
    }
}
