package org.apache.dubbo.admin.data.config.impl;

import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;

public class ZookeeperConfiguration implements GovernanceConfiguration {
    private CuratorFramework zkClient;
    private URL url;

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
        zkClient = CuratorFrameworkFactory.newClient(url.getAddress(), new ExponentialBackoffRetry(1000, 3));
        zkClient.start();
    }

    @Override
    public String setConfig(String key, String value) {
        try {
            if (zkClient.checkExists().forPath(key) == null) {
                zkClient.create().creatingParentsIfNeeded().forPath(key);
            }
            zkClient.setData().forPath(key, value.getBytes());
            return value;
        } catch (Exception e) {

        }
        return null;
    }

    @Override
    public String getConfig(String key) {

        try {
            if (zkClient.checkExists().forPath(key) == null) {
                return null;
            }
            return new String(zkClient.getData().forPath(key));
        } catch (Exception e) {
            e.printStackTrace();
        }
        return null;
    }
}
