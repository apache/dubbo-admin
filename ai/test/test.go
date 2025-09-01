package test

import "dubbo-admin-ai/internal/agent"

func init() {
	if err := agent.InitAgent(); err != nil {
		panic(err)
	}
}
