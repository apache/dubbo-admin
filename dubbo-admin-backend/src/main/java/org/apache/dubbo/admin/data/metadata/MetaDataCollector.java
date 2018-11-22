package org.apache.dubbo.admin.data.metadata;

import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.extension.SPI;
import org.apache.dubbo.metadata.identifier.ConsumerMetadataIdentifier;
import org.apache.dubbo.metadata.identifier.ProviderMetadataIdentifier;

@SPI("zookeeper")
public interface MetaDataCollector {

    void setUrl(URL url);

    URL getUrl();

    void init();

    String getProviderMetaData(ProviderMetadataIdentifier key);

    String getConsumerMetaData(ConsumerMetadataIdentifier key);
}
