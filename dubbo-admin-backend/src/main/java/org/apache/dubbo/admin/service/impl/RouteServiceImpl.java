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
package org.apache.dubbo.admin.service.impl;

import org.apache.dubbo.admin.common.exception.ResourceNotFoundException;
import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.common.util.YamlParser;
import org.apache.dubbo.admin.model.domain.Route;
import org.apache.dubbo.admin.model.dto.ConditionRouteDTO;
import org.apache.dubbo.admin.model.dto.TagRouteDTO;
import org.apache.dubbo.admin.service.RouteService;
import org.apache.dubbo.common.URL;
import org.apache.dubbo.common.utils.StringUtils;
import org.springframework.stereotype.Component;

/**
 * IbatisRouteService
 *
 */
@Component
public class RouteServiceImpl extends AbstractService implements RouteService {

    private String prefix = Constants.CONFIG_KEY;

    @Override
    public void createConditionRoute(ConditionRouteDTO conditionRoute) {
        conditionRoute = convertRouteDTOtoStore(conditionRoute);
        String path = getPath(conditionRoute.getKey(),Constants.CONDITION_ROUTE);
        //register2.7
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(conditionRoute));

        //register2.6
        if (conditionRoute.getScope().equals("service")) {
            Route old = convertRouteToOldRoute(conditionRoute);
            registry.register(old.toUrl());
        }

    }

    @Override
    public void updateConditionRoute(ConditionRouteDTO oldConditionRoute, ConditionRouteDTO newConditionRoute) {
        oldConditionRoute = convertRouteDTOtoStore(oldConditionRoute);
        newConditionRoute = convertRouteDTOtoStore(newConditionRoute);
        String path = getPath(newConditionRoute.getKey(), Constants.CONDITION_ROUTE);
        if (dynamicConfiguration.getConfig(path) == null) {
            throw new ResourceNotFoundException("no existing condition route for path: " + path);
        }
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(newConditionRoute));

        if (newConditionRoute.getScope().equals("service")) {
            Route old = convertRouteToOldRoute(oldConditionRoute);
            Route updated = convertRouteToOldRoute(newConditionRoute);
            registry.unregister(old.toUrl());
            registry.register(updated.toUrl());
        }
    }

    @Override
    public void deleteConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        dynamicConfiguration.deleteConfig(path);

        //for 2.6
        if (StringUtils.isNoneEmpty(config)) {
            ConditionRouteDTO route = YamlParser.loadObject(config, ConditionRouteDTO.class);
            if (route.getScope().equals("service")) {
                Route old = convertRouteToOldRoute(route);
                registry.unregister(old.toUrl());
            }
        }
    }


    @Override
    public void enableConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            ConditionRouteDTO conditionRoute = YamlParser.loadObject(config, ConditionRouteDTO.class);

            if (conditionRoute.getScope().equals("service")) {
                //for2.6
                URL oldURL = convertRouteToOldRoute(conditionRoute).toUrl();
                registry.unregister(oldURL);
                oldURL = oldURL.addParameter("enabled", true);
                registry.register(oldURL);
            }

            //2.7
            conditionRoute.setEnabled(true);
            dynamicConfiguration.setConfig(path, YamlParser.dumpObject(conditionRoute));
        }

    }

    @Override
    public void disableConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            ConditionRouteDTO conditionRoute = YamlParser.loadObject(config, ConditionRouteDTO.class);

            if (conditionRoute.getScope().equals("service")) {
                //for 2.6
                URL oldURL = convertRouteToOldRoute(conditionRoute).toUrl();
                registry.unregister(oldURL);
                oldURL = oldURL.addParameter("enabled", false);
                registry.register(oldURL);
            }

            //2.7
            conditionRoute.setEnabled(false);
            dynamicConfiguration.setConfig(path, YamlParser.dumpObject(conditionRoute));
        }

    }

    @Override
    public ConditionRouteDTO findConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return YamlParser.loadObject(config, ConditionRouteDTO.class);
        }
        return null;
    }

    @Override
    public void createTagRoute(TagRouteDTO tagRoute) {
        tagRoute = convertTagRouteDTOtoStore(tagRoute);
        String path = getPath(tagRoute.getKey(),Constants.TAG_ROUTE);
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(tagRoute));
    }

    @Override
    public void updateTagRoute(TagRouteDTO tagRoute) {
        tagRoute = convertTagRouteDTOtoStore(tagRoute);
        String path = getPath(tagRoute.getKey(), Constants.TAG_ROUTE);
        if (dynamicConfiguration.getConfig(path) == null) {
            throw new ResourceNotFoundException("can not find tagroute: " + tagRoute.getKey());
            //throw exception
        }
        dynamicConfiguration.setConfig(path, YamlParser.dumpObject(tagRoute));

    }

    @Override
    public void deleteTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        dynamicConfiguration.deleteConfig(path);
    }

    @Override
    public void enableTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            TagRouteDTO tagRoute = YamlParser.loadObject(config, TagRouteDTO.class);
            tagRoute.setEnabled(true);
            dynamicConfiguration.setConfig(path, YamlParser.dumpObject(tagRoute));
        }

    }

    @Override
    public void disableTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            TagRouteDTO tagRoute = YamlParser.loadObject(config, TagRouteDTO.class);
            tagRoute.setEnabled(false);
            dynamicConfiguration.setConfig(path, YamlParser.dumpObject(tagRoute));
        }

    }

    @Override
    public TagRouteDTO findTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return YamlParser.loadObject(config, TagRouteDTO.class);
        }
        return null;
    }

    private String getPath(String key, String type) {
        if (type.equals(Constants.CONDITION_ROUTE)) {
            return prefix + Constants.PATH_SEPARATOR + key + Constants.PATH_SEPARATOR + "routers";
        } else {
            return prefix + Constants.PATH_SEPARATOR + key + Constants.PATH_SEPARATOR + "tagrouters";
        }
    }

    private String parseCondition(String[] conditions) {
        StringBuilder when = new StringBuilder();
        StringBuilder then = new StringBuilder();
        for (String condition : conditions) {
            condition = condition.trim();
            if (condition.contains("=>")) {
                String[] array = condition.split("=>", 2);
                String consumer = array[0].trim();
                String provider = array[1].trim();
                if (consumer != "") {
                    if (when.length() != 0) {
                        when.append(" & ").append(consumer);
                    } else {
                        when.append(consumer);
                    }
                }
                if (provider != "") {
                    if (then.length() != 0) {
                        then.append(" & ").append(provider);
                    } else {
                        then.append(provider);
                    }
                }
            }
        }
        return (when.append(" => ").append(then)).toString();
    }

    private Route convertRouteToOldRoute(ConditionRouteDTO route) {
        Route old = new Route();
        old.setService(route.getKey());
        old.setEnabled(route.isEnabled());
        old.setForce(route.isForce());
        old.setRuntime(route.isRuntime());
        old.setPriority(route.getPriority());
        String rule = parseCondition(route.getConditions());
        old.setRule(rule);
        return old;
    }

    private ConditionRouteDTO convertRouteDTOtoStore(ConditionRouteDTO conditionRoute) {
        if (StringUtils.isNoneEmpty(conditionRoute.getApplication())) {
            conditionRoute.setScope("application");
            conditionRoute.setKey(conditionRoute.getApplication());
            conditionRoute.setApplication(null);
        } else {
            conditionRoute.setScope("service");
            conditionRoute.setKey(conditionRoute.getService());
            conditionRoute.setService(null);
        }
        return conditionRoute;
    }

    private TagRouteDTO convertTagRouteDTOtoStore(TagRouteDTO tagRouteDTO) {
        tagRouteDTO.setKey(tagRouteDTO.getApplication());
        tagRouteDTO.setScope("application");
        tagRouteDTO.setApplication(null);
        return tagRouteDTO;
    }
}
