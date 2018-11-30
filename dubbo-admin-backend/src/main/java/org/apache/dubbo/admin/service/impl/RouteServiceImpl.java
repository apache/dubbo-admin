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

import org.apache.dubbo.admin.common.util.Constants;
import org.apache.dubbo.admin.model.domain.ConditionRoute;
import org.apache.dubbo.admin.model.domain.TagRoute;
import org.apache.dubbo.admin.service.RouteService;
import org.springframework.stereotype.Component;
import org.yaml.snakeyaml.Yaml;

/**
 * IbatisRouteService
 *
 */
@Component
public class RouteServiceImpl extends AbstractService implements RouteService {

    private String prefix = Constants.CONFIG_KEY;
    Yaml yaml = new Yaml();

    @Override
    public void createConditionRoute(ConditionRoute conditionRoute) {
        String path = getPath(conditionRoute.getKey(),Constants.CONDITION_ROUTE);
        dynamicConfiguration.setConfig(path, yaml.dumpAsMap(conditionRoute));
    }

    @Override
    public void updateConditionRoute(ConditionRoute conditionRoute) {
        String path = getPath(conditionRoute.getKey(), Constants.CONDITION_ROUTE);
        if (dynamicConfiguration.getConfig(path) == null) {
           //throw exception
        }
        dynamicConfiguration.setConfig(path, yaml.dumpAsMap(conditionRoute));

    }

    @Override
    public void deleteConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        dynamicConfiguration.deleteConfig(path);

    }


    @Override
    public void enableConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            ConditionRoute conditionRoute = yaml.loadAs(config, ConditionRoute.class);
            conditionRoute.setEnabled(true);
            dynamicConfiguration.setConfig(path, yaml.dumpAsMap(conditionRoute));
        }
    }

    @Override
    public void disableConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            ConditionRoute conditionRoute = yaml.loadAs(config, ConditionRoute.class);
            conditionRoute.setEnabled(false);
            dynamicConfiguration.setConfig(path, yaml.dumpAsMap(conditionRoute));
        }

    }

    @Override
    public ConditionRoute findConditionRoute(String serviceName) {
        String path = getPath(serviceName, Constants.CONDITION_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return yaml.loadAs(config, ConditionRoute.class);
        }
        return null;
    }

    @Override
    public void createTagRoute(TagRoute tagRoute) {
        String path = getPath(tagRoute.getKey(),Constants.TAG_ROUTE);
        dynamicConfiguration.setConfig(path, yaml.dumpAsMap(tagRoute));
    }

    @Override
    public void updateTagRoute(TagRoute tagRoute) {
        String path = getPath(tagRoute.getKey(), Constants.TAG_ROUTE);
        if (dynamicConfiguration.getConfig(path) == null) {
            //throw exception
        }
        dynamicConfiguration.setConfig(path, yaml.dumpAsMap(tagRoute));

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
            TagRoute tagRoute = yaml.loadAs(config, TagRoute.class);
            tagRoute.setEnabled(true);
            dynamicConfiguration.setConfig(path, yaml.dumpAsMap(tagRoute));
        }

    }

    @Override
    public void disableTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            TagRoute tagRoute = yaml.loadAs(config, TagRoute.class);
            tagRoute.setEnabled(false);
            dynamicConfiguration.setConfig(path, yaml.dumpAsMap(tagRoute));
        }

    }

    @Override
    public TagRoute findTagRoute(String id) {
        String path = getPath(id, Constants.TAG_ROUTE);
        String config = dynamicConfiguration.getConfig(path);
        if (config != null) {
            return yaml.loadAs(config, TagRoute.class);
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
    @Override
    public ConditionRoute getBlackwhitelistRouteByService(String service) {
        return null;
    }


}
