package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.admin.mapper.MockRuleMapper;
import org.apache.dubbo.admin.model.domain.MockRule;
import org.apache.dubbo.admin.model.dto.GlobalMockRuleDTO;
import org.apache.dubbo.admin.model.dto.MockRuleDTO;
import org.apache.dubbo.admin.registry.config.GovernanceConfiguration;
import org.apache.dubbo.admin.service.MockRuleService;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.google.gson.Gson;
import org.apache.dubbo.common.utils.StringUtils;
import org.apache.dubbo.mock.api.MockConstants;
import org.apache.dubbo.mock.api.MockResult;
import org.springframework.beans.factory.annotation.Autowired;
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

    @Autowired
    private MockRuleMapper mockRuleMapper;

    @Autowired
    private GovernanceConfiguration configuration;

    @Override
    public void createOrUpdateMockRule(MockRuleDTO mockRule) {
        MockRule rule = MockRule.toMockRule(mockRule);
        enableOrDisableMockRuleInConfigurationCenter(mockRule);
        if (Objects.nonNull(rule.getId())) {
            mockRuleMapper.updateById(rule);
            return;
        }
        mockRuleMapper.insert(rule);
    }

    @Override
    public void deleteMockRuleById(Long id) {
        mockRuleMapper.deleteById(id);
    }

    @Override
    public void updateMockRule(MockRuleDTO mockRule) {
        MockRule rule = MockRule.toMockRule(mockRule);
        mockRuleMapper.updateById(rule);
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

        String content = configuration.getConfig(MockConstants.ADMIN_MOCK_RULE_GROUP, MockConstants.ADMIN_MOCK_RULE_KEY);
        if (StringUtils.isBlank(content)) {
            return globalMockRule;
        }
        org.apache.dubbo.mock.api.MockRule mockRule = new Gson().fromJson(content, org.apache.dubbo.mock.api.MockRule.class);
        if (Objects.isNull(mockRule)) {
            return globalMockRule;
        }
        globalMockRule.setEnableMock(mockRule.isEnableMock());
        return globalMockRule;
    }

    @Override
    public void changeGlobalMockRule(GlobalMockRuleDTO globalMockRule) {
        org.apache.dubbo.mock.api.MockRule mockRule = new org.apache.dubbo.mock.api.MockRule();
        mockRule.setEnableMock(globalMockRule.getEnableMock());
        String content = configuration.getConfig(MockConstants.ADMIN_MOCK_RULE_GROUP, MockConstants.ADMIN_MOCK_RULE_KEY);
        if (StringUtils.isNotEmpty(content)) {
            org.apache.dubbo.mock.api.MockRule existMockRule =
                    new Gson().fromJson(content, org.apache.dubbo.mock.api.MockRule.class);
            if (Objects.nonNull(existMockRule)) {
                mockRule.setEnabledMockRules(existMockRule.getEnabledMockRules());
            }
        }
        String newContent = new Gson().toJson(mockRule);
        configuration.setConfig(MockConstants.ADMIN_MOCK_RULE_GROUP,  MockConstants.ADMIN_MOCK_RULE_KEY, newContent);
    }

    @Override
    public MockResult getMockData(String interfaceName, String methodName, Object[] arguments) {
        QueryWrapper<MockRule> queryWrapper = new QueryWrapper<>();
        queryWrapper.eq("service_name", interfaceName);
        queryWrapper.eq("method_name", methodName);
        MockRule mockRule = mockRuleMapper.selectOne(queryWrapper);
        MockResult mockResult = new MockResult();
        mockResult.setEnable(true);
        if (Objects.isNull(mockRule)) {
            return mockResult;
        }
        mockResult.setContent(mockRule.getRule());
        return mockResult;
    }

    private void enableOrDisableMockRuleInConfigurationCenter(MockRuleDTO mockRule) {
        String methodName = mockRule.getServiceName() + "#" + mockRule.getMethodName();
        String content = configuration.getConfig(MockConstants.ADMIN_MOCK_RULE_GROUP,  MockConstants.ADMIN_MOCK_RULE_KEY);
        org.apache.dubbo.mock.api.MockRule rule;
        if (StringUtils.isBlank(content)) {
            rule = new org.apache.dubbo.mock.api.MockRule();
            rule.setEnableMock(false);
            if (mockRule.getEnable()) {
                rule.setEnabledMockRules(Collections.singleton(methodName));
            }
        } else {
            rule = new Gson().fromJson(content, org.apache.dubbo.mock.api.MockRule.class);
            Optional.ofNullable(rule.getEnabledMockRules())
                    .ifPresent(rules -> {
                        if (mockRule.getEnable()) {
                            rules.add(methodName);
                        } else {
                            rules.remove(methodName);
                        }
                    });
        }
        configuration.setConfig(MockConstants.ADMIN_MOCK_RULE_GROUP,  MockConstants.ADMIN_MOCK_RULE_KEY, new Gson().toJson(rule));
    }
}
