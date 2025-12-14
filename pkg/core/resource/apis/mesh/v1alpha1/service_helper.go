package v1alpha1

import "github.com/apache/dubbo-admin/pkg/common/constants"

// BuildServiceKey build service key
// {service}:{version}:{group}:{appName}
func BuildServiceKey(serviceName, version, group, appName string) string {
	return serviceName + constants.ColonSeparator + version +
		constants.ColonSeparator + group + constants.ColonSeparator + appName
}
