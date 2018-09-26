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

package org.apache.dubbo.admin.web.mvc.governance;

import org.apache.dubbo.admin.registry.common.domain.Override;
import com.alibaba.dubbo.common.URL;
import com.alibaba.dubbo.common.utils.StringUtils;
import org.apache.dubbo.admin.governance.service.ConsumerService;
import org.apache.dubbo.admin.governance.service.OverrideService;
import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Consumer;
import org.apache.dubbo.admin.registry.common.domain.Provider;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.apache.dubbo.admin.registry.common.route.OverrideUtils;
import org.apache.dubbo.admin.registry.common.route.RouteRule;
import org.apache.dubbo.admin.web.mvc.BaseController;
import org.apache.dubbo.admin.web.pulltool.Tool;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.util.*;


@Controller
public class ServiceDetailController extends BaseController{

    @Autowired
    private ProviderService providerService;

    @Autowired
    private ConsumerService consumerService;

    @Autowired
    private OverrideService overrideService;

    @Autowired
    private RouteService routeService;

    @RequestMapping("/serviceDetail")
    public String serviceDetail(@RequestParam(required = false) String app,
                                @RequestParam(required = false) String service,
                                HttpServletRequest request,
                                HttpServletResponse response, Model model) {
        model.addAttribute("service", service);
        model.addAttribute("app", app);
        return "serviceDetail";
    }


    @RequestMapping(value =  "/create", method = RequestMethod.POST)  //post
    public boolean create(@RequestParam String service, @ModelAttribute Provider provider,
                          HttpServletRequest request, HttpServletResponse response,
                          Model model) {
        prepare(request, response, model,"create" ,"providers");
        boolean success = true;
        if (provider.getService() == null) {
            provider.setService(service);
        }
        if (provider.getParameters() == null) {
            String url = provider.getUrl();
            if (url != null) {
                int i = url.indexOf('?');
                if (i > 0) {
                    provider.setUrl(url.substring(0, i));
                    provider.setParameters(url.substring(i + 1));
                }
            }
        }
        provider.setDynamic(false); // Provider add through web page must be static
        providerService.create(provider);
        return true;
    }

