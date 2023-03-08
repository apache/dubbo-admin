package application

const (
	appGoFile = `package main

import (
	_ "dubbo.apache.org-go-app/pkg/service"

	"dubbo.apache.org.apache.org/dubbo.apache.org-go/v3/config"
	_ "dubbo.apache.org.apache.org/dubbo.apache.org-go/v3/imports"
)

// export DUBBO_GO_CONFIG_PATH=$PATH_TO_APP/conf/dubbogo.yaml
func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}
	select {}
}

`
)

func init() {
	fileMap["appGoFile"] = &fileGenerator{
		path:    "./cmd",
		file:    "app.go",
		context: appGoFile,
	}
}
