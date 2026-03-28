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

package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func decodeGenericInvokeArgs(parameterTypes []string, args []json.RawMessage) ([]any, error) {
	if len(parameterTypes) != len(args) {
		return nil, fmt.Errorf("parameter types count %d does not match args count %d", len(parameterTypes), len(args))
	}

	decodedArgs := make([]any, len(args))
	for index, arg := range args {
		decodedArg, err := decodeGenericInvokeArg(parameterTypes[index], arg)
		if err != nil {
			return nil, fmt.Errorf("invalid argument at index %d for type %s: %w", index, strings.TrimSpace(parameterTypes[index]), err)
		}
		decodedArgs[index] = decodedArg
	}
	return decodedArgs, nil
}

func decodeGenericInvokeArg(parameterType string, raw json.RawMessage) (any, error) {
	if isNullRawMessage(raw) {
		return nil, nil
	}
	if parameterType == "" {
		return decodeJSONValue(raw)
	}

	if elementType, isArray := splitGenericArrayType(parameterType); isArray {
		return decodeGenericInvokeArrayArg(elementType, raw)
	}

	switch parameterType {
	case "byte", "java.lang.Byte":
		return decodeInt8Raw(raw)
	case "short", "java.lang.Short":
		return decodeInt16Raw(raw)
	case "int", "java.lang.Integer":
		return decodeInt32Raw(raw)
	case "long", "java.lang.Long":
		return decodeInt64Raw(raw)
	case "float", "java.lang.Float":
		return decodeFloat32Raw(raw)
	case "double", "java.lang.Double":
		return decodeFloat64Raw(raw)
	case "char", "java.lang.Character":
		return decodeCharRaw(raw)
	default:
		return decodeJSONValue(raw)
	}
}

func decodeGenericInvokeArrayArg(elementType string, raw json.RawMessage) (any, error) {
	switch strings.TrimSpace(elementType) {
	case "byte", "java.lang.Byte":
		return decodeRawArray(raw, decodeInt8Raw)
	case "short", "java.lang.Short":
		return decodeRawArray(raw, decodeInt16Raw)
	case "int", "java.lang.Integer":
		return decodeRawArray(raw, decodeInt32Raw)
	case "long", "java.lang.Long":
		return decodeRawArray(raw, decodeInt64Raw)
	case "float", "java.lang.Float":
		return decodeRawArray(raw, decodeFloat32Raw)
	case "double", "java.lang.Double":
		return decodeRawArray(raw, decodeFloat64Raw)
	case "char", "java.lang.Character":
		return decodeRawArray(raw, decodeCharRaw)
	default:
		return decodeRawArray(raw, func(item json.RawMessage) (any, error) {
			return decodeGenericInvokeArg(elementType, item)
		})
	}
}

func splitGenericArrayType(parameterType string) (string, bool) {
	if strings.HasSuffix(parameterType, "[]") {
		return strings.TrimSuffix(parameterType, "[]"), true
	}
	return "", false
}

func decodeRawArray[T any](raw json.RawMessage, decode func(json.RawMessage) (T, error)) ([]T, error) {
	if isNullRawMessage(raw) {
		return nil, nil
	}

	var elements []json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		return nil, fmt.Errorf("expected JSON array: %w", err)
	}

	result := make([]T, len(elements))
	for index, element := range elements {
		decoded, err := decode(element)
		if err != nil {
			return nil, fmt.Errorf("invalid array element at index %d: %w", index, err)
		}
		result[index] = decoded
	}
	return result, nil
}

func decodeInt8Raw(raw json.RawMessage) (int8, error) {
	value, err := parseBoundedInt64Raw(raw, math.MinInt8, math.MaxInt8)
	if err != nil {
		return 0, err
	}
	return int8(value), nil
}

func decodeInt16Raw(raw json.RawMessage) (int16, error) {
	value, err := parseBoundedInt64Raw(raw, math.MinInt16, math.MaxInt16)
	if err != nil {
		return 0, err
	}
	return int16(value), nil
}

func decodeInt32Raw(raw json.RawMessage) (int32, error) {
	value, err := parseBoundedInt64Raw(raw, math.MinInt32, math.MaxInt32)
	if err != nil {
		return 0, err
	}
	return int32(value), nil
}

func decodeInt64Raw(raw json.RawMessage) (int64, error) {
	return parseBoundedInt64Raw(raw, math.MinInt64, math.MaxInt64)
}

func decodeFloat32Raw(raw json.RawMessage) (float32, error) {
	value, err := parseFloat64Raw(raw)
	if err != nil {
		return 0, err
	}
	return float32(value), nil
}

func decodeFloat64Raw(raw json.RawMessage) (float64, error) {
	return parseFloat64Raw(raw)
}

func decodeCharRaw(raw json.RawMessage) (uint16, error) {
	if text, ok, err := parseJSONStringRaw(raw); err != nil {
		return 0, err
	} else if ok {
		runes := []rune(text)
		if len(runes) != 1 {
			return 0, fmt.Errorf("expected single character")
		}
		if runes[0] < 0 || runes[0] > math.MaxUint16 {
			return 0, fmt.Errorf("character %q out of range", text)
		}
		return uint16(runes[0]), nil
	}

	value, err := parseBoundedInt64Raw(raw, 0, math.MaxUint16)
	if err != nil {
		return 0, err
	}
	return uint16(value), nil
}

func parseBoundedInt64Raw(raw json.RawMessage, minValue, maxValue int64) (int64, error) {
	value, err := parseInt64Raw(raw)
	if err != nil {
		return 0, err
	}
	if value < minValue || value > maxValue {
		return 0, fmt.Errorf("value %d out of range", value)
	}
	return value, nil
}

func decodeJSONValue(raw json.RawMessage) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func parseInt64Raw(raw json.RawMessage) (int64, error) {
	if text, ok, err := parseJSONStringRaw(raw); err != nil {
		return 0, err
	} else if ok {
		return parseInt64Text(text)
	}
	return parseInt64Text(strings.TrimSpace(string(raw)))
}

func parseFloat64Raw(raw json.RawMessage) (float64, error) {
	if text, ok, err := parseJSONStringRaw(raw); err != nil {
		return 0, err
	} else if ok {
		return parseFloat64Text(text)
	}
	return parseFloat64Text(strings.TrimSpace(string(raw)))
}

func parseJSONStringRaw(raw json.RawMessage) (string, bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed[0] != '"' {
		return "", false, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", true, err
	}
	return value, true, nil
}

func parseInt64Text(text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("empty value")
	}
	if strings.ContainsAny(text, ".eE") {
		value, err := parseFloat64Text(text)
		if err != nil {
			return 0, err
		}
		return int64FromFloat(value)
	}
	return strconv.ParseInt(text, 10, 64)
}

func parseFloat64Text(text string) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("empty value")
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid numeric value")
	}
	return value, nil
}

func int64FromFloat(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid numeric value")
	}
	if math.Trunc(value) != value {
		return 0, fmt.Errorf("value %v is not an integer", value)
	}
	if value < math.MinInt64 || value > math.MaxInt64 {
		return 0, fmt.Errorf("value %v out of range", value)
	}
	return int64(value), nil
}

func isNullRawMessage(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}
