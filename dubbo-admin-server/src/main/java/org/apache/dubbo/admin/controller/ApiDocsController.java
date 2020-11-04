/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.controller.editors.CustomLocalDateEditor;
import org.apache.dubbo.admin.controller.editors.CustomLocalDateTimeEditor;
import org.apache.dubbo.admin.model.dto.docs.ApiInfoRequest;
import org.apache.dubbo.admin.model.dto.docs.CallDubboServiceRequest;
import org.apache.dubbo.admin.model.dto.docs.CallDubboServiceRequestInterfacePrarm;
import org.apache.dubbo.admin.utils.DubboUtil;

import com.alibaba.fastjson.JSON;
import com.alibaba.fastjson.serializer.SimplePropertyPreFilter;
import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.beans.propertyeditors.StringTrimmerEditor;
import org.springframework.web.bind.WebDataBinder;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.InitBinder;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.annotation.PostConstruct;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;

/**
 * dubbo doc ui server api.
 * @author klw(213539@qq.com)
 * 2020/11/2 11:12
 */
@Api(tags = {"dubbo-api-docs-api"})
@RestController
@Slf4j
@RequestMapping("/api/{env}/docs")
public class ApiDocsController {

    private static final SimplePropertyPreFilter CLASS_NAME_PRE_FILTER = new SimplePropertyPreFilter(HashMap.class);
    static {
        // Remove the "class" attribute from the returned result
        CLASS_NAME_PRE_FILTER.getExcludes().add("class");
    }

    /**
     * retries for dubbo provider
     */
    @Value("${dubbo.consumer.retries:0}")
    private int retries;

    /**
     * timeout
     */
    @Value("${dubbo.consumer.timeout:1000}")
    private int timeout;

    @InitBinder
    public void initBinder(WebDataBinder binder) {
        binder.registerCustomEditor(String.class, new StringTrimmerEditor(true));
        binder.registerCustomEditor(LocalDate.class, new CustomLocalDateEditor());
        binder.registerCustomEditor(LocalDateTime.class, new CustomLocalDateTimeEditor());
    }

    /**
     * Set timeout and retries for {@link org.apache.dubbo.admin.utils.DubboUtil}
     * 2020-11-02 11:16:28
     * @param:
     * @return void
     */
    @PostConstruct
    public void setRetriesAndTimeout(){
        DubboUtil.setRetriesAndTimeout(retries, timeout);
    }

    @ApiOperation(value = "request dubbo api", notes = "request dubbo api", httpMethod = "POST", produces = "application/json")
    @PostMapping("/requestDubbo")
    public String callDubboService(CallDubboServiceRequest dubboCfg, @RequestBody List<CallDubboServiceRequestInterfacePrarm> methodPrarms){
        String[] prarmTypes = null;
        Object[] prarmValues = null;
        if(null != methodPrarms && !methodPrarms.isEmpty()){
            prarmTypes = new String[methodPrarms.size()];
            prarmValues = new Object[methodPrarms.size()];
            for(int i = 0; i < methodPrarms.size(); i++){
                CallDubboServiceRequestInterfacePrarm prarm = methodPrarms.get(i);
                prarmTypes[i] = prarm.getPrarmType();
                Object prarmValue = prarm.getPrarmValue();
                if(isBaseType(prarm.getPrarmType()) && null != prarmValue){
                    if(prarmValue instanceof Map){
                        Map<?, ?> tempMap = (Map<?, ?>) prarmValue;
                        if(!tempMap.isEmpty()) {
                            this.emptyString2Null(tempMap);
                            prarmValues[i] = tempMap.values().stream().findFirst().orElse(null);
                        }
                    } else {
                        prarmValues[i] = emptyString2Null(prarmValue);
                    }
                } else {
                    this.emptyString2Null(prarmValue);
                    prarmValues[i] = prarmValue;
                }
            }
        }
        if (null == prarmTypes) {
            prarmTypes = new String[0];
        }
        if (null == prarmValues) {
            prarmValues = new Object[0];
        }
        CompletableFuture<Object> future = DubboUtil.invoke(dubboCfg.getRegistryCenterUrl(), dubboCfg.getInterfaceClassName(),
                dubboCfg.getMethodName(), dubboCfg.isAsync(), prarmTypes, prarmValues);
        try {
            Object objResult = future.get();
            return JSON.toJSONString(objResult, CLASS_NAME_PRE_FILTER);
        } catch (InterruptedException | ExecutionException e) {
            log.error(e.getMessage(), e);
            return "Some exceptions have occurred, please check the log.";
        }
    }

    private Object emptyString2Null(Object prarmValue){
        if(null != prarmValue) {
            if (prarmValue instanceof String && StringUtils.isBlank((String) prarmValue)) {
                return null;
            } else if (prarmValue instanceof Map) {
                Map<String, Object> tempMap = (Map<String, Object>) prarmValue;
                tempMap.forEach((k, v) -> {
                    if (v != null && v instanceof String && StringUtils.isBlank((String) v)) {
                        tempMap.put(k, null);
                    } else {
                        this.emptyString2Null(v);
                    }
                });
            }
        }
        return prarmValue;
    }

    @ApiOperation(value = "Get basic information of all modules, excluding API parameter information", notes = "Get basic information of all modules, excluding API parameter information", httpMethod = "GET", produces = "application/json")
    @GetMapping("/apiModuleList")
    public String apiModuleList(ApiInfoRequest apiInfoRequest){
        CallDubboServiceRequest req = new CallDubboServiceRequest();
        req.setRegistryCenterUrl("dubbo://" + apiInfoRequest.getDubboIp() + ":" + apiInfoRequest.getDubboPort());
        req.setInterfaceClassName("org.apache.dubbo.apidocs.core.providers.IDubboDocProvider");
        req.setMethodName("apiModuleList");
        req.setAsync(false);
        return callDubboService(req, null);
    }

    @ApiOperation(value = "Get the parameter information of the specified API", notes = "Get the parameter information of the specified API", httpMethod = "GET", produces = "application/json")
    @GetMapping("/apiParamsResp")
    public String apiParamsResp(ApiInfoRequest apiInfoRequest){
        CallDubboServiceRequest req = new CallDubboServiceRequest();
        req.setRegistryCenterUrl("dubbo://" + apiInfoRequest.getDubboIp() + ":" + apiInfoRequest.getDubboPort());
        req.setInterfaceClassName("org.apache.dubbo.apidocs.core.providers.IDubboDocProvider");
        req.setMethodName("apiParamsResponseInfo");
        req.setAsync(false);

        List<CallDubboServiceRequestInterfacePrarm> methodPrarms = new ArrayList<>(1);
        CallDubboServiceRequestInterfacePrarm prarm = new CallDubboServiceRequestInterfacePrarm();
        prarm.setPrarmType(String.class.getName());
        prarm.setPrarmValue(apiInfoRequest.getApiName());
        methodPrarms.add(prarm);
        return callDubboService(req, methodPrarms);
    }

    private static boolean isBaseType(String typeStr) {
        if ("java.lang.Integer".equals(typeStr) ||
                "java.lang.Byte".equals(typeStr) ||
                "java.lang.Long".equals(typeStr) ||
                "java.lang.Double".equals(typeStr) ||
                "java.lang.Float".equals(typeStr) ||
                "java.lang.Character".equals(typeStr) ||
                "java.lang.Short".equals(typeStr) ||
                "java.lang.Boolean".equals(typeStr) ||
                "java.lang.String".equals(typeStr)) {
            return true;
        }
        return false;
    }
}
