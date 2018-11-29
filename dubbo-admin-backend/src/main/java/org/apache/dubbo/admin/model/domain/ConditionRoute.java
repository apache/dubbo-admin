package org.apache.dubbo.admin.model.domain;

public class ConditionRoute extends Route{
    private String scope;
    private String[] conditions;

    public String getScope() {
        return scope;
    }

    public void setScope(String scope) {
        this.scope = scope;
    }

    public String[] getConditions() {
        return conditions;
    }

    public void setConditions(String[] conditions) {
        this.conditions = conditions;
    }
}