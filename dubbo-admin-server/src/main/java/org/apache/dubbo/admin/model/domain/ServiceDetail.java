package org.apache.dubbo.admin.model.domain;

import java.util.Date;

public class ServiceDetail {
    private String service;
    private String result;
    private String statistics;
    private Date collected;
    private Date expired;

    public String getService() {
        return service;
    }

    public void setService(String service) {
        this.service = service;
    }

    public String getResult() {
        return result;
    }

    public void setResult(String result) {
        this.result = result;
    }

    public String getStatistics() {
        return statistics;
    }

    public void setStatistics(String statistics) {
        this.statistics = statistics;
    }

    public Date getCollected() {
        return collected;
    }

    public void setCollected(Date collected) {
        this.collected = collected;
    }

    public Date getExpired() {
        return expired;
    }


    public void setExpired(Date expired) {
        this.expired = expired;
    }
}
