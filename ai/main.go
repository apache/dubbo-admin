package main

import (
	"dubbo-admin-ai/internal/agent"
)

func main() {
	if err := agent.InitAgent(); err != nil {
		panic(err)
	}
	select {}
}
