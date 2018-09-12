package org.apache.dubbo.admin.controller;

import org.apache.dubbo.admin.governance.service.ProviderService;
import org.apache.dubbo.admin.governance.service.RouteService;
import org.apache.dubbo.admin.registry.common.domain.Route;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.yaml.snakeyaml.Yaml;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/routes")
public class RoutesController {

    @Autowired
    private RouteService routeService;
    @Autowired
    private ProviderService providerService;

    @RequestMapping("/create")
    public void createRule(@RequestParam(required = false) String serviceName,
                           @RequestParam(required = false) String app,
                           @RequestParam String rule) {
        if (serviceName == null && app == null) {

        }
        Yaml yaml = new Yaml();
        Map<String, Object> result = yaml.load(rule);
        if (serviceName != null) {
            result.put("scope", serviceName);
            result.put("group/service:version", result.get("group") + "/" + serviceName);
            //2.6
            String version = null;
            String service = serviceName;
            if (serviceName.contains(":") && !serviceName.endsWith(":")) {
                version = serviceName.split(":")[1];
                service = serviceName.split(":")[0];
            }

            List<String> conditions = (List)result.get("conditions");
            for (String condition : conditions) {
                Route route = new Route();
                route.setService(service);
                route.setVersion(version);
                route.setEnabled((boolean)getParameter(result, "enabled", true));
                route.setForce((boolean)getParameter(result, "force", false));
                route.setGroup((String)getParameter(result, "group", null));
                route.setDynamic((boolean)getParameter(result, "dynamic", false));
                route.setRuntime((boolean)getParameter(result, "runtime", false));
                route.setPriority((int)getParameter(result, "priority", 0));
                route.setRule(condition);
                routeService.createRoute(route);
            }


        } else {
            //new feature in 2.7
            result.put("scope", "application");
            result.put("appname", app);
        }

    }

    @RequestMapping("/all")
    public List<Route> allRoutes(@RequestParam(required = false) String serviceName,
                                 @RequestParam(required = false) String app) {
        List<Route> routes = null;
        if (app != null) {
           // app scope in 2.7
        }
        if (serviceName != null) {
            routes = routeService.findByService(serviceName);
        }
        if (serviceName == null && app == null) {
            // TODO throw exception
        }
        //no support for findAll and findByaddress
        return routes;
    }

    @RequestMapping("/detail")
    public Route routeDetail(@RequestParam long id) {
        Route route = routeService.findRoute(id);
        if (route == null) {
            // TODO throw exception
        }
        return route;
    }

    @RequestMapping("/delete")
    public boolean deleteRoute(@RequestParam long id) {
        routeService.deleteRoute(id);
        return true;
    }

    @RequestMapping("/edit")
    public Route editRule(@RequestParam long id) {
        Route route = routeService.findRoute(id);
        if (route == null) {
            // TODO throw exception
        }
        return route;
    }

    @RequestMapping("/enable")
    public boolean enableRoute(@RequestParam long id) {
        routeService.enableRoute(id);
        return true;
    }

    @RequestMapping("/disable")
    public boolean disableRoute(@RequestParam long id) {
        routeService.disableRoute(id);
        return true;
    }

    private Object getParameter(Map<String, Object> result, String key, Object defaultValue) {
        if (result.get(key) != null) {
            return result.get(key);
        }
        return defaultValue;
    }

    public static void main(String[] args) {
        String yaml =
                "enable: true\n" +
                        "priority: 0\n" +
                        "runtime: true\n" +
                        "category: routers\n" +
                        "dynamic: true\n" +
                        "conditions:\n" +
                        "  - '=> host != 172.22.3.91'\n" +
                        "  - 'host != 10.20.153.10,10.20.153.11 =>'\n" +
                        "  - 'host = 10.20.153.10,10.20.153.11 =>'\n" +
                        "  - 'application != kylin => host != 172.22.3.95,172.22.3.96'\n" +
                        "  - 'method = find*,list*,get*,is* => host = 172.22.3.94,172.22.3.95,172.22.3.96'";
    }

}