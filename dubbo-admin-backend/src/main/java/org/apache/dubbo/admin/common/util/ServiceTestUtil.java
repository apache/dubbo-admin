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

import org.apache.dubbo.admin.model.domain.MethodMetadata;
import org.apache.dubbo.admin.model.dto.MethodDTO;
import org.apache.dubbo.metadata.definition.model.FullServiceDefinition;
import org.apache.dubbo.metadata.definition.model.MethodDefinition;
import org.apache.dubbo.metadata.definition.model.ServiceDefinition;
import org.apache.dubbo.metadata.definition.model.TypeDefinition;

import java.util.*;
import java.util.stream.Collectors;

public class ServiceTestUtil {

    public static boolean sameMethod(MethodDefinition m, MethodDTO methodDTO) {
        return (m.getName().equals(methodDTO.getName())
                && m.getReturnType().equals(methodDTO.getReturnType())
                && m.getParameterTypes().equals(methodDTO.getParameterTypes().toArray()));
    }

    public static MethodMetadata generateMethodMeta(FullServiceDefinition serviceDefinition, MethodDefinition methodDefinition) {
        MethodMetadata methodMetadata = new MethodMetadata();
        String[] parameterTypes = methodDefinition.getParameterTypes();
        String returnType = methodDefinition.getReturnType();
        String signature = methodDefinition.getName() + "~" + Arrays.stream(parameterTypes).collect(Collectors.joining(";"));
        methodMetadata.setSignature(signature);
        methodMetadata.setReturnType(returnType);
        List parameters = generateParameterTypes(parameterTypes, serviceDefinition);
        methodMetadata.setParameterTypes(parameters);
        return methodMetadata;
    }

    private static boolean isPrimitiveType(String type) {
        return type.equals("byte") || type.equals("java.lang.Byte") ||
                type.equals("short") || type.equals("java.lang.Short") ||
                type.equals("int") || type.equals("java.lang.Integer") ||
                type.equals("long") || type.equals("java.lang.Long") ||
                type.equals("float") || type.equals("java.lang.Float") ||
                type.equals("double") || type.equals("java.lang.Double") ||
                type.equals("boolean") || type.equals("java.lang.Boolean") ||
                type.equals("void") || type.equals("java.lang.Void") ||
                type.equals("java.lang.String") ||
                type.equals("java.util.Date") ||
                type.equals("java.lang.Object");
    }

    private static List generateParameterTypes(String[] parameterTypes, ServiceDefinition serviceDefinition) {
        List parameters = new ArrayList();
        for (String type : parameterTypes) {
            if (isPrimitiveType(type)) {
                generatePrimitiveType(parameters, type);
            } else {
                TypeDefinition typeDefinition = findTypeDefinition(serviceDefinition, type);
                Map<String, Object> holder = new HashMap<>();
                generateComplexType(holder, typeDefinition);
                parameters.add(holder);
            }
        }
        return parameters;
    }

    private static TypeDefinition findTypeDefinition(ServiceDefinition serviceDefinition, String type) {
        return serviceDefinition.getTypes().stream()
                .filter(t -> t.getType().equals(type))
                .findFirst().orElse(new TypeDefinition(type));
    }

    private static void generateComplexType(Map<String, Object> holder, TypeDefinition td) {
        for (Map.Entry<String, TypeDefinition> entry : td.getProperties().entrySet()) {
            String type = entry.getValue().getType();
            if (isPrimitiveType(type)) {
                holder.put(entry.getKey(), type);
            } else {
                generateEnclosedType(holder, entry.getKey(), entry.getValue());
            }
        }
    }

    private static void generatePrimitiveType(List parameters, String type) {
        parameters.add(type);
    }

    private static void generateEnclosedType(Map<String, Object> holder, String key, TypeDefinition typeDefinition) {
        if (typeDefinition.getProperties() == null || typeDefinition.getProperties().size() == 0) {
            holder.put(key, typeDefinition.getType());
        } else {
            Map<String, Object> enclosedMap = new HashMap<>();
            holder.put(key, enclosedMap);
            generateComplexType(enclosedMap, typeDefinition);
        }
    }
}
