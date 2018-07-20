package com.alibaba.dubboadmin.web.rest;

import java.util.List;

import com.alibaba.dubboadmin.dal.ServiceDetailDO;
import com.alibaba.dubboadmin.governance.service.ConsumerService;
import com.alibaba.dubboadmin.governance.service.OverrideService;
import com.alibaba.dubboadmin.governance.service.ProviderService;
import com.alibaba.dubboadmin.registry.common.domain.Consumer;
import com.alibaba.dubboadmin.registry.common.domain.Provider;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author zmx ON 2018/7/20
 */

@RestController
public class ServiceDetailController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @Autowired
    private OverrideService overrideService;

    @RequestMapping("/serviceDetail")
    public ServiceDetailDO serviceDetail(@RequestParam String app, @RequestParam String service) {
        List<Provider> providers = providerService.findByAppandService(app, service);

        List<Consumer> consumers = consumerService.findByAppandService(app, service);

        ServiceDetailDO serviceDetailDO = new ServiceDetailDO();
        serviceDetailDO.setConsumers(consumers);
        serviceDetailDO.setProviders(providers);
        return serviceDetailDO;
    }

    @RequestMapping("/metaData")
    public List<String> metaData(@RequestParam String app, @RequestParam String service) {
        return null;
    }

}
