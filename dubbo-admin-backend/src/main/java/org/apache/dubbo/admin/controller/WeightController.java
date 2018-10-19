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
import org.apache.dubbo.admin.dto.WeightDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.apache.dubbo.admin.registry.common.domain.Weight;
import org.apache.dubbo.admin.registry.common.util.OverrideUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;

@RestController
@RequestMapping("/api/{env}/rules/weight")
public class WeightController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createWeight(@RequestBody WeightDTO weightDTO, @PathVariable String env) {
        String[] addresses = weightDTO.getProvider();
        for (String address : addresses) {
            Weight weight = new Weight();
            weight.setService(weightDTO.getService());
            weight.setWeight(weight.getWeight());
            weight.setAddress(address);
            overrideService.saveOverride(OverrideUtils.weightToOverride(weight));
        }
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateWeight(@PathVariable String id, @RequestBody WeightDTO weightDTO, @PathVariable String env) {
        if (id == null) {
            throw new ParamValidationException("Unknown ID!");
        }
        Override override = overrideService.findById(id);
        if (override == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        Weight old = OverrideUtils.overrideToWeight(override);
        Weight weight = new Weight();
        weight.setWeight(weightDTO.getWeight());
        weight.setHash(id);
        weight.setService(old.getService());
        overrideService.updateOverride(OverrideUtils.weightToOverride(weight));
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<WeightDTO> searchWeight(@RequestParam(required = false) String service, @PathVariable String env) {
        List<Override> overrides;
        if (StringUtils.isEmpty(service)) {
            overrides = overrideService.findAll();
        } else {
            overrides = overrideService.findByService(service);
        }
        List<WeightDTO> weightDTOS = new ArrayList<>();
        for (Override override : overrides) {
            Weight w = OverrideUtils.overrideToWeight(override);
            if (w != null) {
                WeightDTO weightDTO = new WeightDTO();
                weightDTO.setProvider(new String[]{w.getAddress()});
                weightDTO.setService(w.getService());
                weightDTO.setWeight(w.getWeight());
                weightDTO.setId(w.getHash());
                weightDTOS.add(weightDTO);
            }
        }
        return weightDTOS;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public WeightDTO detailWeight(@PathVariable String id, @PathVariable String env) {
        Override override = overrideService.findById(id);
        if (override != null) {

            Weight w = OverrideUtils.overrideToWeight(override);
            WeightDTO weightDTO = new WeightDTO();
            weightDTO.setProvider(new String[]{w.getAddress()});
            weightDTO.setService(w.getService());
            weightDTO.setWeight(w.getWeight());
            weightDTO.setId(w.getHash());
            return weightDTO;
        }
        return null;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteWeight(@PathVariable String id, @PathVariable String env) {
        overrideService.deleteOverride(id);
        return true;
    }
}
