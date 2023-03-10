package manifest

import (
	"io/fs"
	"strconv"
	"strings"
)

// StripPrefix removes the given prefix from prefix.
func StripPrefix(path, prefix string) string {
	pl := len(strings.Split(prefix, "/"))
	pv := strings.Split(path, "/")
	return strings.Join(pv[pl:], "/")
}

func SplitSetFlag(flag string) (string, string) {
	items := strings.Split(flag, "=")
	if len(items) != 2 {
		return flag, ""
	}
	return strings.TrimSpace(items[0]), strings.TrimSpace(items[1])
}

func GetFileNames(f fs.FS, root string) ([]string, error) {
	var fileNames []string
	if err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileNames = append(fileNames, path)
		return nil
	}); err != nil {
		return nil, err
	}
	return fileNames, nil
}

func GetValueFromSetFlags(setFlags []string, key string) string {
	var res string
	for _, setFlag := range setFlags {
		k, v := SplitSetFlag(setFlag)
		if k == key {
			res = v
		}
	}
	return res
}

// ParseValue parses string into a value
func ParseValue(valueStr string) interface{} {
	var value interface{}
	if v, err := strconv.Atoi(valueStr); err == nil {
		value = v
	} else if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
		value = v
	} else if v, err := strconv.ParseBool(valueStr); err == nil {
		value = v
	} else {
		value = strings.ReplaceAll(valueStr, "\\,", ",")
	}
	return value
}
