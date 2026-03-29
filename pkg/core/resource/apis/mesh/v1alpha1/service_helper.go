package v1alpha1

import "github.com/apache/dubbo-admin/pkg/common/constants"

// BuildServiceKey builds a metadata key with app name.
// {service}:{version}:{group}:{appName}
func BuildServiceKey(serviceName, version, group, appName string) string {
	return serviceName + constants.ColonSeparator + version +
		constants.ColonSeparator + group + constants.ColonSeparator + appName
}

// BuildServiceIdentityKey builds the unique service identity key.
// {service}:{version}:{group}
func BuildServiceIdentityKey(serviceName, version, group string) string {
	return serviceName + constants.ColonSeparator + version + constants.ColonSeparator + group
}
