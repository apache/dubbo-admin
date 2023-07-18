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

package traffic

import (
	"strings"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func removeFromOverride(key, side, param string) error {
	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		return err
	}

	if oldRule == "" {
		return perrors.Errorf("Override rule does not exist!")
	}

	override := &model.Override{}
	err = yaml.Unmarshal([]byte(oldRule), override)
	if err != nil {
		logger.Error("Unrecognized override rule!")
		return err
	}
	for i, c := range override.Configs {
		if c.Side == side && c.Parameters[param] != "" {
			if len(c.Parameters) == 1 {
				override.Configs = append(override.Configs[:i], override.Configs[i+1:]...)
			} else {
				delete(c.Parameters, param)
			}
		}
	}

	if len(override.Configs) == 0 {
		err = config.Governance.DeleteConfig(key)
		if err != nil {
			logger.Error("Failed to delete override rule!")
			return err
		}
	} else {
		bytes, _ := yaml.Marshal(override)
		err = config.Governance.SetConfig(key, string(bytes))
		if err != nil {
			return err
		}
	}
	return nil
}

func createOrUpdateOverride(key string, side, param string, newRule model.Override) error {
	var mergedRule string
	newRuleByte, _ := yaml.Marshal(newRule)

	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No existing configuration found, will create a new one directly!")
			err := config.Governance.SetConfig(key, string(newRuleByte))
			if err != nil {
				logger.Errorf("Failed to save configuration status, please try again!", err)
				return err
			}
			return nil
		} else {
			logger.Errorf("Failed to check previous configuration status, please try again!", err)
		}
		return err
	}

	if oldRule != "" {
		//oJsonByte, err := yaml2.YAMLToJSON([]byte(oldRule))
		//if err != nil {
		//	logger.Errorf("Failed to convert yaml to json!", err)
		//}
		//nJsonByte, err := yaml2.YAMLToJSON(newRuleByte)
		//if err != nil {
		//	logger.Errorf("Failed to convert yaml to json!", err)
		//}
		//patch, err := jsonpatch.CreateMergePatch(nJsonByte, oJsonByte)
		//fmt.Printf(string(patch))
		//testConsumer := `[ { "op": "test", "path": "/side", "value": "consumer" } ]`
		//testTimeout := `[ { "op": "test", "path": "/parameters/timeout", "value": null } ]`
		//consumerPatch, _ := jsonpatch.DecodePatch(testConsumer)
		//timeoutPatch, _ := jsonpatch.DecodePatch(testTimeout)

		override := &model.Override{}
		_ = yaml.Unmarshal([]byte(oldRule), override)

		if param == "weight" {
			mergeWeight(override, side, param, newRule)
		} else {
			mergeOverride(override, side, param, newRule)
		}
		mergedRuleByte, err := yaml.Marshal(override)
		if err != nil {
			return err
		}
		mergedRule = string(mergedRuleByte)
	} else {
		mergedRule = string(newRuleByte)
	}

	err = config.Governance.SetConfig(key, mergedRule)
	if err != nil {
		logger.Errorf("Failed to save timeout yaml rule!", err)
	}
	return nil
}

// mergeOverride applies to keys like 'timeout', 'accesslog' and 'mock'
func mergeOverride(override *model.Override, side string, param string, newRule model.Override) {
	updated := false
	for _, c := range override.Configs {
		if c.Side == side && c.Parameters[param] != "" {
			c.Parameters[param] = newRule.Configs[0].Parameters[param]
			c.Enabled = newRule.Enabled
			updated = true
			break
		}
	}
	if !updated {
		override.Configs = append(override.Configs, newRule.Configs[0])
	}

	override.Enabled = newRule.Enabled
}

// mergeWeight applies to key 'weight'
func mergeWeight(override *model.Override, side string, param string, newRule model.Override) {
	for i, c := range override.Configs {
		if c.Side == side && c.Parameters[param] != "" {
			// todo, add warning promote
			override.Configs = append(override.Configs[:i], override.Configs[i+1:]...)
		}
	}
	override.Configs = append(override.Configs, newRule.Configs...)

	override.Enabled = newRule.Enabled
}

