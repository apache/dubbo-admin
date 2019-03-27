package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.monitor.MetricsService;

public class MetrcisCollectServiceImpl {

    private ReferenceConfig<MetricsService> referenceConfig;

    public MetrcisCollectServiceImpl() {
        referenceConfig = new ReferenceConfig<>();
        referenceConfig.setApplication(new ApplicationConfig("dubbo-admin"));
        referenceConfig.setInterface(MetricsService.class);
    }

    public void setUrl(String url) {
        referenceConfig.setUrl(url);
    }

    public Object invoke(String group) {
        MetricsService metricsService = referenceConfig.get();
        return metricsService.getMetricsByGroup(group);
    }
}
