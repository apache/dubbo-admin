package org.apache.dubbo.admin.controller;

import com.google.gson.Gson;
import org.apache.dubbo.admin.common.util.ConvertUtil;
import org.apache.dubbo.admin.common.util.ServiceTestUtil;
import org.apache.dubbo.admin.model.domain.MethodMetadata;
import org.apache.dubbo.admin.model.dto.ServiceTestDTO;
import org.apache.dubbo.admin.service.ProviderService;
import org.apache.dubbo.admin.service.impl.GenericServiceImpl;
import org.apache.dubbo.common.Constants;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.definition.model.MethodDefinition;
import org.apache.dubbo.metadata.identifier.MetadataIdentifier;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/{env}/test")
public class ServiceTestController {

    @Autowired
    private GenericServiceImpl genericService;

    @Autowired
    private ProviderService providerService;

    @RequestMapping(method = RequestMethod.POST)
    public Object test(@PathVariable String env, @RequestBody ServiceTestDTO serviceTestDTO) {
        return genericService.invoke(serviceTestDTO.getService(), serviceTestDTO.getMethod(), serviceTestDTO.getParameterTypes(), serviceTestDTO.getParams());
    }

    @RequestMapping(value = "/method", method = RequestMethod.GET)
    public MethodMetadata methodDetail(@PathVariable String env, @RequestParam String application, @RequestParam String service,
                                       @RequestParam String method) {
        Map<String, String> info = ConvertUtil.serviceName2Map(service);
        MetadataIdentifier identifier = new MetadataIdentifier(info.get(Constants.INTERFACE_KEY),
                info.get(Constants.VERSION_KEY),
                info.get(Constants.GROUP_KEY), Constants.PROVIDER_SIDE, application);
        String metadata = providerService.getProviderMetaData(identifier);
        MethodMetadata methodMetadata = null;
        if (metadata != null) {
            Gson gson = new Gson();
            FullServiceDefinition serviceDefinition = gson.fromJson(metadata, FullServiceDefinition.class);
            List<MethodDefinition> methods = serviceDefinition.getMethods();
            if (methods != null) {
                for (MethodDefinition m : methods) {
                    if (ServiceTestUtil.sameMethod(m, method)) {
                        methodMetadata = ServiceTestUtil.generateMethodMeta(serviceDefinition, m);
                        break;
                    }
                }
            }
        }
        return methodMetadata;
    }
}
