package util

import (
	"testing"
)

type TagRoute struct {
	Priority int
	Enable   bool
	Force    bool
	Runtime  bool
	Key      string
}

func TestDumpObject(t *testing.T) {
	tagRoute := TagRoute{
		Priority: 1,
		Enable:   true,
		Force:    true,
		Runtime:  true,
	}
	str, err := DumpObject(tagRoute)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}

func TestLoadObject(t *testing.T) {
	str := `priority: 0
enable: false
force: false
runtime: false
key: go1
tags:
- name: tag1
  addresses:
  - 192.168.0.1:20881
- name: tag2
  addresses:
  - 192.168.0.2:20882
`
	var tagRoute TagRoute
	LoadObject(str, &tagRoute)
	println(tagRoute.Key)
}
