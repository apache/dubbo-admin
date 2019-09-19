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
package org.apache.dubbo.admin.model.dto;

import java.util.Arrays;
import java.util.List;
import java.util.Objects;

/**
 * relation about node for relation graph
 */
public class RelationDTO {

    private List<Categories> categories;
    private List<Node> nodes;
    private List<Link> links;

    public static final Categories CONSUMER_CATEGORIES = new RelationDTO.Categories(0, "consumer", "consumer");;
    public static final Categories PROVIDER_CATEGORIES = new RelationDTO.Categories(1, "provider", "provider");
    public static final Categories CONSUMER_AND_PROVIDER_CATEGORIES = new RelationDTO.Categories(2, "consumer and provider", "consumer and provider");

    public static final List<RelationDTO.Categories> CATEGORIES_LIST = Arrays.asList(CONSUMER_CATEGORIES, PROVIDER_CATEGORIES, CONSUMER_AND_PROVIDER_CATEGORIES);

    public RelationDTO() {
    }

    public RelationDTO(List<Node> nodes, List<Link> links) {
        this.categories = CATEGORIES_LIST;
        this.nodes = nodes;
        this.links = links;
    }

    public static class Categories {
        private Integer index;
        private String name;
        private String base;

        public Categories() {
        }

        public Categories(Integer index, String name, String base) {
            this.index = index;
            this.name = name;
            this.base = base;
        }

        public Integer getIndex() {
            return index;
        }

        public void setIndex(Integer index) {
            this.index = index;
        }

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }

        public String getBase() {
            return base;
        }

        public void setBase(String base) {
            this.base = base;
        }
    }

    public static class Node {

        private Integer index;
        private String name;
        private int category;

        public Node() {
        }

        public Node(Integer index, String name, int category) {
            this.index = index;
            this.name = name;
            this.category = category;
        }

        public Integer getIndex() {
            return index;
        }

        public void setIndex(Integer index) {
            this.index = index;
        }

        public String getName() {
            return name;
        }

        public void setName(String name) {
            this.name = name;
        }

        public int getCategory() {
            return category;
        }

        public void setCategory(int category) {
            this.category = category;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            Node node = (Node) o;
            return category == node.category &&
                    index.equals(node.index) &&
                    name.equals(node.name);
        }

        @Override
        public int hashCode() {
            return Objects.hash(index, name, category);
        }
    }

    public static class Link {

        private int source;
        private int target;

        public Link() {
        }

        public Link(int source, int target) {
            this.source = source;
            this.target = target;
        }

        public int getSource() {
            return source;
        }

        public void setSource(int source) {
            this.source = source;
        }

        public int getTarget() {
            return target;
        }

        public void setTarget(int target) {
            this.target = target;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            Link link = (Link) o;
            return source == link.source &&
                    target == link.target;
        }

        @Override
        public int hashCode() {
            return Objects.hash(source, target);
        }

        @Override
        public String toString() {
            return "Link{" +
                    "source=" + source +
                    ", target=" + target +
                    '}';
        }
    }

    public List<Categories> getCategories() {
        return categories;
    }

    public void setCategories(List<Categories> categories) {
        this.categories = categories;
    }

    public List<Node> getNodes() {
        return nodes;
    }

    public void setNodes(List<Node> nodes) {
        this.nodes = nodes;
    }

    public List<Link> getLinks() {
        return links;
    }

    public void setLinks(List<Link> links) {
        this.links = links;
    }
}
