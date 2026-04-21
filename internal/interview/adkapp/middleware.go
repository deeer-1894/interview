package adkapp

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	adkreduction "github.com/cloudwego/eino/adk/middlewares/reduction"
	adkskill "github.com/cloudwego/eino/adk/middlewares/skill"
	adksummarization "github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"mockinterview/internal/interview"
)

func buildAgentHandlers(ctx context.Context, llm model.BaseChatModel) ([]adk.ChatModelAgentMiddleware, error) {
	fileBackend := osFilesystemBackend{}

	skillBackend, err := adkskill.NewBackendFromFilesystem(ctx, &adkskill.BackendFromFilesystemConfig{
		Backend: fileBackend,
		BaseDir: interview.SkillsBaseDir(),
	})
	if err != nil {
		return nil, fmt.Errorf("build skill backend: %w", err)
	}

	skillHandler, err := adkskill.NewMiddleware(ctx, &adkskill.Config{
		Backend: interviewOnlySkillBackend{base: skillBackend},
	})
	if err != nil {
		return nil, fmt.Errorf("build skill middleware: %w", err)
	}

	reductionHandler, err := adkreduction.New(ctx, &adkreduction.Config{
		Backend:                   fileBackend,
		ReadFileToolName:          "read_file",
		RootDir:                   "/tmp/mockinterview-adk",
		MaxLengthForTrunc:         8 * 1024,
		MaxTokensForClear:         24 * 1024,
		ClearAtLeastTokens:        512,
		TokenCounter:              estimateTokenCount,
		SkipTruncation:            true,
		SkipClear:                 false,
		ClearRetentionSuffixLimit: 2,
	})
	if err != nil {
		return nil, err
	}

	summarizationHandler, err := adksummarization.New(ctx, &adksummarization.Config{
		Model: llm,
		Trigger: &adksummarization.TriggerCondition{
			ContextMessages: 18,
			ContextTokens:   12 * 1024,
		},
		TranscriptFilePath: "conversation://full-transcript",
	})
	if err != nil {
		return nil, err
	}

	return []adk.ChatModelAgentMiddleware{
		skillHandler,
		reductionHandler,
		summarizationHandler,
	}, nil
}

func estimateTokenCount(ctx context.Context, msgs []adk.Message, _ []*schema.ToolInfo) (int64, error) {
	var total int64
	for _, msg := range msgs {
		total += int64(len(msg.Content) / 4)
		if total == 0 && msg.Content != "" {
			total++
		}
	}
	return total, nil
}
