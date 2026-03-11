package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// CheckTaskTool retrieves the status and output of delegated tasks.
type CheckTaskTool struct {
	tracker *TaskTracker
}

// NewCheckTaskTool creates a new check_task tool.
func NewCheckTaskTool(tracker *TaskTracker) *CheckTaskTool {
	return &CheckTaskTool{tracker: tracker}
}

func (c *CheckTaskTool) Name() string { return "check_task" }

func (c *CheckTaskTool) Description() string {
	return "Check the status or results of a delegated task (read-only)"
}

func (c *CheckTaskTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"task_id": {
			"type": "string",
			"description": "The task ID to check (omit to list all tasks)"
		}
	}
}`)
}

type checkTaskInput struct {
	TaskID string `json:"task_id,omitempty"`
}

func (c *CheckTaskTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in checkTaskInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	if in.TaskID != "" {
		rec, ok := c.tracker.Get(in.TaskID)
		if !ok {
			return "", fmt.Errorf("task %q not found", in.TaskID)
		}
		data, _ := json.Marshal(rec)
		return string(data), nil
	}

	tasks := c.tracker.List()
	data, _ := json.Marshal(tasks)
	return string(data), nil
}
