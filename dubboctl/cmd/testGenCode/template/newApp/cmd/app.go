package main

import (
	_ "dubbo-go-app/pkg/service"

	_ "dubbo.apache.org.apache.org/dubbo.apache.org-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/config"
)

// export DUBBO_GO_CONFIG_PATH=$PATH_TO_APP/conf/dubbogo.yaml
func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}
	select {}
}
