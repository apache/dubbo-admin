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

package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.model.dto.ConfigDTO;
import org.apache.dubbo.admin.service.ManagementService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

@RestController
@RequestMapping("/api/{env}/manage")
public class ManagementController {

    private final ManagementService managementService;
    private static Pattern CLASS_NAME_PATTERN = Pattern.compile("([a-zA-Z_$][a-zA-Z\\d_$]*\\.)*[a-zA-Z_$][a-zA-Z\\d_$]*");


    @Autowired
    public ManagementController(ManagementService managementService) {
        this.managementService = managementService;
    }

    @RequestMapping(value ="/config", method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createConfig(@RequestBody ConfigDTO config, @PathVariable String env) {
        managementService.setConfig(config);
        return true;
    }

    @RequestMapping(value = "/config/{key}", method = RequestMethod.PUT)
    public boolean updateConfig(@PathVariable String key, @RequestBody ConfigDTO configDTO, @PathVariable String env) {
        if (key == null) {
            throw new ParamValidationException("Unknown ID!");
        }
        String exitConfig = managementService.getConfig(key);
        if (exitConfig == null) {
            throw new ResourceNotFoundException("Unknown ID!"); }
        return managementService.updateConfig(configDTO);
    }

    @RequestMapping(value = "/config/{key}", method = RequestMethod.GET)
    public List<ConfigDTO> getConfig(@PathVariable String key, @PathVariable String env) {
        String config = managementService.getConfig(key);
        List<ConfigDTO> configDTOs = new ArrayList<>();
        if (config == null) {
            return configDTOs;
        }
        ConfigDTO configDTO = new ConfigDTO();
        configDTO.setKey(key);
        configDTO.setConfig(config);
        configDTO.setPath(managementService.getConfigPath(key));
        if (Constants.GLOBAL_CONFIG.equals(key)) {
            configDTO.setScope(Constants.GLOBAL_CONFIG);
        } else if(CLASS_NAME_PATTERN.matcher(key).matches()){
            configDTO.setScope(Constants.SERVICE);
        } else {
            configDTO.setScope(Constants.APPLICATION);
        }
        configDTOs.add(configDTO);
        return configDTOs;
    }

    @RequestMapping(value = "/config/{key}", method = RequestMethod.DELETE)
    public boolean deleteConfig(@PathVariable String key, @PathVariable String env) {
        return managementService.deleteConfig(key);
    }
}
