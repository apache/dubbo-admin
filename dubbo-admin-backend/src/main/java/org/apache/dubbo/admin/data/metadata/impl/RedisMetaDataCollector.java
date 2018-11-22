package org.apache.dubbo.admin.data.metadata.impl;


import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;

public class RedisMetaDataCollector implements MetaDataCollector {

    private  URL url;
    private JedisPool pool;
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
        pool = new JedisPool(new JedisPoolConfig(), url.getHost(), url.getPort());
    }

    @Override
    public String getMetaData(String path) {
        Jedis jedis = pool.getResource();
        return jedis.get(path);
    }
}
