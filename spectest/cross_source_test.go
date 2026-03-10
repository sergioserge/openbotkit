package spectest

import (
	"context"
	"testing"
	"time"
)

// TestSpec_SummarizeCommunicationsAcrossSources seeds emails and WhatsApp
// messages from the same person, then asks the agent to summarize all
// communications. The agent must discover and use both email-read and
// whatsapp-read skills, query both databases, and synthesize the results.
func TestSpec_SummarizeCommunicationsAcrossSources(t *testing.T) {
	fx := NewLocalFixture(t)

	fx.GivenEmails(t, []Email{
		{From: "alice@acme.com", To: "me@example.com", Subject: "Q3 Budget Review", Body: "Hi, please review the Q3 budget spreadsheet I shared. We need to finalize numbers by Friday."},
		{From: "alice@acme.com", To: "me@example.com", Subject: "Team Offsite Planning", Body: "I'm thinking we do the offsite in Portland in October. Thoughts?"},
	})

	fx.GivenWhatsAppMessages(t, []WhatsAppMessage{
		{SenderJID: "alice@s.whatsapp.net", SenderName: "Alice", ChatJID: "alice@s.whatsapp.net", ChatName: "Alice", Text: "Hey, did you see my email about the Q3 budget? Let me know if the numbers look right."},
		{SenderJID: "alice@s.whatsapp.net", SenderName: "Alice", ChatJID: "alice@s.whatsapp.net", ChatName: "Alice", Text: "Also I booked the restaurant for Friday dinner. Italian place downtown."},
		{SenderJID: "me@s.whatsapp.net", SenderName: "Me", ChatJID: "alice@s.whatsapp.net", ChatName: "Alice", Text: "Sounds great! I'll review the budget tonight.", IsFromMe: true},
	})

	a := fx.Agent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	prompt := "Summarize all my recent communications with Alice across email and WhatsApp."
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("agent.Run: %v", err)
	}

	AssertNotEmpty(t, result)
	AssertJudge(t, fx.Provider, fx.Model, prompt, result,
		"The response must cover topics from BOTH email and WhatsApp. "+
			"It should mention the Q3 budget review from email AND the restaurant/dinner plans from WhatsApp. "+
			"It should not claim that data from one source is missing if it was provided.")
}

// TestSpec_RecallMemoryAndCorrelateEmails seeds personal memories about a
// relationship and emails with project details. The agent must check memories
// for context about the person, then search emails for specifics, and combine
// both into a coherent answer.
func TestSpec_RecallMemoryAndCorrelateEmails(t *testing.T) {
	fx := NewLocalFixture(t)

	fx.GivenMemories(t, []UserMemory{
		{Content: "Alice Chen is my project manager at Acme Corp", Category: "relationship"},
		{Content: "The Horizon project has a hard deadline of March 30, 2025", Category: "project"},
		{Content: "Alice prefers async communication over meetings", Category: "relationship"},
	})

	fx.GivenEmails(t, []Email{
		{From: "alice.chen@acme.com", To: "me@example.com", Subject: "Horizon Milestone 3 Update", Body: "Milestone 3 is due March 15. We need the API integration tests complete by then. Please prioritize this."},
		{From: "alice.chen@acme.com", To: "me@example.com", Subject: "Re: Horizon Timeline", Body: "The client moved the final demo to March 28. We have two days less than planned."},
		{From: "bob@acme.com", To: "me@example.com", Subject: "Unrelated standup notes", Body: "Here are the standup notes from today."},
	})

	a := fx.Agent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	prompt := "What do I know about Alice and the Horizon project? What are the upcoming deadlines?"
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("agent.Run: %v", err)
	}

	AssertNotEmpty(t, result)
	AssertJudge(t, fx.Provider, fx.Model, prompt, result,
		"The response must include information from BOTH memories and emails. "+
			"It should mention Alice is the project manager (from memory) AND reference specific deadlines "+
			"like March 15 for Milestone 3 or March 28 for the demo (from emails). "+
			"It should not only use one source.")
}
