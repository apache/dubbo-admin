/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionRuleInputUsesVersionSpecificConditions(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		conditions string
		wantLegacy int
		wantV31    int
	}{
		{name: "v3.0", version: "v3.0", conditions: `["=> application=demo"]`, wantLegacy: 1},
		{name: "v3.1", version: "v3.1", conditions: `[{"from":{"match":"method=SayHello"},"to":[{"match":"application=demo","weight":0}]}]`, wantV31: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &ConditionRuleInput{
				ConfigVersion: tt.version,
				Conditions:    json.RawMessage(tt.conditions),
			}
			got, err := input.ToProto()
			require.NoError(t, err)
			assert.Len(t, got.Conditions, tt.wantLegacy)
			assert.Len(t, got.ConditionRules, tt.wantV31)
			if tt.wantV31 != 0 {
				assert.Zero(t, got.ConditionRules[0].To[0].Weight)
			}
		})
	}
}

func TestConditionRuleInputRejectsCrossVersionConditionShape(t *testing.T) {
	_, err := (&ConditionRuleInput{
		ConfigVersion: "v3.1",
		Conditions:    json.RawMessage(`["=> application=demo"]`),
	}).ToProto()
	assert.Error(t, err)
}
