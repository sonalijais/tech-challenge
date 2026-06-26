package assistant

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/openai/openai-go/v2"
	"go.opentelemetry.io/otel"
)

var holidayTracer = otel.Tracer("holiday-tool")

type HolidayTool struct{}

func NewHolidayTool() *HolidayTool {
	return &HolidayTool{}
}

func (t *HolidayTool) Name() string {
	return "get_holidays"
}

func (t *HolidayTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
		openai.FunctionDefinitionParam{
			Name: "get_holidays",
			Description: openai.String(
				"Gets local bank and public holidays. Each line is a single holiday in the format 'YYYY-MM-DD: Holiday Name'.",
			),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"before_date": map[string]string{
						"type":        "string",
						"description": "Optional date in RFC3339 format to get holidays before this date.",
					},
					"after_date": map[string]string{
						"type":        "string",
						"description": "Optional date in RFC3339 format to get holidays after this date.",
					},
					"max_count": map[string]any{
						"type":        "integer",
						"description": "Optional maximum number of holidays to return.",
					},
				},
			},
		},
	)
}

func (t *HolidayTool) Execute(
	ctx context.Context,
	args string,
) (string, error) {
	ctx, span := holidayTracer.Start(ctx, "HolidayTool.Execute")
	defer span.End()

	var payload struct {
		BeforeDate time.Time `json:"before_date,omitempty"`
		AfterDate  time.Time `json:"after_date,omitempty"`
		MaxCount   int       `json:"max_count,omitempty"`
	}

	if err := json.Unmarshal(
		[]byte(args),
		&payload,
	); err != nil {
		return "", err
	}

	link := "https://www.officeholidays.com/ics/spain/catalonia"

	if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
		link = v
	}

	events, err := LoadCalendar(ctx, link)
	if err != nil {
		return "", err
	}

	var holidays []string

	for _, event := range events {
		date, err := event.GetAllDayStartAt()
		if err != nil {
			continue
		}

		if payload.MaxCount > 0 &&
			len(holidays) >= payload.MaxCount {
			break
		}

		if !payload.BeforeDate.IsZero() &&
			date.After(payload.BeforeDate) {
			continue
		}

		if !payload.AfterDate.IsZero() &&
			date.Before(payload.AfterDate) {
			continue
		}

		holidays = append(
			holidays,
			date.Format(time.DateOnly)+": "+
				event.GetProperty(
					ics.ComponentPropertySummary,
				).Value,
		)
	}

	if len(holidays) == 0 {
		return "No holidays found", nil
	}

	return strings.Join(holidays, "\n"), nil
}
