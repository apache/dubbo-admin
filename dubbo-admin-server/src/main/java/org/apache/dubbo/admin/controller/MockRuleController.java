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

import org.apache.dubbo.admin.annotation.Authority;
import org.apache.dubbo.admin.model.dto.MockRuleDTO;
import org.apache.dubbo.admin.service.MockRuleService;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * Mock Rule Controller.
 */
@Authority(needLogin = true)
@RestController
@RequestMapping("/api/{env}/mock/rule")
public class MockRuleController {

    @Autowired
    private MockRuleService mockRuleService;

    @PostMapping
    public boolean createOrUpdateMockRule(@RequestBody MockRuleDTO mockRule) {
        mockRuleService.createOrUpdateMockRule(mockRule);
        return true;
    }

    @DeleteMapping
    public boolean deleteMockRule(@RequestBody MockRuleDTO mockRule) {
        mockRuleService.deleteMockRuleById(mockRule.getId());
        return true;
    }

    @GetMapping("/list")
    public Page<MockRuleDTO> listMockRules(@RequestParam(required = false) String filter, Pageable pageable) {
        return mockRuleService.listMockRulesByPage(filter, pageable);
    }
}
