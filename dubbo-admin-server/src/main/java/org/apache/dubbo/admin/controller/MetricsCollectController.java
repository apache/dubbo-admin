package org.apache.dubbo.admin.controller;

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.model.domain.Consumer;
import org.apache.dubbo.admin.model.domain.Provider;
import org.apache.dubbo.admin.model.dto.MetricDTO;
import org.apache.dubbo.admin.service.ConsumerService;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.admin.service.impl.MetrcisCollectServiceImpl;
import org.apache.dubbo.config.ApplicationConfig;
import org.apache.dubbo.config.ReferenceConfig;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.definition.model.MethodDefinition;
import org.apache.dubbo.metadata.definition.model.ServiceDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.apache.dubbo.monitor.MetricsService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.*;


@RestController
@RequestMapping("/api/{env}/metrics")
public class MetricsCollectController {

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @RequestMapping(method = RequestMethod.POST)
    public String metricsCollect(@RequestParam String group) {
        MetrcisCollectServiceImpl service = new MetrcisCollectServiceImpl();
        service.setUrl("dubbo://127.0.0.1:20880?scope=remote&cache=true");

        return service.invoke(group).toString();
    }

    private String getOnePortMessage(String group, String ip, String port, String protocol) {
        MetrcisCollectServiceImpl metrcisCollectService = new MetrcisCollectServiceImpl();
        metrcisCollectService.setUrl(protocol + "://" + ip + ":" + port +"?scope=remote&cache=true");
        String res = metrcisCollectService.invoke(group).toString();
        return res;
    }

    @RequestMapping( value = "/ipAddr", method = RequestMethod.GET)
    public List<MetricDTO> searchService(@RequestParam String ip, @RequestParam String group) {

        Map<String, String> configMap = new HashMap<String, String>();
        //TODO get this message from config file
        //     key:port value:protocol
        configMap.put("54188", "dubbo");
        configMap.put("54199", "dubbo");

        // default value
        if (configMap.size() <= 0) {
            configMap.put("20880", "dubbo");
        }

        List<MetricDTO> metricDTOS = new ArrayList<>();
        for (String port : configMap.keySet()) {
            String protocol = configMap.get(port);
            String res = getOnePortMessage(group, ip, port, protocol);
            metricDTOS.addAll(new Gson().fromJson(res, new TypeToken<List<MetricDTO>>(){}.getType()));
        }

        List<MethodDefinition> methods = new ArrayList<>();
        Set<String> serviceSet = new HashSet<>();
        metricDTOS.stream().forEach(metricDTO -> {
            String service = metricDTO.getService();
            if(service != null && !serviceSet.contains(service)) {
                serviceSet.add(service);
                methods.addAll(getMethodsMetaData(service, true));
                methods.addAll(getMethodsMetaData(service, false));
            }
        });
        Map map = ConvertUtil.methodList2Map(methods);

        metricDTOS.stream().forEach(metricDTO -> {
            metricDTO.changeMethod(map);
        });

        return metricDTOS;
    }

    protected List<MethodDefinition> getMethodsMetaData(String service, boolean isProvider) {
        List services = isProvider ? providerService.findByService(service) : consumerService.findByService(service);
        if(services.size() <= 0) {
            return new ArrayList<>();
        }
        String application = isProvider ? ((Provider)services.get(0)).getApplication()
                : ((Consumer)services.get(0)).getApplication();
        MetadataIdentifier metadataIdentifier = new MetadataIdentifier(service
                ,null,null, isProvider ? Constants.PROVIDER_SIDE : Constants.CONSUMER_SIDE,application);
        String metadata = isProvider ? providerService.getProviderMetaData(metadataIdentifier)
                : consumerService.getConsumerMetadata(metadataIdentifier);
        ServiceDefinition serviceDefinition = new Gson().fromJson(metadata, ServiceDefinition.class);
        return serviceDefinition.getMethods();
    }
}