func getValue(rawRule, side, param string) (interface{}, error) {
	override := &model.Override{}
	err := yaml.Unmarshal([]byte(rawRule), override)
	if err != nil {
		return nil, err
	}
	for _, c := range override.Configs {
		if c.Side == side && c.Parameters[param] != nil {
			return c.Parameters[param], nil
		}
	}

	return nil, nil
}

func createOrUpdateCondition(key string, newRule model.ConditionRoute) error {
	var mergedRule string
	newRuleByte, _ := yaml.Marshal(newRule)

	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No existing configuration found, will create a new one directly!")
			err := config.Governance.SetConfig(key, string(newRuleByte))
			if err != nil {
				logger.Errorf("Failed to save configuration status, please try again!", err)
				return err
			}
			return nil
		} else {
			logger.Errorf("Failed to check previous configuration status, please try again!", err)
		}
		return err
	}

	if oldRule != "" {
		route := &model.ConditionRoute{}
		_ = yaml.Unmarshal([]byte(oldRule), route)

		exist := false
		for _, c := range route.Conditions {
			if c == newRule.Conditions[0] {
				exist = true
			}
		}
		if !exist {
			route.Conditions = append(route.Conditions, newRule.Conditions[0])
		}

		route.Force = newRule.Force
		route.Enabled = newRule.Enabled
		route.Runtime = newRule.Runtime
		mergedRuleByte, err := yaml.Marshal(route)
		if err != nil {
			return err
		}
		mergedRule = string(mergedRuleByte)
	} else {
		mergedRule = string(newRuleByte)
	}

	err = config.Governance.SetConfig(key, mergedRule)
	if err != nil {
		logger.Errorf("Failed to save region condition rule!", err)
	}
	return nil
}

func removeCondition(key, rule string, identifier string) error {
	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		return err
	}

	if oldRule == "" {
		return perrors.Errorf("Condition rule does not exist!")
	}

	route := &model.ConditionRoute{}
	err = yaml.Unmarshal([]byte(oldRule), route)
	if err != nil {
		logger.Error("Unrecognized condition rule!")
		return err
	}
	for i, c := range route.Conditions {
		if strings.Contains(c, identifier) {
			route.Conditions = append(route.Conditions[:i], route.Conditions[i+1:]...)
			break
		}
	}

	if len(route.Conditions) == 0 {
		err = config.Governance.DeleteConfig(key)
		if err != nil {
			logger.Error("Failed to delete override rule!")
			return err
		}
	} else {
		bytes, _ := yaml.Marshal(route)
		err = config.Governance.SetConfig(key, string(bytes))
		if err != nil {
			return err
		}
	}
	return nil
}

func createOrUpdateTag(key string, newRule model.TagRoute) error {
	var mergedRule string
	newRuleByte, _ := yaml.Marshal(newRule)

	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		if _, ok := err.(*config.RuleNotFound); ok {
			logger.Infof("No existing configuration found, will create a new one directly!")
			err := config.Governance.SetConfig(key, string(newRuleByte))
			if err != nil {
				logger.Errorf("Failed to save configuration status, please try again!", err)
				return err
			}
			return nil
		} else {
			logger.Errorf("Failed to check previous configuration status, please try again!", err)
		}
		return err
	}

	if oldRule != "" {
		logger.Warn("Will override the existing tag rule with the new one!")
	}
	mergedRule = string(newRuleByte)

	err = config.Governance.SetConfig(key, mergedRule)
	if err != nil {
		logger.Errorf("Failed to save region condition rule!", err)
	}
	return nil
}

func deleteTag(key string) error {
	oldRule, err := config.Governance.GetConfig(key)
	if err != nil {
		return err
	}

	if oldRule == "" {
		logger.Errorf("Tag rule does not exist!")
		return nil
	}

	return config.Governance.DeleteConfig(key)
}
