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

package org.apache.dubbo.admin.model.domain;

import org.apache.dubbo.common.utils.StringUtils;

import com.google.gson.annotations.SerializedName;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;


/**
 * copy from {@link org.apache.dubbo.metadata.definition.model.TypeDefinition} compatible 2.x version
 */
public class TypeDefinition implements Serializable {
    private String id;
    private String type;
    @SerializedName("items")
    private List<TypeDefinition> items;
    @SerializedName("enum")
    private List<String> enums;
    private String $ref;
    private Map<String, TypeDefinition> properties;
    private String typeBuilderName;

    public TypeDefinition() {
    }

    public TypeDefinition(String type) {
        this.setType(type);
    }

    public static String[] formatTypes(String[] types) {
        String[] newTypes = new String[types.length];

        for (int i = 0; i < types.length; ++i) {
            newTypes[i] = formatType(types[i]);
        }

        return newTypes;
    }

    public static String formatType(String type) {
        return isGenericType(type) ? formatGenericType(type) : type;
    }

    private static String formatGenericType(String type) {
        return StringUtils.replace(type, ", ", ",");
    }

    private static boolean isGenericType(String type) {
        return type.contains("<") && type.contains(">");
    }

    public String get$ref() {
        return this.$ref;
    }

    public List<String> getEnums() {
        if (this.enums == null) {
            this.enums = new ArrayList();
        }

        return this.enums;
    }

    public String getId() {
        return this.id;
    }

    public List<TypeDefinition> getItems() {
        if (this.items == null) {
            this.items = new ArrayList();
        }

        return this.items;
    }

    public Map<String, TypeDefinition> getProperties() {
        if (this.properties == null) {
            this.properties = new HashMap();
        }

        return this.properties;
    }

    public String getType() {
        return this.type;
    }

    public String getTypeBuilderName() {
        return this.typeBuilderName;
    }

    public void set$ref(String $ref) {
        this.$ref = $ref;
    }

    public void setEnums(List<String> enums) {
        this.enums = enums;
    }

    public void setId(String id) {
        this.id = id;
    }

    public void setItems(List<TypeDefinition> items) {
        this.items = items;
    }

    public void setProperties(Map<String, TypeDefinition> properties) {
        this.properties = properties;
    }

    public void setType(String type) {
        this.type = formatType(type);
    }

    public void setTypeBuilderName(String typeBuilderName) {
        this.typeBuilderName = typeBuilderName;
    }

    public String toString() {
        return "TypeDefinition [id=" + this.id + ", type=" + this.type + ", properties=" + this.properties + ", $ref=" + this.$ref + "]";
    }

    public boolean equals(Object o) {
        if (this == o) {
            return true;
        } else if (!(o instanceof TypeDefinition)) {
            return false;
        } else {
            TypeDefinition that = (TypeDefinition) o;
            return Objects.equals(this.getId(), that.getId()) && Objects.equals(this.getType(), that.getType()) && Objects.equals(this.getItems(), that.getItems()) && Objects.equals(this.getEnums(), that.getEnums()) && Objects.equals(this.get$ref(), that.get$ref()) && Objects.equals(this.getProperties(), that.getProperties());
        }
    }

    public int hashCode() {
        return Objects.hash(new Object[]{this.getId(), this.getType(), this.getItems(), this.getEnums(), this.get$ref(), this.getProperties()});
    }
}
