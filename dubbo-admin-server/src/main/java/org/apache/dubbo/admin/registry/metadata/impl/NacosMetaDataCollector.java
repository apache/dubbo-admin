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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.registry.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.SPI;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier.KeyTypeEnum;

import com.alibaba.nacos.api.NacosFactory;
import com.alibaba.nacos.api.config.ConfigService;
import com.alibaba.nacos.api.exception.NacosException;

import java.util.Properties;

import static com.alibaba.nacos.api.PropertyKeyConst.SERVER_ADDR;
import static com.alibaba.nacos.api.PropertyKeyConst.NAMESPACE;

@SPI("nacos")
public class NacosMetaDataCollector implements MetaDataCollector {
    private static final Logger logger = LoggerFactory.getLogger(NacosMetaDataCollector.class);
    private ConfigService configService;
    private String group;
    private URL url;
    @Override
    public void setUrl(URL url) {
        this.url = url;
    }

    @Override
    public URL getUrl() {
        return url;
    }

    @Override
    public void init() {
        group = url.getParameter(Constants.GROUP_KEY, "DEFAULT_GROUP");

        configService = buildConfigService(url);
    }

    private ConfigService buildConfigService(URL url) {
        Properties nacosProperties = buildNacosProperties(url);
        try {
            configService = NacosFactory.createConfigService(nacosProperties);
        } catch (NacosException e) {
            if (logger.isErrorEnabled()) {
                logger.error(e.getErrMsg(), e);
            }
            throw new IllegalStateException(e);
        }
        return configService;
    }

    private Properties buildNacosProperties(URL url) {
        Properties properties = new Properties();
        setServerAddr(url, properties);
        return properties;
    }

    private void setServerAddr(URL url, Properties properties) {

        String serverAddr = url.getHost() + // Host
                ":" +
                url.getPort() // Port
                ;
        String namespace = url.getParameter(NAMESPACE);
        properties.put(SERVER_ADDR, serverAddr);
        properties.put(NAMESPACE, namespace);
    }

    @Override
    public String getProviderMetaData(MetadataIdentifier key) {
        return getMetaData(key);
    }

    @Override
    public String getConsumerMetaData(MetadataIdentifier key) {
        return getMetaData(key);
    }

    private String getMetaData(MetadataIdentifier identifier) {
        try {
            return configService.getConfig(getUniqueKey(identifier, MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY),
                    group, 1000 * 10);
        } catch (NacosException e) {
            logger.warn("Failed to get " + identifier + " from nacos, cause: " + e.getMessage(), e);
        }
        return null;
    }

    private String getUniqueKey(MetadataIdentifier identifier, KeyTypeEnum keyType) {
    	String serviceInterface = identifier.getServiceInterface();
    	String SEPARATOR = identifier.SEPARATOR;
    	String version = identifier.getVersion();
    	String group = identifier.getGroup();
    	String side = identifier.getSide();
    	String application = identifier.getApplication();
        return serviceInterface + SEPARATOR + (version == null ? "" : version + SEPARATOR) + (group == null ? "" : group + SEPARATOR) + side + SEPARATOR + application;
    }
}
