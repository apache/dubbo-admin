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
import org.apache.dubbo.admin.common.util.YamlParser;
import org.apache.dubbo.admin.model.domain.Override;
import org.apache.dubbo.admin.model.domain.OverrideConfig;
import org.apache.dubbo.admin.model.dto.OverrideDTO;
import org.apache.dubbo.admin.service.OverrideService;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;


/**
 * IbatisOverrideDAO.java
 *
 */
@Component
public class OverrideServiceImpl extends AbstractService implements OverrideService {
    private String prefix = Constants.CONFIG_KEY;

    @java.lang.Override
    public void saveOverride(OverrideDTO override) {
        override = convertOverrideDTOtoStore(override);
        String path = getPath(override.getKey());
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(override));

        //for2.6
        if (override.getScope().equals("service")) {
            List<Override> result = convertDTOtoOldOverride(override);
            for (Override o : result) {
                registry.register(o.toUrl());
            }
        }
    }

    @java.lang.Override
    public void updateOverride(OverrideDTO old, OverrideDTO update) {
        old = convertOverrideDTOtoStore(old);
        update = convertOverrideDTOtoStore(update);
        String path = getPath(update.getKey());
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(update));

        //for 2.6
        if (update.getScope().equals("service")) {
            List<Override> oldOverrides = convertDTOtoOldOverride(old);
            List<Override> updatedOverrides = convertDTOtoOldOverride(update);
            for (Override o : oldOverrides) {
                registry.unregister(o.toUrl());
            }
            for (Override o : updatedOverrides) {
                registry.register(o.toUrl());
            }
        }
    }

    @java.lang.Override
    public void deleteOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            // throw exception
        }
        String path = getPath(id);
        String config = dynamicConfiguration.getConfig(path);
        if (config == null) {
            //throw exception
        }
        dynamicConfiguration.deleteConfig(path);

        //for 2.6
        OverrideDTO overrideDTO = YamlParser.loadObject(config, OverrideDTO.class);
        if (overrideDTO.getScope().equals("service")) {
            List<Override> overrides = convertDTOtoOldOverride(overrideDTO);
            for (Override o : overrides) {
                registry.unregister(o.toUrl());
            }
        }
    }

    @java.lang.Override
    public void enableOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            //throw exception
        }
        String path = getPath(id);
        String config = dynamicConfiguration.getConfig(path);
        if (config == null) {
            //throw exception
        }
        OverrideDTO override = YamlParser.loadObject(config, OverrideDTO.class);
        override.setEnabled(true);
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(override));

        //2.6
        if (override.getScope().equals("service")) {
            List<Override> overrides = convertDTOtoOldOverride(override);
            for (Override o : overrides) {
                o.setEnabled(false);
                registry.unregister(o.toUrl());
                o.setEnabled(true);
                registry.register(o.toUrl());
            }
        }
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
        OverrideDTO override = YamlParser.loadObject(config, OverrideDTO.class);
        override.setEnabled(false);
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(override));

        //for 2.6
        if (override.getScope().equals("service")) {
            List<Override> overrides = convertDTOtoOldOverride(override);
            for (Override o : overrides) {
                o.setEnabled(true);
                registry.unregister(o.toUrl());
                o.setEnabled(false);
                registry.register(o.toUrl());
            }
        }
    }

    @java.lang.Override
    public OverrideDTO findOverride(String id) {
        if (StringUtils.isEmpty(id)) {
            //throw exception
        }
        String path = getPath(id);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return YamlParser.loadObject(config, OverrideDTO.class);
        }
        return null;
    }

    private OverrideDTO convertOverrideDTOtoStore(OverrideDTO overrideDTO) {
        if (StringUtils.isNotEmpty(overrideDTO.getApplication())) {
           overrideDTO.setScope("application");
           overrideDTO.setKey(overrideDTO.getApplication());
        } else {
            overrideDTO.setScope("service");
            overrideDTO.setKey(overrideDTO.getService());
        }
        overrideDTO.setApplication(null);
        overrideDTO.setService(null);
        return overrideDTO;
    }

    private void overrideDTOToParams(Override override, OverrideConfig config) {
        Map<String, Object> parameters = config.getParameters();
        StringBuilder params = new StringBuilder();

        if (parameters != null) {
            for (Map.Entry<String, Object> entry : parameters.entrySet()) {
                String value = entry.getKey() + "=" + entry.getValue();
                params.append(value).append("&");
            }
        }
        if (StringUtils.isNotEmpty(params)) {
            int length = params.length();
            if (params.charAt(length - 1) == '&') {
                params.deleteCharAt(length - 1);
            }
        }
        override.setParams(params.toString());
    }
    private List<Override> convertDTOtoOldOverride(OverrideDTO overrideDTO) {
        List<Override> result = new ArrayList<>();
        OverrideConfig[] configs = overrideDTO.getConfigs();
        for (OverrideConfig config : configs) {
            String[] apps = config.getApplications();
            String[] addresses = config.getAddresses();
            for (String address : addresses) {
                if (apps != null && apps.length > 0) {
                    for (String app : apps) {
                        Override override = new Override();
                        override.setService(overrideDTO.getKey());
                        override.setAddress(address);
                        override.setEnabled(overrideDTO.isEnabled());
                        overrideDTOToParams(override, config);
                        override.setApplication(app);
                        result.add(override);
                    }
                } else {
                    Override override = new Override();
                    override.setService(overrideDTO.getKey());
                    override.setAddress(address);
                    override.setEnabled(overrideDTO.isEnabled());
                    overrideDTOToParams(override, config);
                    result.add(override);
                }
            }
        }
        return result;
    }
    private String getPath(String key) {
        return prefix + Constants.PATH_SEPARATOR + key + Constants.PATH_SEPARATOR + "configurators";
    }

}
