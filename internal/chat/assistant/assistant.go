package assistant

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/weather"
	"github.com/openai/openai-go/v2"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("assistant")

type Assistant struct {
	cli   openai.Client
	tools *ToolRegistry
}

func New() *Assistant {
	weatherClient := &weather.Client{
		APIKey: os.Getenv("WEATHER_API_KEY"),
	}

	registry := NewToolRegistry(
		NewWeatherTool(weatherClient),
		NewDateTool(),
		NewHolidayTool(),
		NewDayTool(),
	)

	return &Assistant{
		cli:   openai.NewClient(),
		tools: registry,
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	ctx, span := tracer.Start(ctx, "Assistant.Title")
	defer span.End()

	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	// only the first user message is used when generating a conversation title.
	msgs := make([]openai.ChatCompletionMessageParamUnion, 2)

	msgs[0] = openai.AssistantMessage(
		"Generate a concise title for the conversation based on the user's first message. " +
			"The title should describe the user's request, not the assistant's response. " +
			"Preserve the user's wording where appropriate (for example, do not replace 'day' with 'date' unless the user explicitly asks for the date). " +
			"The title must be a single line, no more than 80 characters, must not answer the question, and should not include special characters or emojis. " +
			"Return only the title.",
	)

	msgs[1] = openai.UserMessage(conv.Messages[0].Content)

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4_1,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	ctx, span := tracer.Start(ctx, "Assistant.Reply")
	defer span.End()

	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    a.tools.Definitions(),
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				tool, ok := a.tools.Get(call.Function.Name)
				if !ok {
					return "", errors.New("unknown tool call: " + call.Function.Name)
				}

				result, err := tool.Execute(
					ctx,
					call.Function.Arguments,
				)

				if err != nil {
					msgs = append(
						msgs,
						openai.ToolMessage(
							"tool execution failed: "+err.Error(),
							call.ID,
						),
					)
					continue
				}

				msgs = append(
					msgs,
					openai.ToolMessage(
						result,
						call.ID,
					),
				)
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
