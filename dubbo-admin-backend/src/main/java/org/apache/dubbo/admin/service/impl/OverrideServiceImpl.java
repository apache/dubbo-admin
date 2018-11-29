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

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.service.OverrideService;
import org.apache.dubbo.admin.model.domain.Override;
import org.springframework.stereotype.Component;
import org.yaml.snakeyaml.Yaml;

import java.util.List;


/**
 * IbatisOverrideDAO.java
 *
 */
@Component
public class OverrideServiceImpl extends AbstractService implements OverrideService {
    private String prefix = Constants.CONFIG_KEY;
    Yaml yaml = new Yaml();

    @java.lang.Override
    public void saveOverride(Override override) {
        String path = getPath(override.getKey());
        dynamicConfiguration.setConfig(path, yaml.dump(override));
    }

    @java.lang.Override
    public void updateOverride(Override override) {
        String path = getPath(override.getKey());
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        dynamicConfiguration.setConfig(path, yaml.dump(override));
    }

    @java.lang.Override
    public void deleteOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            // throw exception
        }
        String path = getPath(id);
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        dynamicConfiguration.deleteConfig(path);
    }

    @java.lang.Override
    public void enableOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            //throw exception
        }
        String path = getPath(id);
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        String config = dynamicConfiguration.getConfig(path);
        Override override = yaml.loadAs(config, Override.class);
        override.setEnabled(true);
        dynamicConfiguration.setConfig(path, yaml.dump(override));
    }

    @java.lang.Override
    public void disableOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            //throw exception
        }
        String path = getPath(id);
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        String config = dynamicConfiguration.getConfig(path);
        Override override = yaml.loadAs(config, Override.class);
        override.setEnabled(false);
        dynamicConfiguration.setConfig(path, yaml.dump(override));
    }

    @java.lang.Override
    public Override findOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            //throw exception
        }
        String path = getPath(id);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return yaml.loadAs(config, Override.class);
        }
        return null;
    }

    private String getPath(String key) {
        return prefix + Constants.PATH_SEPARATOR + key + Constants.PATH_SEPARATOR + "configurators";
    }

}
