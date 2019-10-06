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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.logger.Logger;
import org.apache.dubbo.common.logger.LoggerFactory;

import com.alibaba.nacos.api.NacosFactory;
import com.alibaba.nacos.api.config.ConfigService;
import com.alibaba.nacos.api.exception.NacosException;
import org.apache.commons.lang3.StringUtils;

import java.util.Properties;

import static com.alibaba.nacos.api.PropertyKeyConst.SERVER_ADDR;

public class NacosConfiguration implements GovernanceConfiguration {
    private static final Logger logger = LoggerFactory.getLogger(NacosConfiguration.class);

    private ConfigService configService;
    private String group;
    private URL url;

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
    public void setUrl(URL url) {
        this.url = url;
    }

    @Override
    public URL getUrl() {
        return url;
    }

    @Override
    public String setConfig(String key, String value) {
        return setConfig(group, key, value);
    }

    @Override
    public String getConfig(String key) {
        return getConfig(group, key);
    }

    @Override
    public boolean deleteConfig(String key) {
        return deleteConfig(group, key);
    }

    @Override
    public String setConfig(String group, String key, String value) {
        String[] groupAndDataId = parseGroupAndDataId(key, group);
        if (null == groupAndDataId) {
            return null;
        }

        try {
            configService.publishConfig(groupAndDataId[1], groupAndDataId[0], value);
            return value;
        } catch (NacosException e) {
            logger.error(e.getMessage(), e);

        }
        return null;
    }

    @Override
    public String getConfig(String group, String key) {
        String[] groupAndDataId = parseGroupAndDataId(key, group);
        if (null == groupAndDataId) {
            return null;
        }
        try {
            return configService.getConfig(groupAndDataId[1], groupAndDataId[0],1000 * 10);
        } catch (NacosException e) {
            logger.error(e.getMessage(), e);
        }
        return null;
    }

    @Override
    public boolean deleteConfig(String group, String key) {
        String[] groupAndDataId = parseGroupAndDataId(key, group);
        if (null == groupAndDataId) {
            return false;
        }
        try {
           return configService.removeConfig(groupAndDataId[1], groupAndDataId[0]);
        } catch (NacosException e) {
            logger.error(e.getMessage(), e);
        }
        return false;
    }

    @Override
    public String getPath(String key) {
        return null;
    }

    @Override
    public String getPath(String group, String key) {
        return null;
    }

    private String[] parseGroupAndDataId(String key, String group) {
        if (StringUtils.isBlank(key) || StringUtils.isBlank(group)) {
            if (logger.isWarnEnabled()) {
                logger.warn("key or group is blank");
                return null;
            }
        }

        String[] groupAndDataId = new String[2];
        String[] split = key.split("/");
        if (split.length != 3) {
            return null;
        }
        if (Constants.DUBBO_PROPERTY.equals(split[2])) {

            if (this.group.equals(split[1])) {
                groupAndDataId[0] = this.group;
            } else {
                groupAndDataId[0] = split[1];
            }
            groupAndDataId[1] = split[2];
        } else {
            groupAndDataId[0] = group;
            groupAndDataId[1] = split[1] + Constants.PUNCTUATION_POINT + split[2];
        }
        return groupAndDataId;
    }
}
