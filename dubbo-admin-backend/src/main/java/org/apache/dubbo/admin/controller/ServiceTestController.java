package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.model.dto.ServiceTestDTO;
import org.apache.dubbo.admin.service.impl.GenericServiceImpl;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/{env}/test")
public class ServiceTestController {

    @Autowired
    private GenericServiceImpl genericService;

    @RequestMapping(method = RequestMethod.POST)
    public Object test(@PathVariable String env, @RequestBody ServiceTestDTO serviceTestDTO) {
        return genericService.invoke(serviceTestDTO.getService(), serviceTestDTO.getMethod(), serviceTestDTO.getTypes(), serviceTestDTO.getParams());
//        return null;
    }
}
