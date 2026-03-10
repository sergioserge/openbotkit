package spectest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/provider"
)

// AssertJudge uses an LLM to evaluate whether the agent's response satisfies
// the given criteria. The criteria should describe what a correct response
// looks like (e.g., "mentions both email topics and WhatsApp messages from Alice").
func AssertJudge(t *testing.T, p provider.Provider, model string, prompt, response, criteria string) {
	t.Helper()

	judgePrompt := `You are a strict test evaluator. You will be given:
1. The user's original question
2. The AI assistant's response
3. Success criteria

The assistant had access to real databases (email, WhatsApp, memories) via tool calls.
The details in the response (names, dates, email addresses, subjects) come from those databases — they are NOT hallucinated.
Your job is ONLY to check whether the response covers the topics described in the criteria.

Respond with exactly one line: "PASS" or "FAIL"
Then on the next line, a brief explanation (1-2 sentences).

User question: ` + prompt + `

Assistant response:
` + response + `

Success criteria: ` + criteria

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := p.Chat(ctx, provider.ChatRequest{
		Model: model,
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, judgePrompt),
		},
		MaxTokens: 256,
	})
	if err != nil {
		t.Fatalf("judge LLM call failed: %v", err)
	}

	verdict := resp.TextContent()
	firstLine := strings.SplitN(strings.TrimSpace(verdict), "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	if !strings.EqualFold(firstLine, "PASS") {
		t.Errorf("judge FAIL for criteria %q\njudge said: %s\nagent response was:\n%s", criteria, verdict, response)
	}
}
