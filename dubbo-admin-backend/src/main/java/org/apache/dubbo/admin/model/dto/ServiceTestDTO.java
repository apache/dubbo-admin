package org.apache.dubbo.admin.model.dto;

public class ServiceTestDTO {
    private String service;
    private String method;
    private String[] paramaterTypes;
    private Object[] params;

    public String getService() {
        return service;
    }

    public void setService(String service) {
        this.service = service;
    }

    public String getMethod() {
        return method;
    }

    public void setMethod(String method) {
        this.method = method;
    }

    public String[] getParamaterTypes() {
        return paramaterTypes;
    }

    public void setParamaterTypes(String[] paramaterTypes) {
        this.paramaterTypes = paramaterTypes;
    }

    public Object[] getParams() {
        return params;
    }

    public void setParams(Object[] params) {
        this.params = params;
    }
}
