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
package org.apache.dubbo.admin.controller.beans;


/**
 * api parameter bean.
 */
public class DubboApiDocsParamInfoBean {

    private String fieldName;

    private String fieldJavaType;

    private String methodParamType;

    private int methodParamIndex;

    private Object fieldValue;

    public DubboApiDocsParamInfoBean(String fieldName,String fieldJavaType,String methodParamType,int methodParamIndex) {
        this.fieldName = fieldName;
        this.fieldJavaType = fieldJavaType;
        this.methodParamType = methodParamType;
        this.methodParamIndex = methodParamIndex;
    }

    public String getFieldName() {
        return fieldName;
    }

    public void setFieldName(String fieldName) {
        this.fieldName = fieldName;
    }

    public String getFieldJavaType() {
        return fieldJavaType;
    }

    public void setFieldJavaType(String fieldJavaType) {
        this.fieldJavaType = fieldJavaType;
    }

    public String getMethodParamType() {
        return methodParamType;
    }

    public void setMethodParamType(String methodParamType) {
        this.methodParamType = methodParamType;
    }

    public int getMethodParamIndex() {
        return methodParamIndex;
    }

    public void setMethodParamIndex(int methodParamIndex) {
        this.methodParamIndex = methodParamIndex;
    }

    public Object getFieldValue() {
        return fieldValue;
    }

    public void setFieldValue(Object fieldValue) {
        this.fieldValue = fieldValue;
    }

    @Override
    public String toString() {
        return "DubboApiDocsParamInfoBean{" +
                "fieldName='" + fieldName + '\'' +
                ", fieldJavaType='" + fieldJavaType + '\'' +
                ", methodParamType='" + methodParamType + '\'' +
                ", methodParamIndex=" + methodParamIndex +
                ", paramValue=" + fieldValue +
                '}';
    }
}
