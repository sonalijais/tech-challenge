package assistant

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
	"go.opentelemetry.io/otel"
)

var dayTracer = otel.Tracer("day-tool")

type DayTool struct{}

func NewDayTool() *DayTool {
	return &DayTool{}
}

func (d *DayTool) Name() string {
	return "get_day"
}
func (t *DayTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
		openai.FunctionDefinitionParam{
			Name:        "get_day",
			Description: openai.String("Get today's week day"),
		},
	)
}

func (d *DayTool) Execute(ctx context.Context, args string) (string, error) {
	ctx, span := dayTracer.Start(ctx, "DayTool.Execute")
	defer span.End()

	return time.Now().Weekday().String(), nil
}
