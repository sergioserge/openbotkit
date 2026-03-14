package cli

import (
	"testing"

	"github.com/priyanshujain/openbotkit/config"
)

func TestProfileTestModels_PicksOnePerProvider(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
			Fast:    "openrouter/google/gemini-2.0-flash-lite",
			Nano:    "groq/llama-3.1-8b-instant",
		},
	}

	result := profileTestModels(profile)

	if len(result) != 2 {
		t.Fatalf("expected 2 providers, got %d: %v", len(result), result)
	}
	if _, ok := result["openrouter"]; !ok {
		t.Error("expected openrouter in result")
	}
	if result["groq"] != "llama-3.1-8b-instant" {
		t.Errorf("groq model = %q, want llama-3.1-8b-instant", result["groq"])
	}
}

func TestProfileTestModels_EmptyTiers(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{},
	}
	result := profileTestModels(profile)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestProfileTestModels_FirstModelPerProvider(t *testing.T) {
	profile := config.ModelProfile{
		Tiers: config.ProfileTiers{
			Default: "openrouter/anthropic/claude-haiku-4-5",
			Complex: "openrouter/anthropic/claude-sonnet-4-6",
		},
	}

	result := profileTestModels(profile)
	// Should pick the first one encountered (Default) for openrouter.
	if result["openrouter"] != "anthropic/claude-haiku-4-5" {
		t.Errorf("openrouter model = %q, want anthropic/claude-haiku-4-5", result["openrouter"])
	}
}

func TestWarnDefaultContextWindow_NoWarningForLargeContext(t *testing.T) {
	// Should not panic or error for models with >=128k context.
	warnDefaultContextWindow("anthropic/claude-sonnet-4-6")
}

func TestWarnDefaultContextWindow_EmptySpec(t *testing.T) {
	// Should not panic on empty spec.
	warnDefaultContextWindow("")
}

func TestWarnDefaultContextWindow_InvalidSpec(t *testing.T) {
	// Should not panic on spec without slash.
	warnDefaultContextWindow("no-slash")
}
