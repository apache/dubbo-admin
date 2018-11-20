package org.apache.dubbo.admin.data.config;

import org.apache.dubbo.configcenter.DynamicConfiguration;


public interface GovernanceConfiguration extends DynamicConfiguration {
    String setConfig(String key, String value);

    String setConfig(String key, String group, String value);

    String setConfig(String key, String group, int timeout, String value);

}
