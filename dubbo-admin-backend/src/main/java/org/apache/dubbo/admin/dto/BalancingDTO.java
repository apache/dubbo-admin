package org.apache.dubbo.admin.dto;

/**
 * @author zmx ON 2018/9/25
 */
public class BalancingDTO {

    private Long id;
    private String serviceName;
    private String rule;

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getServiceName() {
        return serviceName;
    }

    public void setServiceName(String serviceName) {
        this.serviceName = serviceName;
    }

    public String getRule() {
        return rule;
    }

    public void setRule(String rule) {
        this.rule = rule;
    }
}
