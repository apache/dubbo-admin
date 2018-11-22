package org.apache.dubbo.admin.data.metadata.impl;


import org.apache.dubbo.admin.data.metadata.MetaDataCollector;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.metadata.identifier.ConsumerMetadataIdentifier;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.apache.dubbo.metadata.identifier.ProviderMetadataIdentifier;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;

public class RedisMetaDataCollector implements MetaDataCollector {

    private  URL url;
    private JedisPool pool;
    private static final String META_DATA_SOTRE_TAG = ".metaData";
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
    public String getProviderMetaData(ProviderMetadataIdentifier key) {
        return doGetMetaData(key);
    }

    @Override
    public String getConsumerMetaData(ConsumerMetadataIdentifier key) {
        return doGetMetaData(key);
    }

    private String doGetMetaData(MetadataIdentifier identifier) {
        //TODO error handing
        Jedis jedis = pool.getResource();
        String result = jedis.get(identifier.getUniqueKey(MetadataIdentifier.KeyTypeEnum.UNIQUE_KEY) + META_DATA_SOTRE_TAG);
        return result;
    }
}
