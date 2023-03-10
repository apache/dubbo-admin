package util

import "dubbo.apache.org/dubbo-go/v3/common/yaml"

func DumpObject(obj interface{}) (string, error) {
	bytes, err := yaml.MarshalYML(obj)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func LoadObject(content string, obj interface{}) error {
	return yaml.UnmarshalYML([]byte(content), obj)
}
