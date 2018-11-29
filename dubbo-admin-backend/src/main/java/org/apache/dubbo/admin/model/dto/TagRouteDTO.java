package org.apache.dubbo.admin.model.dto;

import org.apache.dubbo.admin.model.domain.Tag;

public class TagRouteDTO extends RouteDTO{
    private Tag[] tags;

    public Tag[] getTags() {
        return tags;
    }

    public void setTags(Tag[] tags) {
        this.tags = tags;
    }

}
