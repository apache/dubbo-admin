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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.mapper.MockRuleMapper;
import org.apache.dubbo.admin.model.domain.MockRule;
import org.apache.dubbo.admin.model.dto.GlobalMockRuleDTO;
import org.apache.dubbo.admin.model.dto.MockRuleDTO;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.admin.service.MockRuleService;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.google.gson.Gson;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.mock.api.GlobalMockRule;
import org.apache.dubbo.mock.api.MockConstants;
import org.apache.dubbo.mock.api.MockContext;
import org.apache.dubbo.mock.api.MockResult;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageImpl;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Component;

import java.util.Collections;
import java.util.List;
import java.util.Objects;
import java.util.Optional;
import java.util.stream.Collectors;

/**
 * @author chenglu
 * @date 2021-08-24 15:50
 */
@Component
public class MockRuleServiceImpl implements MockRuleService {

    private static final String CONFIG_KEY = Constants.CONFIG_KEY + Constants.PATH_SEPARATOR +
            MockConstants.ADMIN_MOCK_RULE_KEY;

    @Autowired
    private MockRuleMapper mockRuleMapper;

    @Autowired
    private GovernanceConfiguration configuration;

    @Override
    public void createOrUpdateMockRule(MockRuleDTO mockRule) {
        if (Objects.isNull(mockRule.getServiceName()) || Objects.isNull(mockRule.getMethodName())
                || Objects.isNull(mockRule.getRule())) {
            throw new IllegalStateException("Param serviceName, methodName, rule cannot be null");
        }
        MockRule rule = MockRule.toMockRule(mockRule);
        QueryWrapper<MockRule> queryWrapper = new QueryWrapper<>();
        queryWrapper.eq("service_name", mockRule.getServiceName());
        queryWrapper.eq("method_name", mockRule.getMethodName());
        MockRule existRule = mockRuleMapper.selectOne(queryWrapper);

        // check if we can save or update the rule, we need keep the serviceName + methodName is unique.
        if (Objects.nonNull(existRule)) {
            if (Objects.equals(rule.getServiceName(), existRule.getServiceName())
                    && Objects.equals(rule.getMethodName(), existRule.getMethodName())) {
                if (!Objects.equals(rule.getId(), existRule.getId())) {
                    throw new DuplicateKeyException("Service Name and Method Name must be unique");
                }
            }
        }

        enableOrDisableMockRuleInConfigurationCenter(rule);
        if (Objects.nonNull(rule.getId())) {
            mockRuleMapper.updateById(rule);
            return;
        }
        mockRuleMapper.insert(rule);
    }

    @Override
    public void deleteMockRuleById(Long id) {
        MockRule mockRule = mockRuleMapper.selectById(id);
        if (Objects.isNull(mockRule)) {
            throw new IllegalStateException("Mock Rule cannot find");
        }
        // remove the config in config center
        mockRule.setEnable(false);
        enableOrDisableMockRuleInConfigurationCenter(mockRule);
        mockRuleMapper.deleteById(id);
    }

    @Override
    public Page<MockRuleDTO> listMockRulesByPage(String filter, Pageable pageable) {
        QueryWrapper<MockRule> queryWrapper = new QueryWrapper<>();
        Optional.ofNullable(filter)
                .ifPresent(f -> queryWrapper.like("service_name", f));
        List<MockRule> mockRules = mockRuleMapper.selectList(queryWrapper);
        int total = mockRules.size();
        final List<MockRuleDTO> content = mockRules.stream()
                .skip(pageable.getOffset())
                .limit(pageable.getPageSize())
                .map(MockRuleDTO::toMockRuleDTO)
                .collect(Collectors.toList());
        return new PageImpl<>(content, pageable, total);
    }

    @Override
    public GlobalMockRuleDTO getGlobalMockRule() {
        GlobalMockRuleDTO globalMockRule = new GlobalMockRuleDTO();
        globalMockRule.setEnableMock(false);

        String content = configuration.getConfig(CONFIG_KEY);
        if (StringUtils.isBlank(content)) {
            return globalMockRule;
        }
        GlobalMockRule mockRule = new Gson().fromJson(content, GlobalMockRule.class);
        if (Objects.isNull(mockRule)) {
            return globalMockRule;
        }
        globalMockRule.setEnableMock(mockRule.getEnableMock());
        return globalMockRule;
    }

    @Override
    public void changeGlobalMockRule(GlobalMockRuleDTO globalMockRule) {
        GlobalMockRule mockRule = new GlobalMockRule();
        mockRule.setEnableMock(globalMockRule.getEnableMock());
        String content = configuration.getConfig(CONFIG_KEY);
        if (StringUtils.isNotEmpty(content)) {
            GlobalMockRule existMockRule =
                    new Gson().fromJson(content, GlobalMockRule.class);
            if (Objects.nonNull(existMockRule)) {
                mockRule.setEnabledMockRules(existMockRule.getEnabledMockRules());
            }
        }
        String newContent = new Gson().toJson(mockRule);
        configuration.setConfig(CONFIG_KEY, newContent);
    }

    @Override
    public MockResult getMockData(MockContext mockContext) {
        QueryWrapper<MockRule> queryWrapper = new QueryWrapper<>();
        queryWrapper.eq("service_name", mockContext.getServiceName());
        queryWrapper.eq("method_name", mockContext.getMethodName());
        MockRule mockRule = mockRuleMapper.selectOne(queryWrapper);
        MockResult mockResult = new MockResult();
        mockResult.setEnable(true);
        if (Objects.isNull(mockRule)) {
            return mockResult;
        }
        mockResult.setContent(mockRule.getRule());
        return mockResult;
    }

    private void enableOrDisableMockRuleInConfigurationCenter(MockRule mockRule) {
        String methodName = mockRule.getServiceName() + "#" + mockRule.getMethodName();
        String content = configuration.getConfig(CONFIG_KEY);
        GlobalMockRule rule;
        if (StringUtils.isBlank(content)) {
            rule = new GlobalMockRule();
            rule.setEnableMock(false);
            if (mockRule.getEnable()) {
                rule.setEnabledMockRules(Collections.singleton(methodName));
            }
        } else {
            rule = new Gson().fromJson(content, GlobalMockRule.class);
            Optional.ofNullable(rule.getEnabledMockRules())
                    .ifPresent(rules -> {
                        if (mockRule.getEnable()) {
                            rules.add(methodName);
                        } else {
                            rules.remove(methodName);
                        }
                    });
        }
        configuration.setConfig(CONFIG_KEY, new Gson().toJson(rule));
    }
}
