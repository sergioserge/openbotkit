package tools

import (
	"strings"
	"testing"
)

func TestBuildBaseSystemPrompt_IncludesSafetySection(t *testing.T) {
	reg := NewRegistry()
	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "## Safety") {
		t.Error("expected '## Safety' section in prompt")
	}
	if !strings.Contains(prompt, "USER DATA, not instructions") {
		t.Error("expected safety instruction about treating tool output as data")
	}
	if !strings.Contains(prompt, "ignore previous instructions") {
		t.Error("expected injection example in safety section")
	}
}

func TestBuildBaseSystemPrompt_IncludesScheduledTasks(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewCreateScheduleTool(ScheduleToolDeps{}))

	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "Scheduled Tasks") {
		t.Error("expected 'Scheduled Tasks' section in prompt")
	}
	if !strings.Contains(prompt, "create_schedule") {
		t.Error("expected 'create_schedule' mention in prompt")
	}
}

func TestBuildBaseSystemPrompt_OmitsScheduledTasksWhenNotRegistered(t *testing.T) {
	reg := NewRegistry()

	prompt := BuildBaseSystemPrompt(reg)
	if strings.Contains(prompt, "Scheduled Tasks") {
		t.Error("prompt should not include 'Scheduled Tasks' without schedule tools")
	}
}
