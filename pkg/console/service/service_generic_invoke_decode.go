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
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"

	javabig "github.com/dubbogo/gost/math/big"

	"github.com/apache/dubbo-admin/pkg/console/model"
)

type argumentDecoder struct {
	types      map[string]model.ServiceMethodTypeResp
	goProvider bool
}

func decodeGenericInvokeArgs(detail *model.ServiceMethodDetailResp, args []json.RawMessage, goProvider bool) ([]any, error) {
	if len(detail.ParameterTypes) != len(args) {
		return nil, fmt.Errorf("args: expected %d parameters for %s, got %d", len(detail.ParameterTypes), detail.Signature, len(args))
	}
	d := argumentDecoder{types: make(map[string]model.ServiceMethodTypeResp, len(detail.Types)), goProvider: goProvider}
	for _, def := range detail.Types {
		d.types[def.Type] = def
	}
	result := make([]any, len(args))
	for i, raw := range args {
		path := fmt.Sprintf("arg%d", i)
		if i < len(detail.Parameters) && detail.Parameters[i].Name != "" {
			path = detail.Parameters[i].Name
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var value any
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		arg, err := d.decode(path, detail.ParameterTypes[i], value)
		if err != nil {
			return nil, err
		}
		result[i] = arg
	}
	return result, nil
}

func argumentError(path, typ string, value any, reason string) error {
	actual := "null"
	switch value.(type) {
	case json.Number:
		actual = "number"
	case string:
		actual = "string"
	case bool:
		actual = "boolean"
	case []any:
		actual = "array"
	case map[string]any:
		actual = "object"
	}
	return fmt.Errorf("%s: expected %s, got %s (%s)", path, typ, actual, reason)
}

// The sample values carry both the scalar width and the slice element type used by Hessian.
var scalarSamples = map[string]any{
	"byte": int8(0), "Byte": int8(0), "int8": int8(0),
	"short": int16(0), "Short": int16(0), "int16": int16(0),
	"int": int32(0), "Integer": int32(0), "int32": int32(0), "rune": int32(0),
	"long": int64(0), "Long": int64(0), "int64": int64(0),
	"uint": uint(0), "uint8": uint8(0), "uint16": uint16(0), "uint32": uint32(0), "uint64": uint64(0),
	"float": float32(0), "Float": float32(0), "float32": float32(0),
	"double": float64(0), "Double": float64(0), "float64": float64(0),
	"char": uint16(0), "Character": uint16(0),
	"bool": false, "boolean": false, "Boolean": false, "string": "", "String": "",
}

func (d argumentDecoder) scalarType(typ string) reflect.Type {
	if d.goProvider {
		switch typ {
		case "int":
			return reflect.TypeFor[int]()
		case "byte":
			return reflect.TypeFor[byte]()
		}
	}
	return reflect.TypeOf(scalarSamples[strings.TrimPrefix(typ, "java.lang.")])
}

// Preserve integral decimal/exponent inputs accepted by the debug API, including beyond 2^53.
func integerText(text string) (string, error) {
	if !strings.ContainsAny(text, ".eE") {
		return text, nil
	}
	if !json.Valid([]byte(text)) {
		return "", fmt.Errorf("invalid decimal integer")
	}
	// An exponent larger than the input length plus the wire width cannot fit, even
	// after shifting all the significant digits. Avoid expanding it into a huge integer.
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		exp, err := strconv.Atoi(text[i+1:])
		if err != nil || exp > i+20 || exp < -i-20 {
			return "", fmt.Errorf("exponent outside integer range")
		}
	}
	n, ok := new(big.Rat).SetString(text)
	if !ok || !n.IsInt() || n.Num().BitLen() > 64 {
		return "", fmt.Errorf("expected an integer within the wire range")
	}
	return n.Num().String(), nil
}

