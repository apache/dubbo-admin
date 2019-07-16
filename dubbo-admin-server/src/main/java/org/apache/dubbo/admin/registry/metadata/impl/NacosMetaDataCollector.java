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
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;

import com.alibaba.nacos.api.NacosFactory;
import com.alibaba.nacos.api.config.ConfigService;
import com.alibaba.nacos.api.exception.NacosException;

import java.util.Properties;

import static com.alibaba.nacos.api.PropertyKeyConst.SERVER_ADDR;

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
        properties.put(SERVER_ADDR, serverAddr);
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
            return configService.getConfig(identifier.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY),
                    group, 1000 * 10);
        } catch (NacosException e) {
            logger.warn("Failed to get " + identifier + " from nacos, cause: " + e.getMessage(), e);
        }
        return null;
    }
}
