package org.apache.dubbo.admin.controller.beans;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.Setter;

/**
 * api parameter bean.
 *
 * @author klw(213539 @ qq.com)
 * @date 2020/11/10 9:36
 */
@Setter
@Getter
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
