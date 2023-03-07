package util

import (
	"github.com/apache/dubbo-admin/pkg/admin/model/dto"
	"strings"
)

type Convert struct {
}

//func init(){
//	Convert := Convert{}
//}

func (c Convert) GetIdFromDTO(basedto interface{}) interface{} {

	baseDTO := basedto.(dto.BaseDTO)
	if baseDTO.GetApplication() != "" {
		return baseDTO.GetApplication()
	}
	// id format: "${class}:${version}:${group}"
	var builder strings.Builder
	builder.WriteString(baseDTO.GetService())
	builder.WriteString(":")
	builder.WriteString(null2EmptyString(&baseDTO.ServiceVersion))
	builder.WriteString(":")
	builder.WriteString(null2EmptyString(&baseDTO.ServiceGroup))
	return builder.String()
}
func null2EmptyString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
