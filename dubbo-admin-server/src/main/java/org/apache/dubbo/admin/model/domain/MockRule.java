package org.apache.dubbo.admin.model.domain;

import org.apache.dubbo.admin.model.dto.MockRuleDTO;

/**
 * @author chenglu
 * @date 2021-08-24 17:19
 */
public class MockRule {

    private Long id;

    private String serviceName;

    private String methodName;

    private String rule;

    public static MockRule toMockRule(MockRuleDTO mockRule) {
        MockRule rule = new MockRule();
        rule.setServiceName(mockRule.getServiceName().trim());
        rule.setMethodName(mockRule.getMethodName().trim());
        rule.setId(mockRule.getId());
        rule.setRule(mockRule.getRule());
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

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    @java.lang.Override
    public String toString() {
        return "MockRule{" +
                "id=" + id +
                ", serviceName='" + serviceName + '\'' +
                ", methodName='" + methodName + '\'' +
                ", rule='" + rule + '\'' +
                '}';
    }
}
