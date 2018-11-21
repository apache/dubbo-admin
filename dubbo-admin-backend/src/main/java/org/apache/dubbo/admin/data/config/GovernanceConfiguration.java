package org.apache.dubbo.admin.data.config;

import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.SPI;


@SPI("zookeeper")
public interface GovernanceConfiguration {
    void init();

    void setUrl(URL url);

    URL getUrl();
    String setConfig(String key, String value);

    String getConfig(String key);

}
