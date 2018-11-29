package org.apache.dubbo.admin.model.dto;

public class ConditionRouteDTO extends RouteDTO{
    private String service;
    private String[] conditions;

    public String getService() {
        return service;
    }

    public void setService(String service) {
        this.service = service;
    }

    public String[] getConditions() {
        return conditions;
    }

    public void setConditions(String[] conditions) {
        this.conditions = conditions;
    }
}
