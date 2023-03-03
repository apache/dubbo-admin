package config

import (
	"flag"
	"os"
	"reflect"
	"strconv"
)

func StringVar(p *string, name string, value string, usage string) {
	flag.StringVar(p, name, value, usage)
	flag.Parse()
	if reflect.DeepEqual(p, &value) {
		val, ex := os.LookupEnv(name)
		if !ex {
			*p = value
		} else {
			*p = val
		}
	}
}

func IntVar(p *int, name string, value int, usage string) {
	flag.IntVar(p, name, value, usage)
	flag.Parse()
	if reflect.DeepEqual(p, &value) {
		val, ex := os.LookupEnv(name)
		if !ex {
			*p = value
		} else {
			num, err := strconv.Atoi(val)
			if err != nil {
				*p = num
			} else {
				*p = value
			}
		}
	}
}

func BoolVar(p *bool, name string, value bool, usage string) {
	flag.BoolVar(p, name, value, usage)
	flag.Parse()
	if reflect.DeepEqual(p, &value) {
		val, ex := os.LookupEnv(name)
		if !ex {
			*p = value
		} else {
			num, err := strconv.ParseBool(val)
			if err != nil {
				*p = num
			} else {
				*p = value
			}
		}
	}
}

func Int32Var(p *int32, name string, value int32, usage string) {
	var p64 int = int(*p)
	flag.IntVar(&p64, name, int(value), usage)
	flag.Parse()
	if reflect.DeepEqual(p64, &value) {
		val, ex := os.LookupEnv(name)
		if !ex {
			*p = int32(value)
		} else {
			num, err := strconv.Atoi(val)
			if err != nil {
				*p = int32(num)
			} else {
				*p = int32(value)
			}
		}
	} else {
		*p = int32(value)
	}
}
