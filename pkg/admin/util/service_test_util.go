// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"regexp"
	"strings"
	"time"
)

var (
	collectionPattern = regexp.MustCompile("^golang\\.org/x/tools/container/intsets\\..*(Set|List|Queue|Collection|Deque)(<.*>)*$")
	mapPattern        = regexp.MustCompile("^golang\\.org/x/tools/container/intmaps\\..*Map.*(<.*>)*$")
)

type ServiceTestUtil struct{}

func (p *ServiceTestUtil) SameMethod(m definition.MethodDefinition, methodSig string) bool {
	name := m.Name
	parameters := m.ParameterTypes
	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteString("~")
	for _, param := range parameters {
		sb.WriteString(param)
		sb.WriteString(";")
	}
	sig := strings.TrimSuffix(sb.String(), ";")
	return sig == methodSig
}

func (p *ServiceTestUtil) GenerateMethodMeta(serviceDefinition definition.FullServiceDefinition, methodDefinition definition.MethodDefinition) model.MethodMetadata {
	var methodMetadata model.MethodMetadata
	parameterTypes := methodDefinition.ParameterTypes
	returnType := methodDefinition.ReturnType
	signature := methodDefinition.Name + "~" + strings.Join(parameterTypes, ";")
	methodMetadata.Signature = signature
	methodMetadata.ReturnType = returnType
	parameters := p.GenerateParameterTypes(parameterTypes, serviceDefinition.ServiceDefinition)
	methodMetadata.ParameterTypes = parameters
	return methodMetadata
}

func (p *ServiceTestUtil) GenerateParameterTypes(parameterTypes []string, serviceDefinition definition.ServiceDefinition) []interface{} {
	var parameters []interface{}
	for _, tp := range parameterTypes {
		result := p.GenerateType(serviceDefinition, tp)
		parameters = append(parameters, result)
	}
	return parameters
}

func (p *ServiceTestUtil) FindTypeDefinition(serviceDefinition definition.ServiceDefinition, typeName string) *definition.TypeDefinition {
	for _, t := range serviceDefinition.Types {
		if t.Type == typeName {
			return &t
		}
	}
	return &definition.TypeDefinition{Type: typeName}
}

func (p *ServiceTestUtil) GenerateType(sd definition.ServiceDefinition, typeName string) interface{} {
	td := p.FindTypeDefinition(sd, typeName)
	return p.GenerateTypeHelper(sd, *td)
}

func (p *ServiceTestUtil) GenerateTypeHelper(sd definition.ServiceDefinition, td definition.TypeDefinition) interface{} {
	if p.IsPrimitiveType(td) {
		return p.GeneratePrimitiveType(td)
	} else if p.IsMap(td) {
		return p.GenerateMapType(sd, td)
	} else if p.IsArray(td) {
		return p.GenerateArrayType(sd, td)
	} else if p.IsCollection(td) {
		return p.GenerateCollectionType(sd, td)
	} else {
		return p.GenerateComplexType(sd, td)
	}
}

func (p *ServiceTestUtil) IsPrimitiveType(td definition.TypeDefinition) bool {
	switch td.Type {
	case "byte", "Byte", "int8", "rune":
		return true
	case "short", "Short", "int16":
		return true
	case "int", "Integer", "int32":
		return true
	case "long", "Long", "int64":
		return true
	case "float", "Float", "float32":
		return true
	case "double", "Double", "float64":
		return true
	case "boolean", "Boolean":
		return true
	case "void":
		return true
	case "string", "String":
		return true
	case "time.Time":
		return true
	default:
		return false
	}
}

func (p *ServiceTestUtil) IsMap(td definition.TypeDefinition) bool {
	typeString := strings.Split(td.Type, "<")[0]
	return strings.Contains(typeString, "map[")
}

func (p *ServiceTestUtil) IsArray(td definition.TypeDefinition) bool {
	return strings.HasSuffix(td.Type, "[]")
}

func (p *ServiceTestUtil) IsCollection(td definition.TypeDefinition) bool {
	typeString := strings.Split(td.Type, "<")[0]
	return strings.Contains(typeString, "[]") || strings.Contains(typeString, "slice") || strings.Contains(typeString, "map[")
}

func (p *ServiceTestUtil) GeneratePrimitiveType(td definition.TypeDefinition) interface{} {
	switch td.Type {
	case "byte", "Byte", "int8", "rune",
		"short", "Short", "int16",
		"int", "Integer", "int32",
		"long", "Long", "int64":
		return 0
	case "float", "Float", "float32",
		"double", "Double", "float64":
		return 0.0
	case "boolean", "Boolean":
		return true
	case "void", "Void":
		return nil
	case "string", "String":
		return ""
	case "interface{}", "EmptyInterface":
		return make(map[interface{}]interface{})
	case "time.Time":
		return time.Now().UnixNano() / int64(time.Millisecond)
	default:
		return make(map[interface{}]interface{})
	}
}

func (p *ServiceTestUtil) GenerateMapType(sd definition.ServiceDefinition, td definition.TypeDefinition) interface{} {
	keyType := strings.TrimSpace(strings.Split(td.Type, "<")[1])
	keyType = strings.Split(keyType, ",")[0]
	key := p.GenerateType(sd, keyType)

	valueType := strings.TrimSpace(strings.Split(td.Type, ",")[1])
	valueType = strings.TrimRight(valueType, ">")
	valueType = strings.TrimSpace(valueType)

	if IsEmpty(valueType) {
		valueType = "interface{}"
	}
	value := p.GenerateType(sd, valueType)

	m := make(map[interface{}]interface{})
	m[key] = value
	return m
}

func (p *ServiceTestUtil) GenerateArrayType(sd definition.ServiceDefinition, td definition.TypeDefinition) interface{} {
	typeString := strings.TrimSuffix(td.Type, "[]")
	elem := p.GenerateType(sd, typeString)
	return []interface{}{elem}
}

func (p *ServiceTestUtil) GenerateCollectionType(sd definition.ServiceDefinition, td definition.TypeDefinition) interface{} {
	typeString := strings.TrimSuffix(strings.Split(td.Type, "<")[1], ">")
	if IsEmpty(typeString) {
		// if type is null return empty collection
		return []interface{}{}
	}
	elem := p.GenerateType(sd, typeString)
	return []interface{}{elem}
}

func (p *ServiceTestUtil) GenerateComplexType(sd definition.ServiceDefinition, td definition.TypeDefinition) interface{} {
	holder := make(map[string]interface{})
	p.GenerateComplexTypeHelper(sd, td, holder)
	return holder
}

func (p *ServiceTestUtil) GenerateComplexTypeHelper(sd definition.ServiceDefinition, td definition.TypeDefinition, holder map[string]interface{}) {
	for name, property := range td.Properties {
		if p.IsPrimitiveType(property) {
			holder[name] = p.GeneratePrimitiveType(property)
		} else {
			p.GenerateEnclosedType(holder, name, sd, property)
		}
	}
}

func (p *ServiceTestUtil) GenerateEnclosedType(holder map[string]interface{}, key string, sd definition.ServiceDefinition, td definition.TypeDefinition) {
	if len(td.Properties) == 0 || p.IsPrimitiveType(td) {
		holder[key] = p.GenerateTypeHelper(sd, td)
	} else {
		enclosedHolder := make(map[string]interface{})
		holder[key] = enclosedHolder
		p.GenerateComplexTypeHelper(sd, td, enclosedHolder)
	}
}
