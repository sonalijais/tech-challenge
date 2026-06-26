package assistant

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
	"go.opentelemetry.io/otel"
)

var dateTracer = otel.Tracer("date-tool")

type DateTool struct{}

func NewDateTool() *DateTool {
	return &DateTool{}
}

func (t *DateTool) Name() string {
	return "get_today_date"
}

func (t *DateTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
		openai.FunctionDefinitionParam{
			Name:        "get_today_date",
			Description: openai.String("Get today's date and time in RFC3339 format"),
		},
	)
}

func (t *DateTool) Execute(ctx context.Context, args string) (string, error) {
	ctx, span := dateTracer.Start(ctx, "DateTool.Execute")
	defer span.End()

	return time.Now().Format(time.RFC3339), nil
}
