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

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Objects;

/**
 * copy from {@link org.apache.dubbo.metadata.definition.model.MethodDefinition} compatible 2.x version
 */
public class MethodDefinition {

    private String name;
    private String[] parameterTypes;
    private String returnType;
    private List<TypeDefinition> parameters;
    private List<String> annotations;

    public MethodDefinition() {
    }

    public String getName() {
        return this.name;
    }

    public List<TypeDefinition> getParameters() {
        if (this.parameters == null) {
            this.parameters = new ArrayList();
        }

        return this.parameters;
    }

    public String[] getParameterTypes() {
        return this.parameterTypes;
    }

    public String getReturnType() {
        return this.returnType;
    }

    public void setName(String name) {
        this.name = name;
    }

    public void setParameters(List<TypeDefinition> parameters) {
        this.parameters = parameters;
    }

    public void setParameterTypes(String[] parameterTypes) {
        this.parameterTypes = TypeDefinition.formatTypes(parameterTypes);
    }

    public void setReturnType(String returnType) {
        this.returnType = TypeDefinition.formatType(returnType);
    }

    public List<String> getAnnotations() {
        if (this.annotations == null) {
            this.annotations = Collections.emptyList();
        }

        return this.annotations;
    }

    public void setAnnotations(List<String> annotations) {
        this.annotations = annotations;
    }

    public String toString() {
        return "MethodDefinition [name=" + this.name + ", parameterTypes=" + Arrays.toString(this.parameterTypes) + ", returnType=" + this.returnType + "]";
    }

    public boolean equals(Object o) {
        if (this == o) {
            return true;
        } else if (!(o instanceof MethodDefinition)) {
            return false;
        } else {
            MethodDefinition that = (MethodDefinition) o;
            return Objects.equals(this.getName(), that.getName()) && Arrays.equals(this.getParameterTypes(), that.getParameterTypes()) && Objects.equals(this.getReturnType(), that.getReturnType()) && Objects.equals(this.getParameters(), that.getParameters());
        }
    }

    public int hashCode() {
        int result = Objects.hash(new Object[]{this.getName(), this.getReturnType(), this.getParameters()});
        result = 31 * result + Arrays.hashCode(this.getParameterTypes());
        return result;
    }
}
