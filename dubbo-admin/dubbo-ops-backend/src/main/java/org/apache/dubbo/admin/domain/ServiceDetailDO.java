package org.apache.dubbo.admin.domain;

import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;

import java.util.List;

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
