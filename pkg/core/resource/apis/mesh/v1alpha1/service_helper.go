package v1alpha1

import "github.com/apache/dubbo-admin/pkg/common/constants"

func BuildServiceKey(serviceName, version, group string) string {
	return serviceName + constants.ColonSeparator + version + constants.ColonSeparator + group
}
