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

import com.alibaba.dubbo.common.URL;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.dto.BaseDTO;
import org.apache.dubbo.admin.dto.OverrideDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.util.StringUtils;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/{env}/rules/override")
public class OverridesController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createOverride(@RequestBody OverrideDTO overrideDTO, @PathVariable String env) {
        String serviceName = overrideDTO.getService();
        if (StringUtils.isEmpty(serviceName)) {
            throw new ParamValidationException("serviceName is Empty!");
        }
        Override override = new Override();
        override.setService(serviceName);
        override.setApplication(overrideDTO.getApp());
        override.setAddress(overrideDTO.getAddress());
        override.setEnabled(overrideDTO.isEnabled());
        overrideDTOToParams(override, overrideDTO);
        overrideService.saveOverride(override);
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateOverride(@PathVariable String id, @RequestBody OverrideDTO overrideDTO, @PathVariable String env) {
        Override old = overrideService.findById(id);
        if (old == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        Override override = new Override();
        override.setService(overrideDTO.getService());
        override.setApplication(overrideDTO.getApp());
        override.setAddress(overrideDTO.getAddress());
        override.setEnabled(overrideDTO.isEnabled());
        overrideDTOToParams(override, overrideDTO);
        override.setHash(id);
        overrideService.updateOverride(override);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<OverrideDTO> searchOverride(@RequestParam(required = false) String service, @PathVariable String env) {
        List<Override> overrides;
        if (service == null || service.length() == 0) {
           overrides = overrideService.findAll();
        }
        overrides = overrideService.findByService(service);
        List<OverrideDTO> result = new ArrayList<>();
        for (Override override : overrides) {
            OverrideDTO overrideDTO = new OverrideDTO();
            overrideDTO.setAddress(override.getAddress());
            overrideDTO.setApp(override.getApplication());
            overrideDTO.setEnabled(override.isEnabled());
            overrideDTO.setService(override.getService());
            overrideDTO.setId(override.getHash());
            paramsToOverrideDTO(override, overrideDTO);
            result.add(overrideDTO);
        }
        return result;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public OverrideDTO detailOverride(@PathVariable String id, @PathVariable String env) {
        Override override = overrideService.findById(id);
        if (override == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        OverrideDTO overrideDTO = new OverrideDTO();
        overrideDTO.setAddress(override.getAddress());
        overrideDTO.setApp(override.getApplication());
        overrideDTO.setEnabled(override.isEnabled());
        overrideDTO.setService(override.getService());
        paramsToOverrideDTO(override, overrideDTO);
        return overrideDTO;
    }

    @RequestMapping(value  = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteOverride(@PathVariable String id, @PathVariable String env) {
        overrideService.deleteOverride(id);
        return true;
    }

    @RequestMapping(value = "/enable/{id}", method = RequestMethod.PUT)
    public boolean enableRoute(@PathVariable String id, @PathVariable String env) {

        overrideService.enableOverride(id);
        return true;
    }

    @RequestMapping(value = "/disable/{id}", method = RequestMethod.PUT)
    public boolean disableRoute(@PathVariable String id, @PathVariable String env) {

        overrideService.disableOverride(id);
        return true;
    }

    private void overrideDTOToParams(Override override, OverrideDTO overrideDTO) {
        Map<Object, String>[] mocks = overrideDTO.getMock();
        Map<String, Object>[] parameters = overrideDTO.getParameters();
        StringBuilder params = new StringBuilder();
        if (mocks != null) {
            for (Map<Object, String> mock : mocks) {
                for (Map.Entry<Object, String> entry : mock.entrySet()) {
                    String key;
                    if (entry.getKey().equals("0")) {
                        key = "mock";
                    } else {
                        key = entry.getKey() + ".mock";
                    }
                    String value = key + "=" + URL.encode(entry.getValue());
                    params.append(value).append("&");
                }
            }
        }

        if (parameters != null) {
            for (Map<String, Object> param : parameters) {
                for (Map.Entry<String, Object> entry : param.entrySet()) {
                    String value = entry.getKey() + "=" + entry.getValue();
                    params.append(value).append("&");
                }
            }
        }
        int length = params.length();
        if (params.charAt(length - 1) == '&') {
            params.deleteCharAt(length - 1);
        }
        override.setParams(params.toString());
    }

    private void paramsToOverrideDTO(Override override, OverrideDTO overrideDTO) {
        String params = override.getParams();
        List<Map<Object, String>> mock = new ArrayList<>();
        List<Map<String, Object>> parameters = new ArrayList<>();
        String[] pair = params.split("&");
        for (String p : pair) {
            String key = p.split("=")[0];
            if (key.contains("mock")) {
                //mock
                String value = URL.decode(p.split("=")[1]);
                Map<Object, String> item = new HashMap<>();
                if (key.contains(".")) {
                    //single method mock
                    key = key.split("\\.")[0];
                    item.put(key, value);
                } else {
                    item.put(0, value);
                }
                mock.add(item);
            } else {
                //parameter
                String value = p.split("=")[1];
                Map<String, Object> item = new HashMap<>();
                item.put(key, value);
                parameters.add(item);
            }
        }
        Map<Object, String>[] mockArray = new Map[mock.size()];
        overrideDTO.setMock(mock.toArray(mockArray));
        Map<String, Object>[] paramArray = new Map[parameters.size()];
        overrideDTO.setParameters(parameters.toArray(paramArray));
    }

}
