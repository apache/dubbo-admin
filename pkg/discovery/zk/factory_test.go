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

	"github.com/stretchr/testify/require"
)

func TestZKConfigNameSupportsGroupedAndLegacyPaths(t *testing.T) {
	tests := []struct {
		path string
		name string
		ok   bool
	}{
		{"/dubbo/config/dubbo/provider.affinity-router", "provider.affinity-router", true},
		{"/dubbo/config/dubbo/provider.script-router", "provider.script-router", true},
		{"/dubbo/config/provider.tag-router", "provider.tag-router", true},
		{"/dubbo/config", "", false},
		{"/dubbo/config/dubbo", "", false},
		{"/dubbo/config/other/group", "", false},
	}
	for _, tt := range tests {
		name, ok := zkConfigName(tt.path)
		require.Equal(t, tt.ok, ok, tt.path)
		require.Equal(t, tt.name, name, tt.path)
	}
}