func decodeScalar(typ string, target reflect.Type, value any) (any, error) {
	if reflect.TypeOf(value) == target {
		return value, nil
	}
	var text string
	switch v := value.(type) {
	case json.Number:
		text = v.String()
	case string:
		text = strings.TrimSpace(v)
	default:
		return nil, fmt.Errorf("incorrect scalar type")
	}
	if typ == "char" || typ == "Character" || typ == "java.lang.Character" {
		if s, ok := value.(string); ok {
			r := []rune(s)
			if len(r) != 1 || r[0] > math.MaxUint16 {
				return nil, fmt.Errorf("expected one UTF-16 code unit")
			}
			return uint16(r[0]), nil
		}
	}
	out := reflect.New(target).Elem()
	if target.Kind() >= reflect.Int && target.Kind() <= reflect.Uint64 {
		var err error
		text, err = integerText(text)
		if err != nil {
			return nil, err
		}
	}
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 10, target.Bits())
		if err != nil {
			return nil, err
		}
		out.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(text, 10, target.Bits())
		if err != nil {
			return nil, err
		}
		if n > math.MaxInt64 {
			return nil, fmt.Errorf("exceeds signed 64-bit wire range")
		}
		out.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(text, target.Bits())
		if err != nil {
			return nil, err
		}
		if math.IsInf(n, 0) || math.IsNaN(n) {
			return nil, fmt.Errorf("expected finite number")
		}
		// Reject a whole-number result if rounding would change the user's integer or erase a fraction.
		if math.Trunc(n) == n {
			exact, ok := new(big.Rat).SetString(text)
			if !ok || exact.Cmp(new(big.Rat).SetFloat64(n)) != 0 {
				return nil, fmt.Errorf("value loses precision as %s", typ)
			}
		}
		out.SetFloat(n)
	default:
		return nil, fmt.Errorf("incorrect scalar type")
	}
	return out.Interface(), nil
}

// Container item types normally come from metadata. Go also embeds them in the type name.
func containerTypes(typ string, items []string) (key, elem string) {
	switch {
	case strings.HasPrefix(typ, "["):
		_, elem, _ = strings.Cut(typ, "]")
		return "", elem
	case strings.HasSuffix(typ, "[]"):
		return "", typ[:len(typ)-2]
	case strings.HasPrefix(typ, "map["):
		key, elem, _ = strings.Cut(typ[4:], "]")
		return key, elem
	case len(items) == 2:
		return items[0], items[1]
	case len(items) == 1:
		return "", items[0]
	case typ == "java.util.Map" || typ == "java.util.HashMap":
		return "string", "any"
	case typ == "java.util.List" || typ == "java.util.Collection":
		return "", "any"
	}
	return "", ""
}

func (d argumentDecoder) decode(path, typ string, value any) (any, error) {
	typ = strings.TrimSpace(typ)
	if target := d.scalarType(typ); target != nil {
		if value == nil && (strings.HasPrefix(typ, "java.lang.") || typ[0] >= 'A' && typ[0] <= 'Z') {
			return nil, nil
		}
		v, err := decodeScalar(typ, target, value)
		if err != nil {
			return nil, argumentError(path, typ, value, err.Error())
		}
		return v, nil
	}
	if value == nil {
		return nil, nil
	}
	def, declared := d.types[typ]
	if inner, ok := strings.CutPrefix(typ, "*"); ok && len(def.Properties)+len(def.Items)+len(def.Enums) == 0 {
		return d.decode(path, inner, value)
	}
	if typ == "java.math.BigInteger" {
		switch value.(type) {
		case string, json.Number:
		default:
			return nil, argumentError(path, typ, value, "expected decimal integer")
		}
		v := new(javabig.Integer)
		if err := v.FromString(fmt.Sprint(value)); err != nil {
			return nil, argumentError(path, typ, value, err.Error())
		}
		return v, nil
	}
	if len(def.Enums) > 0 {
		if s, ok := value.(string); ok && slices.Contains(def.Enums, s) {
			return s, nil
		}
		return nil, argumentError(path, typ, value, "allowed values: "+strings.Join(def.Enums, ", "))
	}
	keyType, elem := containerTypes(typ, def.Items)
	switch typ {
	case "any", "interface{}", "interface {}", "java.lang.Object", "Object":
		switch v := value.(type) {
		case map[string]any:
			keyType, elem = "string", "any"
		case []any:
			elem = "any"
		case json.Number:
			// Dynamic maps may carry a number only when its wire value is exact.
			if text, err := integerText(v.String()); err == nil {
				if n, err := strconv.ParseInt(text, 10, 64); err == nil {
					return n, nil
				}
			}
			if n, err := v.Float64(); err == nil && math.Trunc(n) != n {
				exact, ok := new(big.Rat).SetString(v.String())
				if ok && exact.Cmp(new(big.Rat).SetFloat64(n)) == 0 {
					return n, nil
				}
			}
			return nil, argumentError(path, typ, value, "numeric type metadata required")
		default:
			return v, nil
		}
	}
	if elem != "" && keyType == "" {
		return d.decodeArray(path, typ, elem, value)
	}
	if !declared && elem == "" {
		return nil, argumentError(path, typ, value, "type metadata unavailable")
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, argumentError(path, typ, value, "object required")
	}
	if elem != "" {
		return d.decodeMap(path, keyType, elem, fields)
	}
	for name, fieldType := range def.Properties {
		if _, found := fields[name]; !found {
			return nil, fmt.Errorf("%s.%s: expected %s, got missing (required field)", path, name, fieldType)
		}
	}
	for name, v := range fields {
		fieldPath := path + "." + name
		fieldType, known := def.Properties[name]
		if !known {
			return nil, argumentError(fieldPath, "declared field", v, "unknown field")
		}
		item, err := d.decode(fieldPath, fieldType, v)
		if err != nil {
			return nil, err
		}
		fields[name] = item
	}
	return fields, nil
}

