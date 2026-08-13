/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package model

import (
	"encoding/json"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
)

type RouterRuleSearchResp struct {
	CreateTime string `json:"createTime"`
	Enabled    bool   `json:"enabled"`
	RuleName   string `json:"ruleName"`
	Scope      string `json:"scope"`
}

type ConditionRuleInput struct {
	ConfigVersion string          `json:"configVersion"`
	Priority      int32           `json:"priority"`
	Enabled       bool            `json:"enabled"`
	Force         bool            `json:"force"`
	Runtime       bool            `json:"runtime"`
	Key           string          `json:"key"`
	Scope         string          `json:"scope"`
	Conditions    json.RawMessage `json:"conditions"`
}

func (i *ConditionRuleInput) ToProto() (*meshproto.ConditionRoute, error) {
	result := &meshproto.ConditionRoute{
		ConfigVersion: i.ConfigVersion, Priority: i.Priority, Enabled: i.Enabled,
		Force: i.Force, Runtime: i.Runtime, Key: i.Key, Scope: i.Scope,
	}
	if i.ConfigVersion == constants.ConfiguratorVersionV3x1 {
		if err := json.Unmarshal(i.Conditions, &result.ConditionRules); err != nil {
			return nil, err
		}
	} else if err := json.Unmarshal(i.Conditions, &result.Conditions); err != nil {
		return nil, err
	}
	return result, nil
}

func GenConditionRuleToResp(data *meshproto.ConditionRoute) *CommonResp {
	if data == nil {
		return NewSuccessResp(nil)
	}
	conditions := any(data.Conditions)
	if data.ConfigVersion == constants.ConfiguratorVersionV3x1 {
		conditions = data.ConditionRules
	}
	return NewSuccessResp(struct {
		ConfigVersion string `json:"configVersion"`
		Priority      int32  `json:"priority"`
		Enabled       bool   `json:"enabled"`
		Force         bool   `json:"force"`
		Runtime       bool   `json:"runtime"`
		Key           string `json:"key"`
		Scope         string `json:"scope"`
		Conditions    any    `json:"conditions"`
	}{
		ConfigVersion: data.ConfigVersion,
		Priority:      data.Priority,
		Enabled:       data.Enabled,
		Force:         data.Force,
		Runtime:       data.Runtime,
		Key:           data.Key,
		Scope:         data.Scope,
		Conditions:    conditions,
	})
}

type AffinityRuleInput struct {
	ConfigVersion string                   `json:"configVersion"`
	Scope         string                   `json:"scope"`
	Key           string                   `json:"key"`
	Runtime       bool                     `json:"runtime"`
	Enabled       bool                     `json:"enabled"`
	AffinityAware *meshproto.AffinityAware `json:"affinityAware"`
}

func (i *AffinityRuleInput) ToProto() *meshproto.AffinityRoute {
	return &meshproto.AffinityRoute{ConfigVersion: i.ConfigVersion, Scope: i.Scope, Key: i.Key, Runtime: i.Runtime, Enabled: i.Enabled, Affinity: i.AffinityAware}
}

func GenAffinityRuleResp(data *meshproto.AffinityRoute) *CommonResp {
	if data == nil {
		return NewSuccessResp(nil)
	}
	return NewSuccessResp(AffinityRuleInput{
		ConfigVersion: data.ConfigVersion,
		Scope:         data.Scope,
		Key:           data.Key,
		Runtime:       data.Runtime,
		Enabled:       data.Enabled,
		AffinityAware: data.Affinity,
	})
}
