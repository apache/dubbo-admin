package org.apache.dubbo.admin.model.domain;

import org.apache.dubbo.admin.model.dto.MockRuleDTO;

import org.springframework.beans.BeanUtils;

/**
 * @author chenglu
 * @date 2021-08-24 17:19
 */
public class MockRule {

    private Long id;

    private String serviceName;

    private String methodName;

    private String rule;

    private Boolean enable;

    public static MockRule toMockRule(MockRuleDTO mockRule) {
        MockRule rule = new MockRule();
        BeanUtils.copyProperties(mockRule, rule);
        return rule;
    }

    public String getServiceName() {
        return serviceName;
    }

    public void setServiceName(String serviceName) {
        this.serviceName = serviceName;
    }

    public String getMethodName() {
        return methodName;
    }

    public void setMethodName(String methodName) {
        this.methodName = methodName;
    }

    public String getRule() {
        return rule;
    }

    public void setRule(String rule) {
        this.rule = rule;
    }

    public Boolean getEnable() {
        return enable;
    }

    public void setEnable(Boolean enable) {
        this.enable = enable;
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }
}
