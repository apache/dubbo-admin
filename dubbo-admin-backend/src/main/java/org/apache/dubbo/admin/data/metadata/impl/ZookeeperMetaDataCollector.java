package org.apache.dubbo.admin.data.metadata.impl;

import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.metadata.identifier.ConsumerMetadataIdentifier;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.apache.dubbo.metadata.identifier.ProviderMetadataIdentifier;

public class ZookeeperMetaDataCollector implements MetaDataCollector {

    private CuratorFramework client;
    private URL url;
    private String root;
    private final static String METADATA_NODE_NAME = "service.data";
    private final static String DEFAULT_ROOT = "dubbo";

    @Override
    public void setUrl(URL url) {
        this.url = url;
    }

    @Override
    public URL getUrl() {
        return url;
    }

    @Override
    public void init() {
        String group = url.getParameter(Constants.GROUP_KEY, DEFAULT_ROOT);
        if (!group.startsWith(Constants.PATH_SEPARATOR)) {
            group = Constants.PATH_SEPARATOR + group;
        }
        root = group;
        client = CuratorFrameworkFactory.newClient(url.getAddress(), new ExponentialBackoffRetry(1000, 3));
        client.start();
    }


    @Override
    public String getProviderMetaData(ProviderMetadataIdentifier key) {
        return doGetMetadata(key);
    }

    @Override
    public String getConsumerMetaData(ConsumerMetadataIdentifier key) {
        return doGetMetadata(key);
    }

    private String getNodePath(MetadataIdentifier metadataIdentifier) {
        return toRootDir() + metadataIdentifier.getUniqueKey(MetadataIdentifier.KeyTypeEnum.PATH) +
                Constants.PATH_SEPARATOR + METADATA_NODE_NAME;
    }

    private String toRootDir() {
        if (root.equals(Constants.PATH_SEPARATOR)) {
            return root;
        }
        return root + Constants.PATH_SEPARATOR;
    }

    private String doGetMetadata(MetadataIdentifier identifier) {
        //TODO error handing
        try {
            String path = getNodePath(identifier);
            if (client.checkExists().forPath(path) == null) {
                return null;
            }
            return new String(client.getData().forPath(path));
        } catch (Exception e) {
            e.printStackTrace();
        }
        return null;
    }
}
