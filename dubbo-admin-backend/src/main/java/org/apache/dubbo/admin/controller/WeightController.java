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

import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.registry.common.domain.Override;
import org.apache.dubbo.admin.registry.common.domain.Weight;
import org.apache.dubbo.admin.registry.common.util.OverrideUtils;
import org.apache.dubbo.admin.dto.WeightDTO;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/weight")
public class WeightController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createWeight(@RequestBody WeightDTO weightDTO) {
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

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public boolean updateWeight(@RequestBody WeightDTO weightDTO) {
        Long id = weightDTO.getId();
        if (id == null) {
            //TODO throw exception
        }
        Override override = overrideService.findById(id);
        if (override == null) {
            //TODO throw exception
        }
        Weight old = OverrideUtils.overrideToWeight(override);
        old.setWeight(weightDTO.getWeight());
        overrideService.updateOverride(OverrideUtils.weightToOverride(old));
        return true;
    }

    @RequestMapping(value = "/search", method = RequestMethod.POST)
    public List<Weight> allWeight(@RequestBody Map<String, String> params) {
        String serviceName = params.get("serviceName");
        List<Override> overrides = overrideService.findByService(serviceName);
        List<Weight> weights = new ArrayList<>();
        for (Override override : overrides) {
            Weight w = OverrideUtils.overrideToWeight(override);
            if (w != null) {
                weights.add(w);
            }
        }
        return weights;
    }

    @RequestMapping("/detail")
    public Weight detail(@RequestParam Long id) {
        Override override = overrideService.findById(id);
        if (override != null) {
            return OverrideUtils.overrideToWeight(override);
        }
        return null;
    }

    @RequestMapping(value = "/delete", method = RequestMethod.POST)
    public boolean delete(@RequestBody Map<String, Long> params) {
        Long id = params.get("id");
        overrideService.deleteOverride(id);
        return true;
    }
}
