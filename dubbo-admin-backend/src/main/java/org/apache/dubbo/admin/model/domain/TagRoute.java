package org.apache.dubbo.admin.model.domain;


public class TagRoute extends Route{
    private Tag[] tags;


    public Tag[] getTags() {
        return tags;
    }

    public void setTags(Tag[] tags) {
        this.tags = tags;
    }
}
