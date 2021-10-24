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

package org.apache.dubbo.admin.common.util;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.yaml.snakeyaml.Yaml;
import org.yaml.snakeyaml.constructor.SafeConstructor;
import org.yaml.snakeyaml.error.YAMLException;
import org.yaml.snakeyaml.introspector.Property;
import org.yaml.snakeyaml.nodes.NodeTuple;
import org.yaml.snakeyaml.nodes.Tag;
import org.yaml.snakeyaml.representer.Representer;

import java.util.Map;

public class YamlParser {

    private static final ObjectMapper OBJECT_MAPPER = new ObjectMapper();

    public static String dumpObject(Object object) {
        return new Yaml(new SafeConstructor(), new CustomRepresenter()).dumpAsMap(object);
    }

    public static <T> T loadObject(String content, Class<T> type) {
        Map<String, Object> map = new Yaml(new SafeConstructor(), new CustomRepresenter()).load(content);
        try {
            return OBJECT_MAPPER.convertValue(map, type);
        } catch (Exception e) {
            throw new YAMLException(e);
        }
    }

    public static Iterable<Object> loadAll(String content) {
        return new Yaml(new SafeConstructor(), new CustomRepresenter()).loadAll(content);
    }

    public static class CustomRepresenter extends Representer {

        protected NodeTuple representJavaBeanProperty(Object javaBean, Property property, Object propertyValue, Tag customTag) {
            if (propertyValue == null) {
                return null;
            } else {
                return super.representJavaBeanProperty(javaBean, property, propertyValue, customTag);
            }
        }
    }
}
