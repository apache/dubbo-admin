package org.apache.dubbo.admin.model.dto;

/**
 * @author chenglu
 * @date 2021-08-24 15:52
 */
public class MockRuleDTO {

    private Long id;

    private String serviceName;

    private String methodName;

    private String rule;

    private Boolean enable;

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getRule() {
        return rule;
    }

    public void setRule(String rule) {
        this.rule = rule;
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

    public Boolean getEnable() {
        return enable;
    }

    public void setEnable(Boolean enable) {
        this.enable = enable;
    }

    @Override
    public String toString() {
        return "MockRuleDTO{" +
                "id=" + id +
                ", serviceName='" + serviceName + '\'' +
                ", methodName='" + methodName + '\'' +
                ", rule='" + rule + '\'' +
                ", enable=" + enable +
                '}';
    }
}
