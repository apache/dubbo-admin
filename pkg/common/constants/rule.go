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

package constants

import set "github.com/duke-git/lancet/v2/datastructure/set"

var RuleSuffixSet = set.New(
	ConfiguratorsSuffix,
	ConditionRuleSuffix,
	TagRuleSuffix,
	AffinityRuleSuffix,
	ScriptRuleSuffix,
)

const (
	ConfiguratorVersionV3   = `v3.0`
	ConfiguratorVersionV3x1 = `v3.1`
	ConfigVersionKey        = `configVersion`
	ScopeApplication        = `application`
	ScopeService            = `service`
	SideProvider            = `provider`
	SideConsumer            = `consumer`
	// RuleConfigGroup is the group path used by Dubbo 3 dynamic rules in
	// ZooKeeper and the group name used by Nacos.
	RuleConfigGroup = `dubbo`

	ConfiguratorRuleDotSuffix = ".configurators"
	ConfiguratorsSuffix       = "configurators"
	ConditionRuleDotSuffix    = ".condition-router"
	ConditionRuleSuffix       = "condition-router"
	TagRuleDotSuffix          = ".tag-router"
	TagRuleSuffix             = "tag-router"
	AffinityRuleDotSuffix     = ".affinity-router"
	AffinityRuleSuffix        = "affinity-router"
	ScriptRuleDotSuffix       = ".script-router"
	ScriptRuleSuffix          = "script-router"
	ScriptTypeJavaScript      = "javascript"
	// MaxScriptRuleSize bounds the payload Admin accepts before publishing it
	// to a configuration center. The runtime compiles scripts, so an explicit
	// limit prevents accidentally publishing unbounded documents.
	MaxScriptRuleSize = 64 * 1024
)

const (
	NotEqual = "!="
	Equal    = "="
)

const (
	ServiceDefaultTimeout int32 = 1000
	DefaultWeight         int32 = 100
	ServiceDefaultRetries int32 = 2
)
