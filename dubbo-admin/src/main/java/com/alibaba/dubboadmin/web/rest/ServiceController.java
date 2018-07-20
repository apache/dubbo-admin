package com.alibaba.dubboadmin.web.rest;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.utils.StringUtils;
import com.alibaba.dubboadmin.dal.ServiceDO;
import com.alibaba.dubboadmin.governance.service.ConsumerService;
import com.alibaba.dubboadmin.governance.service.OverrideService;
import com.alibaba.dubboadmin.governance.service.ProviderService;
import com.alibaba.dubboadmin.registry.common.domain.Provider;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author zmx ON 2018/7/20
 */

@RestController
public class ServiceController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @Autowired
    private OverrideService overrideService;


    @RequestMapping("/search")
    public List<ServiceDO> search(@RequestParam String filter,
                                  @RequestParam String pattern) {
        List<Provider> allProviders = providerService.findAll();

        List<ServiceDO> result = new ArrayList<>();
        if (pattern.equals("app")) {
            for (Provider provider : allProviders) {
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                String app = map.get(Constants.APPLICATION_KEY);
                if (app.toLowerCase().contains(filter)) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(app);
                    s.setServiceName(provider.getService());
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }
            }

        } else if (pattern.equals("service")) {
            for (Provider provider : allProviders) {
                String service = provider.getService();
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                if (service.toLowerCase().contains(filter.toLowerCase())) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(map.get(Constants.APPLICATION_KEY));
                    s.setServiceName(service);
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }
            }

        } else if (pattern.equals("address")) {
            for (Provider provider : allProviders) {
                String address = provider.getAddress();
                Map<String, String> map = StringUtils.parseQueryString(provider.getParameters());
                if (address.contains(filter)) {
                    ServiceDO s = new ServiceDO();
                    s.setAppName(map.get(Constants.APPLICATION_KEY));
                    s.setServiceName(provider.getService());
                    s.setGroup(map.get(Constants.GROUP_KEY));
                    s.setVersion(map.get(Constants.VERSION_KEY));
                    result.add(s);
                }

            }
        }
        return result;
    }
}
