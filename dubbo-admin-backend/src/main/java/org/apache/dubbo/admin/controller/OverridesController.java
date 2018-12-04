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

import org.apache.commons.lang3.StringUtils;
import org.apache.dubbo.admin.common.exception.ParamValidationException;
import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.model.dto.OverrideDTO;
import org.apache.dubbo.admin.service.OverrideService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;

@RestController
@RequestMapping("/api/{env}/rules/override")
public class OverridesController {

    private final OverrideService overrideService;

    @Autowired
    public OverridesController(OverrideService overrideService) {
        this.overrideService = overrideService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createOverride(@RequestBody OverrideDTO overrideDTO, @PathVariable String env) {
        String serviceName = overrideDTO.getService();
        String application = overrideDTO.getApplication();
        if (StringUtils.isEmpty(serviceName) && StringUtils.isEmpty(application)) {
            throw new ParamValidationException("serviceName and application are Empty!");
        }
        overrideService.saveOverride(overrideDTO);
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateOverride(@PathVariable String id, @RequestBody OverrideDTO overrideDTO, @PathVariable String env) {
        OverrideDTO old = overrideService.findOverride(id);
        if (old == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        overrideService.updateOverride(old, overrideDTO);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<OverrideDTO> searchOverride(@RequestParam(required = false) String serviceName,
                                            @RequestParam(required = false) String application,
                                            @PathVariable String env) {
        OverrideDTO override = null;
        List<OverrideDTO> result = new ArrayList<>();
        if (StringUtils.isNotEmpty(serviceName)) {
            override = overrideService.findOverride(serviceName);
        } else if(StringUtils.isNotEmpty(application)){
            override = overrideService.findOverride(application);
        }
        if (override != null) {
            override = convertOverrideDTOtoDisplay(override);
            result.add(override);
        }
        return result;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public OverrideDTO detailOverride(@PathVariable String id, @PathVariable String env) {
        OverrideDTO override = overrideService.findOverride(id);
        if (override == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }

        override = convertOverrideDTOtoDisplay(override);
        return override;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
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

    private OverrideDTO convertOverrideDTOtoDisplay(OverrideDTO overrideDTO) {
        if (overrideDTO.getScope().equals("application")) {
            overrideDTO.setApplication(overrideDTO.getKey());
        } else {
            overrideDTO.setService(overrideDTO.getKey());
        }
        overrideDTO.setScope(null);
        overrideDTO.setKey(null);
        return overrideDTO;
    }
}
