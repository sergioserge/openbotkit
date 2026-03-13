package agent

import (
	"fmt"
	"strings"

	"github.com/priyanshujain/openbotkit/provider"
)

const defaultMaxHistory = 40

const microcompactAge = 3

func (a *Agent) microcompact() {
	if len(a.history) < microcompactAge*2 {
		return
	}
	cutoff := len(a.history) - microcompactAge*2
	for i := 0; i < cutoff; i++ {
		msg := &a.history[i]
		if msg.Role != provider.RoleUser {
			continue
		}
		for j := range msg.Content {
			block := &msg.Content[j]
			if block.Type != provider.ContentToolResult || block.ToolResult == nil {
				continue
			}
			if len(block.ToolResult.Content) <= 200 {
				continue
			}
			if idx := strings.LastIndex(block.ToolResult.Content, "[Full output: "); idx >= 0 {
				ref := block.ToolResult.Content[idx:]
				block.ToolResult.Content = fmt.Sprintf("[Previous: used %s] %s", block.ToolResult.Name, ref)
			} else {
				block.ToolResult.Content = fmt.Sprintf("[Previous: used %s]", block.ToolResult.Name)
			}
		}
	}
}

func (a *Agent) compactHistory() {
	if len(a.history) <= a.maxHistory {
		return
	}
	keep := a.maxHistory / 2
	if keep < 1 {
		keep = 1
	}
	removed := len(a.history) - keep
	summary := provider.NewTextMessage(provider.RoleUser,
		fmt.Sprintf("[Earlier conversation: %d messages removed]", removed))
	a.history = append([]provider.Message{summary}, a.history[removed:]...)
}
