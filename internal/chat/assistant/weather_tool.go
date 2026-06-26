package assistant

import (
	"context"
	"encoding/json"

	"github.com/acai-travel/tech-challenge/internal/weather"
	"github.com/openai/openai-go/v2"
	"go.opentelemetry.io/otel"
)

var weatherTracer = otel.Tracer("weather-tool")

type WeatherTool struct {
	client *weather.Client
}

func NewWeatherTool(client *weather.Client) *WeatherTool {
	return &WeatherTool{
		client: client,
	}
}

func (t *WeatherTool) Name() string {
	return "get_weather"
}

func (t *WeatherTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
		openai.FunctionDefinitionParam{
			Name:        "get_weather",
			Description: openai.String("Get current weather or forecast for a location"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]string{
						"type": "string",
					},
					"days": map[string]any{
						"type":        "integer",
						"description": "Optional forecast days between 1 and 10",
					},
				},
				"required": []string{"location"},
			},
		},
	)
}

func (t *WeatherTool) Execute(ctx context.Context, args string) (string, error) {
	ctx, span := weatherTracer.Start(ctx, "WeatherTool.Execute")
	defer span.End()

	var payload struct {
		Location string `json:"location"`
		Days     int    `json:"days,omitempty"`
	}

	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "", err
	}

	if payload.Days <= 1 {
		return t.client.Current(ctx, payload.Location)
	}

	return t.client.Forecast(
		ctx,
		payload.Location,
		payload.Days,
	)
}
