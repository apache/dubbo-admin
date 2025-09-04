package main

import (
	"dubbo-admin-ai/internal/agent"
	"dubbo-admin-ai/internal/config"
)

func main() {
	if err := agent.InitAgent(config.DEFAULT_MODEL.Key()); err != nil {
		panic(err)
	}
	select {}
}
