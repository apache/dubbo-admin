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

import org.apache.dubbo.admin.controller.beans.DubboApiDocsParamInfoBean;
import org.apache.dubbo.admin.model.dto.docs.CallDubboServiceRequest;
import org.apache.dubbo.admin.model.dto.docs.CallDubboServiceRequestInterfacePrarm;
import org.apache.dubbo.admin.utils.ApiDocsDubboGenericUtil;
import org.apache.dubbo.admin.utils.ApiDocsUtil;
import org.apache.dubbo.common.utils.CollectionUtils;

import com.alibaba.fastjson.JSON;
import com.alibaba.fastjson.serializer.SimplePropertyPreFilter;
import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import io.swagger.v3.oas.models.OpenAPI;
import io.swagger.v3.oas.models.info.Contact;
import io.swagger.v3.oas.models.info.Info;
import io.swagger.v3.oas.models.info.License;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import javax.servlet.http.HttpServletRequest;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collection;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;

/**
 * .
 *
 * @author klw(213539 @ qq.com)
 * @date 2020/11/5 10:42
 */
@RestController
@Slf4j
@RequestMapping("/api/docs/swagger")
@Api(tags = {"dubbo-api-docs-api-swagger"})
public class ApiDocsSwaggerController {

    /**
     * api parameter cache.
     * Map < ip:port, Map < apiClassName + "." + apiMethodName, Map < prarmType-->prarmName, DubboApiDocsPrarmInfoBean>>>
     */
    private static Map<String, Map<String, Map<String, DubboApiDocsParamInfoBean>>> API_PRARM_CACHES = new HashMap<>(16);

    private static final String PARAM_INDEX_VALUE_SEPARATOR = "-->";

    private static final SimplePropertyPreFilter CLASS_NAME_PRE_FILTER = new SimplePropertyPreFilter(HashMap.class);

    static {
        // Remove the "class" attribute from the returned result
        CLASS_NAME_PRE_FILTER.getExcludes().add("class");
    }

    @ApiOperation(value = "Generating OpenAPI 3.0 JSON for swagger", notes = "Generating OpenAPI 3.0 JSON for swagger", httpMethod = "GET", produces = "application/json")
    @GetMapping("/{dubboIp}/{dubboPort}/json")
    public OpenAPI allApiInfo(@PathVariable("dubboIp") String dubboIp, @PathVariable("dubboPort") String dubboPort) {
        CallDubboServiceRequest req = new CallDubboServiceRequest();
        req.setRegistryCenterUrl("dubbo://" + dubboIp + ":" + dubboPort);
        req.setInterfaceClassName("org.apache.dubbo.apidocs.core.providers.IDubboDocProvider");
        req.setMethodName("apiModuleListAndApiInfo");
        req.setAsync(false);
        List<Map<String, Object>> result = (List<Map<String, Object>>) callDubboService(req, null);
        if (CollectionUtils.isNotEmpty(result)) {
            Map<String, Map<String, DubboApiDocsParamInfoBean>> apiModuleCache = new HashMap<>(result.size());
            for (Map<String, Object> apiModule : result) {
                List<Map<String, Object>> apiList = (List<Map<String, Object>>) apiModule.get("moduleApiList");
                if (CollectionUtils.isNotEmpty(apiList)) {
                    for (Map<String, Object> apiInfo : apiList) {
                        List<Map<String, Object>> prarmInfoList = (List<Map<String, Object>>) apiInfo.get("params");
                        if (CollectionUtils.isNotEmpty(prarmInfoList)) {
                            Map<String, DubboApiDocsParamInfoBean> prarmsCache = new HashMap<>(prarmInfoList.size());
                            for (Map<String, Object> prarmInfo : prarmInfoList) {
                                List<Map<String, String>> prarmList = (List<Map<String, String>>) prarmInfo.get("prarmInfo");
                                if(CollectionUtils.isNotEmpty(prarmList)) {
                                    for (Map<String, String> prarm : prarmList) {
                                        int prarmIndex = (Integer) prarmInfo.get("prarmIndex");
                                        prarmsCache.put(prarmInfo.get("prarmType") + PARAM_INDEX_VALUE_SEPARATOR + prarm.get("name"), new DubboApiDocsParamInfoBean(
                                                prarm.get("name"), prarm.get("javaType"), (String) prarmInfo.get("prarmType"), prarmIndex));
                                    }
                                } else {
                                    int prarmIndex = (Integer) prarmInfo.get("prarmIndex");
                                    prarmsCache.put(prarmInfo.get("prarmType") + PARAM_INDEX_VALUE_SEPARATOR + prarmInfo.get("name"), new DubboApiDocsParamInfoBean(
                                            (String)prarmInfo.get("name"), (String)prarmInfo.get("prarmType"), (String) prarmInfo.get("prarmType"), prarmIndex));
                                }
                            }
                            apiModuleCache.put(apiModule.get("moduleClassName") + "." + apiInfo.get("apiName"), prarmsCache);
                        }
                    }
                }
            }
            API_PRARM_CACHES.put(dubboIp + ":" + dubboPort, apiModuleCache);
        }

        OpenAPI oas = new OpenAPI();
        Info info = new Info()
                .title("Swagger Sample App bootstrap code")
                .description("This is a sample server Petstore server.  You can find out more about Swagger " +
                        "at [http://swagger.io](http://swagger.io) or on [irc.freenode.net, #swagger](http://swagger.io/irc/).  For this sample, " +
                        "you can use the api key `special-key` to test the authorization filters.")
                .termsOfService("http://swagger.io/terms/")
                .contact(new Contact()
                        .email("apiteam@swagger.io"))
                .license(new License()
                        .name("Apache 2.0")
                        .url("http://www.apache.org/licenses/LICENSE-2.0.html"));

        oas.info(info);
        return oas;
    }

