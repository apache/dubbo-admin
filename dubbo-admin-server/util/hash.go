package util

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5_16bit(input string) string {
	hash := Md5_32bit(input)
	return hash[8:24]
}

func Md5_32bit(input string) string {
	hash := md5.Sum([]byte(input))
	result := hex.EncodeToString(hash[:])
	return result
}
