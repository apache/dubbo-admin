package org.apache.dubbo.admin.data.metadata;

import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.SPI;

@SPI("zookeeper")
public interface MetaDataCollector {

    void setUrl(URL url);

    URL getUrl();

    void init();
    String getMetaData(String path);
}
