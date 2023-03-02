package config

import (
	"os"
	"strconv"
)

// 获取环境变量信息
func GetEnvDefault(key, defVal string) string {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	return val
}

// 获取环境变量信息
func GetEnvIntDefault(key string, defVal int) int {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return defVal
	}
	return num
}

// 获取环境变量信息
func GetEnvBoolDefault(key string, defVal bool) bool {
	val, ex := os.LookupEnv(key)
	if !ex {
		return defVal
	}
	res, err := strconv.ParseBool(val)
	if err != nil {
		return defVal
	}
	return res
}
