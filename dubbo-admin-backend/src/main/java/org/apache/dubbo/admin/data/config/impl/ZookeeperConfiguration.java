package org.apache.dubbo.admin.data.config.impl;

import org.apache.curator.framework.CuratorFramework;
import org.apache.curator.framework.CuratorFrameworkFactory;
import org.apache.curator.retry.ExponentialBackoffRetry;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.data.config.GovernanceConfiguration;
import org.apache.dubbo.common.URL;

public class ZookeeperConfiguration implements GovernanceConfiguration {
    private CuratorFramework zkClient;
    private URL url;
    private String root;

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
        String group = url.getParameter(Constants.GROUP_KEY, Constants.DEFAULT_ROOT);
        if (!group.startsWith(Constants.PATH_SEPARATOR)) {
            group = Constants.PATH_SEPARATOR + group;
        }
        root = group;
        zkClient.start();
    }

    @Override
    public String setConfig(String key, String value) {
        String path = getNodePath(key);
        try {
            if (zkClient.checkExists().forPath(path) == null) {
                zkClient.create().creatingParentsIfNeeded().forPath(path);
            }
            zkClient.setData().forPath(path, value.getBytes());
            return value;
        } catch (Exception e) {

        }
        return null;
    }

    @Override
    public String getConfig(String key) {
        String path = getNodePath(key);

        try {
            if (zkClient.checkExists().forPath(path) == null) {
                return null;
            }
            return new String(zkClient.getData().forPath(path));
        } catch (Exception e) {
            e.printStackTrace();
        }
        return null;
    }

    @Override
    public boolean deleteConfig(String key) {
        try {
            zkClient.delete().forPath(key);
        } catch (Exception e) {
            e.printStackTrace();
        }
        return true;
    }

    private String getNodePath(String path) {
        return toRootDir() + path;
    }

    private String toRootDir() {
        if (root.equals(Constants.PATH_SEPARATOR)) {
            return root;
        }
        return root + Constants.PATH_SEPARATOR;
    }
}
