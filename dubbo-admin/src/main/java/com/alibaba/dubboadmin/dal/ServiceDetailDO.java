package com.alibaba.dubboadmin.dal;

import java.util.List;

import com.alibaba.dubboadmin.registry.common.domain.Consumer;
import com.alibaba.dubboadmin.registry.common.domain.Provider;

/**
 * @author zmx ON 2018/7/20
 */
public class ServiceDetailDO {

    List<Provider> providers;
    List<Consumer> consumers;

    public List<Provider> getProviders() {
        return providers;
    }

    public void setProviders(List<Provider> providers) {
        this.providers = providers;
    }

    public List<Consumer> getConsumers() {
        return consumers;
    }

    public void setConsumers(List<Consumer> consumers) {
        this.consumers = consumers;
    }
}
