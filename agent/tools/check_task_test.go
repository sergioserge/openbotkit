package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCheckTask_Metadata(t *testing.T) {
	tracker := NewTaskTracker()
	tool := NewCheckTaskTool(tracker)
	if tool.Name() != "check_task" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("InputSchema() is not valid JSON")
	}
}

func TestCheckTask_SpecificTask(t *testing.T) {
	tracker := NewTaskTracker()
	tracker.Start("abc", "research Go", AgentClaude)
	tracker.Complete("abc", "Go 1.23 features include...")

	tool := NewCheckTaskTool(tracker)
	input, _ := json.Marshal(checkTaskInput{TaskID: "abc"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var rec TaskRecord
	if err := json.Unmarshal([]byte(result), &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec.ID != "abc" {
		t.Errorf("ID = %q", rec.ID)
	}
	if rec.Status != TaskCompleted {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.Output != "Go 1.23 features include..." {
		t.Errorf("Output = %q", rec.Output)
	}
}

func TestCheckTask_NotFound(t *testing.T) {
	tracker := NewTaskTracker()
	tool := NewCheckTaskTool(tracker)
	input, _ := json.Marshal(checkTaskInput{TaskID: "nonexistent"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for nonexistent task")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestCheckTask_ListAll(t *testing.T) {
	tracker := NewTaskTracker()
	tracker.Start("t1", "task1", AgentClaude)
	tracker.Start("t2", "task2", AgentGemini)
	tracker.Start("t3", "task3", AgentClaude)

	tool := NewCheckTaskTool(tracker)
	input, _ := json.Marshal(checkTaskInput{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var tasks []*TaskRecord
	if err := json.Unmarshal([]byte(result), &tasks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}
}

func TestCheckTask_ListEmpty(t *testing.T) {
	tracker := NewTaskTracker()
	tool := NewCheckTaskTool(tracker)
	input, _ := json.Marshal(checkTaskInput{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "[]" {
		t.Errorf("result = %q, want []", result)
	}
}

func TestCheckTask_RunningTask(t *testing.T) {
	tracker := NewTaskTracker()
	tracker.Start("t1", "in-progress task", AgentClaude)

	tool := NewCheckTaskTool(tracker)
	input, _ := json.Marshal(checkTaskInput{TaskID: "t1"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var rec TaskRecord
	json.Unmarshal([]byte(result), &rec)
	if rec.Status != TaskRunning {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.Output != "" {
		t.Errorf("Output should be empty for running task: %q", rec.Output)
	}
}
