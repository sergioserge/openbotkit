package spectest

import (
	"testing"

	"github.com/priyanshujain/openbotkit/agent"
)

type Email struct {
	MessageID string
	Account   string
	From      string
	To        string
	Subject   string
	Body      string
}

type WhatsAppMessage struct {
	MessageID  string
	ChatJID    string
	ChatName   string
	SenderJID  string
	SenderName string
	Text       string
	IsFromMe   bool
}

type UserMemory struct {
	Content  string
	Category string // identity, preference, relationship, project
}

type Fixture interface {
	Agent(t *testing.T) *agent.Agent
	GivenEmails(t *testing.T, emails []Email)
	GivenWhatsAppMessages(t *testing.T, messages []WhatsAppMessage)
	GivenMemories(t *testing.T, memories []UserMemory)
}
