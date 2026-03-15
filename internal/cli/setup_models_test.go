package cli

import (
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
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

func TestProfileTestModels_AllRealProfiles(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		testModels := profileTestModels(profile)

		// Every provider in profile.Providers must get a test model.
		for _, provName := range profile.Providers {
			model, ok := testModels[provName]
			if !ok {
				t.Errorf("profile %q: provider %q has no test model", name, provName)
				continue
			}
			if model == "" {
				t.Errorf("profile %q: provider %q has empty test model", name, provName)
			}
		}

		// No test model should map to a provider not in the profile's list.
		provSet := make(map[string]bool)
		for _, p := range profile.Providers {
			provSet[p] = true
		}
		for provName := range testModels {
			if !provSet[provName] {
				t.Errorf("profile %q: test model for %q but not in Providers list", name, provName)
			}
		}
	}
}

func TestProfileProviders_HaveRegisteredFactories(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		for _, provName := range profile.Providers {
			if _, ok := provider.GetFactory(provName); !ok {
				t.Errorf("profile %q: provider %q has no registered factory", name, provName)
			}
		}
	}
}

func TestProfileProviders_HaveEnvVars(t *testing.T) {
	for _, name := range config.ProfileNames {
		profile := config.Profiles[name]
		for _, provName := range profile.Providers {
			if _, ok := provider.ProviderEnvVars[provName]; !ok {
				t.Errorf("profile %q: provider %q has no env var mapping", name, provName)
			}
		}
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
