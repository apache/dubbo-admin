/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package zk

import (
	"testing"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func TestRuleConfigPathsPreserveExistingRuleContracts(t *testing.T) {
	paths := []struct {
		name string
		res  coremodel.Resource
		want string
	}{
		{"affinity", meshresource.NewAffinityRouteResourceWithAttributes("provider.affinity-router", "mesh"), "/dubbo/config/dubbo/provider.affinity-router"},
		{"script", meshresource.NewScriptRouteResourceWithAttributes("provider.script-router", "mesh"), "/dubbo/config/dubbo/provider.script-router"},
		{"condition", meshresource.NewConditionRouteResourceWithAttributes("provider.condition-router", "mesh"), "/dubbo/config/provider.condition-router"},
	}
	for _, tt := range paths {
		if got := ruleConfigPath(tt.res); got != tt.want {
			t.Fatalf("%s ruleConfigPath() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