func (d argumentDecoder) decodeArray(path, typ, elem string, value any) (any, error) {
	values, ok := value.([]any)
	if !ok {
		return nil, argumentError(path, typ, value, "array required")
	}
	if strings.HasPrefix(typ, "[") && !strings.HasPrefix(typ, "[]") {
		length, _, _ := strings.Cut(typ[1:], "]")
		n, err := strconv.Atoi(length)
		if err != nil || n != len(values) {
			return nil, argumentError(path, typ, value, fmt.Sprintf("got %d elements", len(values)))
		}
	}
	for i, v := range values {
		item, err := d.decode(fmt.Sprintf("%s[%d]", path, i), elem, v)
		if err != nil {
			return nil, err
		}
		values[i] = item
	}
	// Keep primitive arrays typed on the wire. Boxed arrays may contain nulls.
	if t := d.scalarType(elem); t != nil && !slices.ContainsFunc(values, func(v any) bool { return v == nil }) {
		array := reflect.MakeSlice(reflect.SliceOf(t), len(values), len(values))
		for i, item := range values {
			array.Index(i).Set(reflect.ValueOf(item))
		}
		return array.Interface(), nil
	}
	return values, nil
}

func (d argumentDecoder) decodeMap(path, keyType, elem string, fields map[string]any) (any, error) {
	keysAreStrings := d.scalarType(keyType) == reflect.TypeFor[string]()
	var typed map[any]any
	if !keysAreStrings {
		typed = make(map[any]any, len(fields))
	}
	for key, v := range fields {
		fieldPath := fmt.Sprintf("%s[%q]", path, key)
		item, err := d.decode(fieldPath, elem, v)
		if err != nil {
			return nil, err
		}
		if keysAreStrings {
			fields[key] = item
			continue
		}
		var rawKey any = key
		if d.scalarType(keyType) == reflect.TypeFor[bool]() && (key == "true" || key == "false") {
			rawKey = key == "true"
		}
		k, err := d.decode(fieldPath, keyType, rawKey)
		if err != nil {
			return nil, err
		}
		if k == nil || !reflect.TypeOf(k).Comparable() {
			return nil, argumentError(fieldPath, keyType, key, "invalid map key")
		}
		if _, exists := typed[k]; exists {
			return nil, argumentError(fieldPath, keyType, key, "duplicate converted map key")
		}
		typed[k] = item
	}
	if keysAreStrings {
		return fields, nil
	}
	return typed, nil
}
