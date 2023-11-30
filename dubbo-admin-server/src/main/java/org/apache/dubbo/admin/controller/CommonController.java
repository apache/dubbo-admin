package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.model.dto.AppConfigDTO;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * There is no description.
 *
 * @author XS <wanghaiqi@beeplay123.com>
 * @version 1.0
 * @date 2023/11/28 13:47
 */
@RestController
@RequestMapping("/api/{env}/common")
public class CommonController {

    /**
     * Admin Title
     */
    @Value("${admin.title:Dubbo Admin}")
    private String adminTitle;

    /**
     * @return {@link AppConfigDTO}
     */
    @GetMapping(value = "/app/config")
    public AppConfigDTO appConfig() {
        return AppConfigDTO.builder().adminTitle(adminTitle).build();
    }

}
