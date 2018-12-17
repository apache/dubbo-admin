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
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.model.dto.AccessDTO;
import org.apache.dubbo.admin.model.dto.WeightDTO;
import org.apache.dubbo.admin.service.OverrideService;
import org.apache.dubbo.admin.model.domain.Override;
import org.apache.dubbo.admin.model.domain.Weight;
import org.apache.dubbo.admin.common.util.OverrideUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;

@RestController
@RequestMapping("/api/{env}/rules/weight")
public class WeightController {

    private final OverrideService overrideService;

    @Autowired
    public WeightController(OverrideService overrideService) {
        this.overrideService = overrideService;
    }

    @RequestMapping(method = RequestMethod.POST)
    @ResponseStatus(HttpStatus.CREATED)
    public boolean createWeight(@RequestBody WeightDTO weightDTO, @PathVariable String env) {
        if (StringUtils.isBlank(weightDTO.getService()) && StringUtils.isBlank(weightDTO.getApplication())) {
            throw new ParamValidationException("Either Service or application is required.");
        }
        overrideService.saveWeight(weightDTO);
//        List<String> addresses = weightDTO.getAddresses();
//        for (String address : addresses) {
//            Weight weight = new Weight();
//            weight.setService(weightDTO.getService());
//            weight.setWeight(weight.getWeight());
//            weight.setAddress(address);
//            overrideService.saveOverride(OverrideUtils.weightToOverride(weight));
//        }
        return true;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.PUT)
    public boolean updateWeight(@PathVariable String id, @RequestBody WeightDTO weightDTO, @PathVariable String env) {
        if (id == null) {
            throw new ParamValidationException("Unknown ID!");
        }
        WeightDTO weight = overrideService.findWeight(id);
        if (weight == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
//        Weight old = OverrideUtils.overrideToWeight(override);
//        Weight weight = new Weight();
//        weight.setWeight(weightDTO.getWeight());
//        weight.setHash(id);
//        weight.setService(old.getService());
//        overrideService.updateOverride(OverrideUtils.weightToOverride(weight));
        overrideService.updateWeight(weightDTO);
        return true;
    }

    @RequestMapping(method = RequestMethod.GET)
    public List<WeightDTO> searchWeight(@RequestParam(required = false) String serviceName,
                                        @RequestParam(required = false) String application,
                                        @PathVariable String env) {
        if (StringUtils.isBlank(serviceName) && StringUtils.isBlank(application)) {
            throw new ParamValidationException("Either service or application is required");
        }
        WeightDTO weightDTO;
        if (StringUtils.isNotBlank(application)) {
            weightDTO = overrideService.findWeight(application);
        } else {
            weightDTO = overrideService.findWeight(serviceName);
        }
        List<WeightDTO> weightDTOS = new ArrayList<>();
        if (weightDTO != null) {
            weightDTOS.add(weightDTO);
        }

//        if (StringUtils.isEmpty(service)) {
//            overrides = overrideService.findAll();
//        } else {
//            overrides = overrideService.findByService(service);
//        }
//        List<WeightDTO> weightDTOS = new ArrayList<>();
//        for (Override override : overrides) {
//            Weight w = OverrideUtils.overrideToWeight(override);
//            if (w != null) {
//                WeightDTO weightDTO = new WeightDTO();
//                weightDTO.setAddresses(new String[]{w.getAddress()});
//                weightDTO.setService(w.getService());
//                weightDTO.setWeight(w.getWeight());
//                weightDTO.setId(w.getHash());
//                weightDTOS.add(weightDTO);
//            }
//        }
        return weightDTOS;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public WeightDTO detailWeight(@PathVariable String id, @PathVariable String env) {
        WeightDTO weightDTO = overrideService.findWeight(id);
        if (weightDTO == null) {
            throw new ResourceNotFoundException("Unknown ID!");
        }
        return weightDTO;
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.DELETE)
    public boolean deleteWeight(@PathVariable String id, @PathVariable String env) {
        overrideService.deleteWeight(id);
        return true;
    }
}
