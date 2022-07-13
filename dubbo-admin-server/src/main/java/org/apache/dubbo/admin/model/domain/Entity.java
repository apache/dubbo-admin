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

import java.io.Serializable;
import java.util.Date;
import java.util.List;
import java.util.Objects;

/**
 * Entity
 */
public abstract class Entity implements Serializable {

    private static final long serialVersionUID = -3031128781434583143L;

    private List<Long> ids;

    private Long id;

    private String hash;

    private Date created;

    private Date modified;

    private Date now;

    private String operator;

    private String operatorAddress;

    private boolean miss;

    public Entity() {
    }

    public Entity(Long id) {
        this.id = id;
    }

    public List<Long> getIds() {
        return ids;
    }

    public void setIds(List<Long> ids) {
        this.ids = ids;
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getHash() {
        return hash;
    }

    public void setHash(String hash) {
        this.hash = hash;
    }

    public Date getCreated() {
        return created;
    }

    public void setCreated(Date created) {
        this.created = created;
    }

    public Date getModified() {
        return modified;
    }

    public void setModified(Date modified) {
        this.modified = modified;
    }

    public Date getNow() {
        return now;
    }

    public void setNow(Date now) {
        this.now = now;
    }

    public String getOperator() {
        return operator;
    }

    public void setOperator(String operator) {
        if (operator != null && operator.length() > 200) {
            operator = operator.substring(0, 200);
        }
        this.operator = operator;
    }

    public String getOperatorAddress() {
        return operatorAddress;
    }

    public void setOperatorAddress(String operatorAddress) {
        this.operatorAddress = operatorAddress;
    }

    public boolean isMiss() {
        return miss;
    }

    public void setMiss(boolean miss) {
        this.miss = miss;
    }

    @java.lang.Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (o == null || getClass() != o.getClass()) {
            return false;
        }
        Entity entity = (Entity) o;
        return miss == entity.miss && Objects.equals(ids, entity.ids) && Objects.equals(id, entity.id) && Objects.equals(hash, entity.hash) && Objects.equals(created, entity.created) && Objects.equals(modified, entity.modified) && Objects.equals(now, entity.now) && Objects.equals(operator, entity.operator) && Objects.equals(operatorAddress, entity.operatorAddress);
    }

    @java.lang.Override
    public int hashCode() {
        return Objects.hash(ids, id, hash, created, modified, now, operator, operatorAddress, miss);
    }
}
