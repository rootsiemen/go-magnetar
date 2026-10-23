package statplugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/go-units"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.stat", &Plugin{})
}

// Plugin registers the /stat chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "stat",
		Help: "Print context statistics (messages, tokens, bytes, models).",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			msgs := params.AgentHandler.Messages()
			cfg := params.AgentHandler.Config()
			stats := collectStats(
				msgs,
				cfg.ProfileParamString("llm.model"),
				cfg.String("rag.llm.model"),
				cfg.Int("rag.llm.vector_size"),
			)
			return &plugin.ChatCommandResponse{Stdout: formatStatsMarkdown(stats)}, nil
		},
	})
	return nil
}

// contextStats aggregates context usage numbers for rendering.
type contextStats struct {
	messages   int
	tokens     int
	bytes      int
	llmModel   string
	ragModel   string
	vectorSize int
}

// collectStats computes context statistics from the conversation history and
// the effective configuration. It performs no I/O.
func collectStats(msgs []openai.ChatCompletionMessage, llmModel, ragModel string, vectorSize int) contextStats {
	var totalBytes, totalTokens int
	for _, m := range msgs {
		b := len(m.Role) + len(m.Content)
		for _, tc := range m.ToolCalls {
			b += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
		totalBytes += b
		totalTokens += max(b/4, 1)
	}
	return contextStats{
		messages:   len(msgs),
		tokens:     totalTokens,
		bytes:      totalBytes,
		llmModel:   llmModel,
		ragModel:   ragModel,
		vectorSize: vectorSize,
	}
}

// formatStatsMarkdown renders context statistics as a Markdown table so the
// chat agent's standard rendering pipeline displays it like other commands.
func formatStatsMarkdown(s contextStats) string {
	var sb strings.Builder
	sb.WriteString("## Context statistics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|---|---|\n")
	fmt.Fprintf(&sb, "| Messages | %d |\n", s.messages)
	fmt.Fprintf(&sb, "| Tokens | ~%d (estimated) |\n", s.tokens)
	fmt.Fprintf(&sb, "| Bytes | %s |\n", units.HumanSize(float64(s.bytes)))
	fmt.Fprintf(&sb, "| LLM model | %s |\n", s.llmModel)
	fmt.Fprintf(&sb, "| RAG model | %s |\n", s.ragModel)
	fmt.Fprintf(&sb, "| Vector size | %d |\n", s.vectorSize)
	return sb.String()
}
