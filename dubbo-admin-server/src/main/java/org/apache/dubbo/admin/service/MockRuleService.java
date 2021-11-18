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

package org.apache.dubbo.admin.service;

import org.apache.dubbo.admin.model.dto.MockRuleDTO;

import org.apache.dubbo.mock.api.MockContext;
import org.apache.dubbo.mock.api.MockResult;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;

/**
 * The {@link MockRuleService} mainly works on the function of response the request of mock consumer
 * and maintain the mock rule data.
 */
public interface MockRuleService {

    /**
     * create or update mock rule. if the request contains id, then will be an update operation.
     *
     * @param mockRule mock rule.
     */
    void createOrUpdateMockRule(MockRuleDTO mockRule);

    /**
     * delete the mock rule data by mock rule id.
     *
     * @param id mock rule id.
     */
    void deleteMockRuleById(Long id);

    /**
     * list the mock rules by filter and return data by page.
     *
     * @param filter filter condition.
     * @param pageable pageable params.
     * @return mock rules by page.
     */
    Page<MockRuleDTO> listMockRulesByPage(String filter, Pageable pageable);

    /**
     * return the mock rule data by {@link MockContext}.
     *
     * @param mockContext mock context provide by consumer.
     * @return mock data.
     */
    MockResult getMockData(MockContext mockContext);
}
