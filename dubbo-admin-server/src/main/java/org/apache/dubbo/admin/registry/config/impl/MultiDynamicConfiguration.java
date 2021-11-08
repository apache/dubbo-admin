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
import org.apache.dubbo.common.config.configcenter.DynamicConfiguration;
import org.apache.dubbo.common.config.configcenter.DynamicConfigurationFactory;
import org.apache.dubbo.common.extension.ExtensionLoader;

/**
 * Use {@link org.apache.dubbo.common.config.configcenter.DynamicConfiguration} adaptation Configuration Center
 */
public class MultiDynamicConfiguration implements GovernanceConfiguration {

    private URL url;

    private DynamicConfiguration dynamicConfiguration;

    private String group;

    @Override
    public void init() {
        if (url == null) {
            throw new IllegalStateException("server url is null, cannot init");
        }
        DynamicConfigurationFactory dynamicConfigurationFactory = ExtensionLoader.getExtensionLoader(DynamicConfigurationFactory.class)
                .getOrDefaultExtension(url.getProtocol());
        dynamicConfiguration = dynamicConfigurationFactory.getDynamicConfiguration(url);
        // group must be consistent with dubbo
        group = url.getParameter(Constants.GROUP_KEY, Constants.DEFAULT_GROUP);
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
    public boolean setConfig(String key, String value) {
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
    public boolean setConfig(String group, String key, String value) {
        if (key == null || value == null) {
            throw new IllegalArgumentException("key or value cannot be null");
        }
        return dynamicConfiguration.publishConfig(key, group, value);
    }

    @Override
    public String getConfig(String group, String key) {
        if (key == null) {
            throw new IllegalArgumentException("key cannot be null");
        }
        return dynamicConfiguration.getConfig(key, group);
    }

    @Override
    public boolean deleteConfig(String group, String key) {
        if (key == null) {
            throw new IllegalArgumentException("key cannot be null");
        }
        return dynamicConfiguration.removeConfig(key, group);
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
