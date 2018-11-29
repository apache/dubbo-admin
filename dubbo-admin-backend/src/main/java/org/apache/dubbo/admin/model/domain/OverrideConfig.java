package org.apache.dubbo.admin.model.domain;

import java.util.Map;

public class OverrideConfig {
    private String side;
    private String[] addresses;
    private String[] providerAddresses;
    private Map<String, Object> parameters;
    private String[] applications;
    private String[] services;

    public String getSide() {
        return side;
    }

    public void setSide(String side) {
        this.side = side;
    }

    public String[] getAddress() {
        return addresses;
    }

    public void setAddress(String[] address) {
        this.addresses = address;
    }

    public String[] getProviderAddresses() {
        return providerAddresses;
    }

    public void setProviderAddresses(String[] providerAddresses) {
        this.providerAddresses = providerAddresses;
    }

    public Map<String, Object> getParameters() {
        return parameters;
    }

    public void setParameters(Map<String, Object> parameters) {
        this.parameters = parameters;
    }

    public String[] getApplications() {
        return applications;
    }

    public void setApplications(String[] applications) {
        this.applications = applications;
    }

    public String[] getServices() {
        return services;
    }

    public void setServices(String[] services) {
        this.services = services;
    }
}