    @RequestMapping(value = "/update", method = RequestMethod.POST) //post
    public boolean update(@ModelAttribute Provider newProvider, HttpServletRequest request, HttpServletResponse response, Model model) {
        boolean success = true;
        Long id = newProvider.getId();
        String parameters = newProvider.getParameters();
        Provider provider = providerService.findProvider(id);
        if (provider == null) {
            return false;
        }
        String service = provider.getService();
        Map<String, String> oldMap = StringUtils.parseQueryString(provider.getParameters());
        Map<String, String> newMap = StringUtils.parseQueryString(parameters);
        for (Map.Entry<String, String> entry : oldMap.entrySet()) {
            if (entry.getValue().equals(newMap.get(entry.getKey()))) {
                newMap.remove(entry.getKey());
            }
        }
        if (provider.isDynamic()) {
            String address = provider.getAddress();
            List<Override> overrides = overrideService.findByServiceAndAddress(provider.getService(), provider.getAddress());
            OverrideUtils.setProviderOverrides(provider, overrides);
            Override override = provider.getOverride();
            if (override != null) {
                if (newMap.size() > 0) {
                    override.setParams(StringUtils.toQueryString(newMap));
                    override.setEnabled(true);
                    override.setOperator(operator);
                    override.setOperatorAddress(operatorAddress);
                    overrideService.updateOverride(override);
                } else {
                    overrideService.deleteOverride(override.getId());
                }
            } else {
                override = new Override();
                override.setService(service);
                override.setAddress(address);
                override.setParams(StringUtils.toQueryString(newMap));
                override.setEnabled(true);
                override.setOperator(operator);
                override.setOperatorAddress(operatorAddress);
                overrideService.saveOverride(override);
            }
        } else {
            provider.setParameters(parameters);
            providerService.updateProvider(provider);
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../providers");
        return true;
    }

    @RequestMapping("/delete")
    public String delete(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) {
        //prepare(request, response, model, "delete", "providers");
        boolean success = true;
        for (Long id : ids) {
            Provider provider = providerService.findProvider(id);
            if (provider == null) {
                model.addAttribute("message", getMessage("NoSuchOperationData", id));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            } else if (provider.isDynamic()) {
                model.addAttribute("message", getMessage("CanNotDeleteDynamicData", id));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
            else if (!super.currentUser.hasServicePrivilege(provider.getService())) {
                model.addAttribute("message", getMessage("HaveNoServicePrivilege", provider.getService()));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
        }
        for (Long id : ids) {
            providerService.deleteStaticProvider(id);
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../../providers");
        return "governance/screen/redirect";
    }

    @RequestMapping("/enable")
    public String enable(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) {
        prepare(request, response, model, "enable", "providers");
        boolean success = true;
        Map<Long, Provider> id2Provider = new HashMap<Long, Provider>();
        for (Long id : ids) {
            Provider provider = providerService.findProvider(id);
            if (provider == null) {
                model.addAttribute("message", getMessage("NoSuchOperationData", id));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
            else if (!super.currentUser.hasServicePrivilege(provider.getService())) {
                model.addAttribute("message", getMessage("HaveNoServicePrivilege", provider.getService()));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
            id2Provider.put(id, provider);
        }
        for (Long id : ids) {
            providerService.enableProvider(id);
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../../providers");
        return "governance/screen/redirect";
    }


    @RequestMapping("/disable")
    public String disable(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response,  Model model) {
        //prepare(request, response, model, "disable", "providers");
        boolean success = true;
        for (Long id : ids) {
            Provider provider = providerService.findProvider(id);
            if (provider == null) {
                model.addAttribute("message", getMessage("NoSuchOperationData", id));
                success = false;
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
            else if (!super.currentUser.hasServicePrivilege(provider.getService())) {
                success = false;
                model.addAttribute("message", getMessage("HaveNoServicePrivilege", provider.getService()));
                model.addAttribute("success", success);
                model.addAttribute("redirect", "../../providers");
                return "governance/screen/redirect";
            }
        }
        for (Long id : ids) {
            providerService.disableProvider(id);
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../../providers");
        return "governance/screen/redirect";
    }

    @RequestMapping("/shield")
    public String shield(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return mock(ids, "force:return null", "shield", request, response, model);
    }

    @RequestMapping("/tolerant")
    public String tolerant(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return mock(ids, "fail:return null", "tolerant", request, response, model);
    }

    @RequestMapping("/recover")
    public String recover(@RequestParam("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return mock(ids,  "", "recover", request, response, model);
    }

    private String mock(Long[] ids, String mock, String methodName, HttpServletRequest request,
                        HttpServletResponse response, Model model) throws Exception {
        prepare(request, response, model, methodName, "consumers");
        boolean success = true;
        if (ids == null || ids.length == 0) {
            model.addAttribute("message", getMessage("NoSuchOperationData"));
            success = false;
            model.addAttribute("success", success);
            model.addAttribute("redirect", "../../consumers");
            return "governance/screen/redirect";
        }
        List<Consumer> consumers = new ArrayList<Consumer>();
        for (Long id : ids) {
            Consumer c = consumerService.findConsumer(id);
            if (c != null) {
                consumers.add(c);
                if (!super.currentUser.hasServicePrivilege(c.getService())) {
                    model.addAttribute("message", getMessage("HaveNoServicePrivilege", c.getService()));
                    success = false;
                    model.addAttribute("success", success);
                    model.addAttribute("redirect", "../../consumers");
                    return "governance/screen/redirect";
                }
            }
        }
        for (Consumer consumer : consumers) {
            String service = consumer.getService();
            String address = Tool.getIP(consumer.getAddress());
            List<Override> overrides = overrideService.findByServiceAndAddress(service, address);
            if (overrides != null && overrides.size() > 0) {
                for (Override override : overrides) {
                    Map<String, String> map = StringUtils.parseQueryString(override.getParams());
                    if (mock == null || mock.length() == 0) {
                        map.remove("mock");
                    } else {
                        map.put("mock", URL.encode(mock));
                    }
                    if (map.size() > 0) {
                        override.setParams(StringUtils.toQueryString(map));
                        override.setEnabled(true);
                        //override.setOperator(operator);
                        override.setOperatorAddress(operatorAddress);
                        overrideService.updateOverride(override);
                    } else {
                        overrideService.deleteOverride(override.getId());
                    }
                }
            } else if (mock != null && mock.length() > 0) {
                Override override = new Override();
                override.setService(service);
                override.setAddress(address);
                override.setParams("mock=" + URL.encode(mock));
                override.setEnabled(true);
                override.setOperator(operator);
                override.setOperatorAddress(operatorAddress);
                overrideService.saveOverride(override);
            }
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../../consumers");
        return "governance/screen/redirect";
    }

    @RequestMapping("/allshield")
    public String allshield(@RequestParam(required = false) String service, HttpServletRequest request,
                            HttpServletResponse response, Model model) throws Exception {
        return allmock(service,  "force:return null", "allshield",request, response, model);
    }

    @RequestMapping("/alltolerant")
    public String alltolerant(@RequestParam(required = false) String service, HttpServletRequest request,
                              HttpServletResponse response, Model model) throws Exception {
        return allmock(service, "fail:return null", "alltolerant", request, response, model);
    }

    @RequestMapping("/allrecover")
    public String allrecover(@RequestParam(required = false) String service, HttpServletRequest request,
                             HttpServletResponse response, Model model) throws Exception {
        return allmock(service, "", "allrecover", request, response, model);
    }

    private String allmock(String service, String mock, String methodName, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        String operatorAddress = request.getRemoteAddr();
        prepare(request, response, model, methodName,"consumers");
        boolean success = true;
        if (service == null || service.length() == 0) {
            model.addAttribute("message", getMessage("NoSuchOperationData"));
            success = false;
            model.addAttribute("success", success);
            model.addAttribute("redirect", "../consumers");
            return "governance/screen/redirect";
        }
        if (!super.currentUser.hasServicePrivilege(service)) {
            model.addAttribute("message", getMessage("HaveNoServicePrivilege", service));
            success = false;
            model.addAttribute("success", success);
            model.addAttribute("redirect", "../consumers");
            return "governance/screen/redirect";
        }
        List<Override> overrides = overrideService.findByService(service);
        Override allOverride = null;
        if (overrides != null && overrides.size() > 0) {
            for (Override override : overrides) {
                if (override.isDefault()) {
                    allOverride = override;
                    break;
                }
            }
        }
        if (allOverride != null) {
            Map<String, String> map = StringUtils.parseQueryString(allOverride.getParams());
            if (mock == null || mock.length() == 0) {
                map.remove("mock");
            } else {
                map.put("mock", URL.encode(mock));
            }
            if (map.size() > 0) {
                allOverride.setParams(StringUtils.toQueryString(map));
                allOverride.setEnabled(true);
                //allOverride.setOperator(operator);
                allOverride.setOperatorAddress(operatorAddress);
                overrideService.updateOverride(allOverride);
            } else {
                overrideService.deleteOverride(allOverride.getId());
            }
        } else if (mock != null && mock.length() > 0) {
            Override override = new Override();
            override.setService(service);
            override.setParams("mock=" + URL.encode(mock));
            override.setEnabled(true);
            override.setOperator(operator);
            override.setOperatorAddress(operatorAddress);
            overrideService.saveOverride(override);
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../consumers");
        return "governance/screen/redirect";
    }

    @RequestMapping("/{ids}/allow")
    public String allow(@PathVariable("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return access(request, response, ids, model, true, false, "allow");
    }

    @RequestMapping("/{ids}/forbid")
    public String forbid(@PathVariable("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return access(request, response, ids, model, false, false, "forbid");
    }

    @RequestMapping("/{ids}/onlyallow")
    public String onlyallow(@PathVariable("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return access(request, response, ids, model, true, true, "onlyallow");
    }

    @RequestMapping("/{ids}/onlyforbid")
    public String onlyforbid(@PathVariable("ids") Long[] ids, HttpServletRequest request, HttpServletResponse response, Model model) throws Exception {
        return access(request, response, ids, model, false, true, "onlyforbid");
    }

    private String access(HttpServletRequest request, HttpServletResponse response, Long[] ids,
                          Model model, boolean allow, boolean only, String methodName) throws Exception {
        prepare(request, response, model, methodName, "consumers");
        boolean success = true;
        if (ids == null || ids.length == 0) {
            model.addAttribute("message", getMessage("NoSuchOperationData"));
            success = false;
            model.addAttribute("success", success);
            model.addAttribute("redirect", "../../consumers");
            return "governance/screen/redirect";
        }
        List<Consumer> consumers = new ArrayList<Consumer>();
        for (Long id : ids) {
            Consumer c = consumerService.findConsumer(id);
            if (c != null) {
                consumers.add(c);
                if (!super.currentUser.hasServicePrivilege(c.getService())) {
                    model.addAttribute("message", getMessage("HaveNoServicePrivilege", c.getService()));
                    success = false;
                    model.addAttribute("success", success);
                    model.addAttribute("redirect", "../../consumers");
                    return "governance/screen/redirect";
                }
            }
        }
        Map<String, Set<String>> serviceAddresses = new HashMap<String, Set<String>>();
        for (Consumer consumer : consumers) {
            String service = consumer.getService();
            String address = Tool.getIP(consumer.getAddress());
            Set<String> addresses = serviceAddresses.get(service);
            if (addresses == null) {
                addresses = new HashSet<String>();
                serviceAddresses.put(service, addresses);
            }
            addresses.add(address);
        }
        for (Map.Entry<String, Set<String>> entry : serviceAddresses.entrySet()) {
            String service = entry.getKey();
            boolean isFirst = false;
            List<Route> routes = routeService.findForceRouteByService(service);
            Route route = null;
            if (routes == null || routes.size() == 0) {
                isFirst = true;
                route = new Route();
                route.setService(service);
                route.setForce(true);
                route.setName(service + " blackwhitelist");
                route.setFilterRule("false");
                route.setEnabled(true);
            } else {
                route = routes.get(0);
            }
            Map<String, RouteRule.MatchPair> when = null;
            RouteRule.MatchPair matchPair = null;
            if (isFirst) {
                when = new HashMap<String, RouteRule.MatchPair>();
                matchPair = new RouteRule.MatchPair(new HashSet<String>(), new HashSet<String>());
                when.put("consumer.host", matchPair);
            } else {
                when = RouteRule.parseRule(route.getMatchRule());
                matchPair = when.get("consumer.host");
            }
            if (only) {
                matchPair.getUnmatches().clear();
                matchPair.getMatches().clear();
                if (allow) {
                    matchPair.getUnmatches().addAll(entry.getValue());
                } else {
                    matchPair.getMatches().addAll(entry.getValue());
                }
            } else {
                for (String consumerAddress : entry.getValue()) {
                    if (matchPair.getUnmatches().size() > 0) { // whitelist take effect
                        matchPair.getMatches().remove(consumerAddress); // remove data in blacklist
                        if (allow) { // if allowed
                            matchPair.getUnmatches().add(consumerAddress); // add to whitelist
                        } else { // if not allowed
                            matchPair.getUnmatches().remove(consumerAddress); // remove from whitelist
                        }
                    } else { // blacklist take effect
                        if (allow) { // if allowed
                            matchPair.getMatches().remove(consumerAddress); // remove from blacklist
                        } else { // if not allowed
                            matchPair.getMatches().add(consumerAddress); // add to blacklist
                        }
                    }
                }
            }
            StringBuilder sb = new StringBuilder();
            RouteRule.contidionToString(sb, when);
            route.setMatchRule(sb.toString());
            route.setUsername(operator);
            if (matchPair.getMatches().size() > 0 || matchPair.getUnmatches().size() > 0) {
                if (isFirst) {
                    routeService.createRoute(route);
                } else {
                    routeService.updateRoute(route);
                }
            } else if (!isFirst) {
                routeService.deleteRoute(route.getId());
            }
        }
        model.addAttribute("success", success);
        model.addAttribute("redirect", "../../consumers");
        return "governance/screen/redirect";
    }

    @RequestMapping("/metaData")
    public List<String> metaData(@RequestParam String app, @RequestParam String service) {
        return null;
    }

}
