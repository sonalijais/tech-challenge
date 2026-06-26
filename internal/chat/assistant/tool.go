package assistant

import (
	"context"

	"github.com/openai/openai-go/v2"
)

type Tool interface {
	Name() string
	Definition() openai.ChatCompletionToolUnionParam
	Execute(ctx context.Context, args string) (string, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry(tools ...Tool) *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	for _, tool := range tools {
		registry.tools[tool.Name()] = tool
	}

	return registry
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) Definitions() []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, 0, len(r.tools))

	for _, tool := range r.tools {
		result = append(result, tool.Definition())
	}

	return result
}
