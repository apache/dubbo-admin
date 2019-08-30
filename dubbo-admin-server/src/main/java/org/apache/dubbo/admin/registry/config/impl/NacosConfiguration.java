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

package org.apache.dubbo.admin.registry.config.impl;

import static com.alibaba.nacos.api.PropertyKeyConst.NAMESPACE;
import static com.alibaba.nacos.api.PropertyKeyConst.SERVER_ADDR;

import java.util.Properties;

import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;
import org.hibernate.internal.util.config.ConfigurationException;

import com.alibaba.nacos.api.NacosFactory;
import com.alibaba.nacos.api.config.ConfigService;
import com.alibaba.nacos.api.exception.NacosException;

public class NacosConfiguration implements GovernanceConfiguration {
    private static final Logger logger = LoggerFactory.getLogger(NacosConfiguration.class);
    private ConfigService configService;
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
        if (url == null) {
            throw new IllegalStateException("server url is null, cannot init");
        }
        
        Properties properties = new Properties();
		properties.put(SERVER_ADDR, url.getAddress());
		properties.put(NAMESPACE, url.getParameter(NAMESPACE, "public"));
		try {
			configService = NacosFactory.createConfigService(properties);
		} catch (NacosException e) {
			throw new ConfigurationException("nacos config start error", e);
		}
    }

    @Override
    public String setConfig(String key, String value) {
        return setConfig(null, key, value);
    }

    @Override
    public String getConfig(String key) {
        return getConfig(null, key);
    }

    @Override
    public boolean deleteConfig(String key) {
        return deleteConfig(null, key);
    }

    @Override
    public String setConfig(String group, String key, String value) {
        if (key == null || value == null) {
            throw new IllegalArgumentException("key or value cannot be null");
        }
        try {
        	configService.publishConfig(key, group, value);
            return value;
        } catch (Exception e) {
            logger.error(e.getMessage(), e);
        }
        return null;
    }

    @Override
    public String getConfig(String group, String key) {
        if (key == null) {
            throw new IllegalArgumentException("key cannot be null");
        }
        try {
            return configService.getConfig(key, group, 5000);
        } catch (Exception e) {
            logger.error(e.getMessage(), e);
        }
        return null;
    }

    @Override
    public boolean deleteConfig(String group, String key) {
        if (key == null) {
            throw new IllegalArgumentException("key cannot be null");
        }
        try {
        	configService.removeConfig(key, group);
        } catch (Exception e) {
            logger.error(e.getMessage(), e);
            return false;
        }
        return true;
    }

    @Override
    public String getPath(String key) {
        return null;
    }

    @Override
    public String getPath(String group, String key) {
        return null;
    }


}
