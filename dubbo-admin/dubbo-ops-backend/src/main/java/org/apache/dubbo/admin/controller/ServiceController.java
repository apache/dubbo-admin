package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.domain.ServiceDO;
import org.apache.dubbo.admin.domain.ServiceDetailDO;
import com.alibaba.dubbo.common.Constants;
import com.alibaba.dubbo.common.utils.StringUtils;
import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;


@RestController
@RequestMapping("/service")
public class ServiceController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @RequestMapping("/search")
    public List<ServiceDO> search(@RequestParam String filter,
                                  @RequestParam String pattern,
                                  HttpServletRequest request,
                                  HttpServletResponse response, Model model) {

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

        } else if (pattern.equals("ip")) {
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
        model.addAttribute("serviceDO", result);
        return result;
    }

    @RequestMapping("/detail")
    public ServiceDetailDO serviceDetail(@RequestParam String app, @RequestParam String service) {
        List<Provider> providers = providerService.findByAppandService(app, service);

        List<Consumer> consumers = consumerService.findByAppandService(app, service);

        ServiceDetailDO serviceDetailDO = new ServiceDetailDO();
        serviceDetailDO.setConsumers(consumers);
        serviceDetailDO.setProviders(providers);
        return serviceDetailDO;
    }

}
