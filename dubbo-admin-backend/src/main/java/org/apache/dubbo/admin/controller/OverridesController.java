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

import org.apache.dubbo.admin.dto.OverrideDTO;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/override")
public class OverridesController {

    @Autowired
    private OverrideService overrideService;

    @RequestMapping(value = "/create", method = RequestMethod.POST)
    public boolean createOverride(@RequestBody OverrideDTO overrideDTO) {
        String serviceName = overrideDTO.getService();
        if (serviceName == null || serviceName.length() == 0) {
            //TODO throw exception
        }
//        String[] mock =

        return false;
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST)
    public boolean updateOverride(@RequestBody OverrideDTO overrideDTO) {
        return false;
    }

    @RequestMapping(value = "/search", method = RequestMethod.POST)
    public List<OverrideDTO> allOverride(@RequestBody Map<String, String> params) {
        return null;
    }

    @RequestMapping("/detail")
    public OverrideDTO detail(@RequestParam Long id) {
       return null;
    }

    @RequestMapping(value  = "/delete", method = RequestMethod.POST)
    public boolean delete(@RequestBody Map<String, Long> params) {
       return false;
    }

}
