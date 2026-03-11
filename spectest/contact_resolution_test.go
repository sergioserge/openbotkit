package spectest

import (
	"context"
	"testing"
	"time"
)

func TestSpec_ResolveContactAndSendWhatsApp(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:        "David Chen",
				Phones:      []string{"+919876543210"},
				WhatsAppJID: "919876543210@s.whatsapp.net",
				Aliases:     []string{"David", "Dave"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Tell David I'll be late"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent must have searched contacts for David and found David Chen's WhatsApp JID "+
				"(919876543210@s.whatsapp.net). It must have attempted to send a WhatsApp message "+
				"using 'obk whatsapp messages send' or similar. An authentication or connection "+
				"failure is acceptable — what matters is the agent resolved the correct contact "+
				"and attempted the send with the right JID.")
	})
}

func TestSpec_ResolveContactAmbiguous(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:        "David Chen",
				Phones:      []string{"+919876543210"},
				WhatsAppJID: "919876543210@s.whatsapp.net",
				Aliases:     []string{"David"},
			},
			{
				Name:        "David Miller",
				Phones:      []string{"+14155551234"},
				WhatsAppJID: "14155551234@s.whatsapp.net",
				Aliases:     []string{"David"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Tell David I'll be late"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent found multiple contacts named David (David Chen and David Miller). "+
				"It must have indicated the ambiguity — either by asking the user to clarify "+
				"which David, or by listing both matches. The agent should NOT have silently "+
				"picked one David without acknowledging the ambiguity.")
	})
}

func TestSpec_ResolveContactForEmail(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:    "Alice Smith",
				Emails:  []string{"alice@acme.com"},
				Aliases: []string{"Alice"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Email Alice about the meeting tomorrow"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent must have searched contacts for Alice and found Alice Smith's email "+
				"(alice@acme.com). It must have attempted to send or draft an email to "+
				"alice@acme.com using 'obk email send' or similar. An authentication or "+
				"configuration failure is acceptable — what matters is the agent resolved "+
				"the correct contact and attempted the email with the right address.")
	})
}
