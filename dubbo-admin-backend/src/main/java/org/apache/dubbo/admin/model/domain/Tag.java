package org.apache.dubbo.admin.model.domain;

/**
 * @author zmx ON 2018/11/28
 */
public class Tag {
    String name;
    String[] address;

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String[] getAddress() {
        return address;
    }

    public void setAddress(String[] address) {
        this.address = address;
    }
}
