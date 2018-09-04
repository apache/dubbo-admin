package org.apache.dubbo.admin.controller;

import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/routes")
public class RoutesController {


    @RequestMapping("/create")
    public void createRule(@RequestParam String serviceName, @RequestParam String rule) {

    }

}