    @ApiOperation(value = "request dubbo api for swagger", notes = "request dubbo api for swagger", httpMethod = "POST", produces = "application/json")
    @PostMapping("/{dubboIp}/{dubboPort}/{interfaceClassName}/{methodName}/{async}/requestDubbo")
    public String requestDubbo(@PathVariable("dubboIp") String dubboIp, @PathVariable("dubboPort") String dubboPort,
                               @PathVariable("interfaceClassName") String interfaceClassName, @PathVariable("methodName") String methodName,
                               @PathVariable("async") boolean async, HttpServletRequest request) {
        CallDubboServiceRequestInterfacePrarm[] postData = null;
        Map<String, Map<String, DubboApiDocsParamInfoBean>> apiModuleCache = API_PRARM_CACHES.get(dubboIp + ":" + dubboPort);
        if (CollectionUtils.isNotEmptyMap(apiModuleCache)) {
            Map<String, DubboApiDocsParamInfoBean> apiInfoCache = apiModuleCache.get(interfaceClassName + "." + methodName);
            if (CollectionUtils.isNotEmptyMap(apiInfoCache)) {
                Map<Integer, List<DubboApiDocsParamInfoBean>> tempMap = new HashMap<>(5);
                Map<String, String[]> parameterMap = request.getParameterMap();
                for (Map.Entry<String, String[]> parameterEntry : parameterMap.entrySet()) {
                    DubboApiDocsParamInfoBean paramInfoBean = apiInfoCache.get(parameterEntry.getKey());
                    if (null == paramInfoBean) {
                        continue;
                    }
                    paramInfoBean.setFieldValue(parameterEntry.getValue()[0]);
                    int tempMapKey = paramInfoBean.getMethodParamIndex();
                    List<DubboApiDocsParamInfoBean> tempMapValueArray = tempMap.get(tempMapKey);
                    if (null == tempMapValueArray) {
                        tempMapValueArray = new ArrayList<>(16);
                        tempMap.put(tempMapKey, tempMapValueArray);
                    }
                    tempMapValueArray.add(paramInfoBean);
                }

                if (!tempMap.isEmpty()) {
                    postData = new CallDubboServiceRequestInterfacePrarm[tempMap.size()];
                    for (Map.Entry<Integer, List<DubboApiDocsParamInfoBean>> tempMapEntry : tempMap.entrySet()) {
                        for (DubboApiDocsParamInfoBean paramInfoBean : tempMapEntry.getValue()) {
                            CallDubboServiceRequestInterfacePrarm postDataItem = postData[paramInfoBean.getMethodParamIndex()];
                            if (null == postDataItem) {
                                postDataItem = new CallDubboServiceRequestInterfacePrarm();
                                postDataItem.setPrarmType(paramInfoBean.getMethodParamType());
                                postData[paramInfoBean.getMethodParamIndex()] = postDataItem;
                                postDataItem.setPrarmValue(new HashMap<String, Object>(16));
                            }

                            String elementName = paramInfoBean.getFieldName();
                            Object elementValue = paramInfoBean.getFieldValue();
                            if (null != elementValue && elementValue instanceof String && ApiDocsUtil.isJsonStr((String) elementValue)) {
                                ((Map<String, Object>) postDataItem.getPrarmValue()).put(elementName, JSON.parse((String) elementValue));
                            } else {
                                ((Map<String, Object>) postDataItem.getPrarmValue()).put(elementName, elementValue);
                            }

                        }
                    }
                }
            }
        }

        CallDubboServiceRequest dubboCfg = new CallDubboServiceRequest();
        dubboCfg.setRegistryCenterUrl("dubbo://" + dubboIp + ":" + dubboPort);
        dubboCfg.setInterfaceClassName(interfaceClassName);
        dubboCfg.setMethodName(methodName);
        dubboCfg.setAsync(async);
        Object objResult = this.callDubboService(dubboCfg, postData == null ? null : Arrays.asList(postData));
        return JSON.toJSONString(objResult, CLASS_NAME_PRE_FILTER);
    }

    private Object callDubboService(CallDubboServiceRequest dubboCfg, List<CallDubboServiceRequestInterfacePrarm> methodPrarms) {
        String[] prarmTypes = null;
        Object[] prarmValues = null;

        if (null != methodPrarms && !methodPrarms.isEmpty()) {
            prarmTypes = new String[methodPrarms.size()];
            prarmValues = new Object[methodPrarms.size()];
            for (int i = 0; i < methodPrarms.size(); i++) {
                CallDubboServiceRequestInterfacePrarm prarm = methodPrarms.get(i);
                prarmTypes[i] = prarm.getPrarmType();
                Object prarmValue = prarm.getPrarmValue();
                if (isBaseType(prarm.getPrarmType()) && null != prarmValue) {
                    if (prarmValue instanceof Map) {
                        Map<?, ?> tempMap = (Map<?, ?>) prarmValue;
                        if (!tempMap.isEmpty()) {
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
        CompletableFuture<Object> future = ApiDocsDubboGenericUtil.invoke(dubboCfg.getRegistryCenterUrl(), dubboCfg.getInterfaceClassName(),
                dubboCfg.getMethodName(), dubboCfg.isAsync(), prarmTypes, prarmValues);
        try {
            return future.get();
        } catch (InterruptedException | ExecutionException e) {
            log.error(e.getMessage(), e);
            return "Some exceptions have occurred, please check the log.";
        }
    }

    private Object emptyString2Null(Object prarmValue) {
        if (null != prarmValue) {
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
