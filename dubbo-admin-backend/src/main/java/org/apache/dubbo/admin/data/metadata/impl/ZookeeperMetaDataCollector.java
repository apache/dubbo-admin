package org.apache.dubbo.admin.data.metadata.impl;

import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.remoting.zookeeper.ZookeeperClient;

public class ZookeeperMetaDataCollector implements MetaDataCollector {

    ZookeeperClient client;

    @Override
    public String getMetaData(String path) {
        client.getContent(path);
        return null;
    }
}
