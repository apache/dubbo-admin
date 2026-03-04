package test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"dubbo-admin-ai/component/agent/react"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema_description:"Location to get weather for"`
}

func defineWeatherFlow(g *genkit.Genkit) *core.Flow[WeatherInput, string, struct{}] {
	getWeatherTool := genkit.DefineTool(g, "getWeather", "Gets the current weather in a given location",
		func(ctx *ai.ToolContext, input WeatherInput) (string, error) {
			// Here, we would typically make an API call or database query. For this
			// example, we just return a fixed value.
			log.Printf("Tool 'getWeather' called for location: %s", input.Location)
			return fmt.Sprintf("The current weather in %s is 63°F and sunny.", input.Location), nil
		})

	return genkit.DefineFlow(g, "getWeatherFlow",
		func(ctx context.Context, location WeatherInput) (string, error) {
			resp, err := genkit.Generate(ctx, g,
				ai.WithTools(getWeatherTool),
				ai.WithPrompt("What's the weather in %s?", location.Location),
			)
			if err != nil {
				return "", err
			}
			return resp.Text(), nil
		})
}

// initTestRegistry 创建测试用的 registry
func initTestRegistry() *genkit.Genkit {
	ctx := context.Background()
	g := genkit.Init(ctx)
	return g
}

func TestTextGeneration(t *testing.T) {
	g := initTestRegistry()

	// 创建 ReAct Agent（使用默认配置）
	stagesCfg := react.ReActDefaultSpec().Stages
	_, _ = react.Create(g, "./prompts", stagesCfg, nil)
	ctx := context.Background()

	resp, err := genkit.GenerateText(ctx, g, ai.WithPrompt("Hello, Who are you?"))
	if err != nil {
		t.Fatalf("failed to generate text: %v", err)
	}
	t.Logf("Generated text: %s", resp)

	fmt.Printf("%s", resp)
}

func TestWeatherFlowRun(t *testing.T) {
	g := initTestRegistry()

	// 创建 ReAct Agent（使用默认配置）
	stagesCfg := react.ReActDefaultSpec().Stages
	_, _ = react.Create(g, "./prompts", stagesCfg, nil)
	ctx := context.Background()

	flow := defineWeatherFlow(g)
	flow.Run(ctx, WeatherInput{Location: "San Francisco"})
}
