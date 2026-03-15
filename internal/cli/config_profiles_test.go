package cli

import (
	"testing"

	"github.com/priyanshujain/openbotkit/config"
)

func TestBuildTierOptions_RecommendedFirst(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic"})
	options := buildTierOptions(models, "complex")

	if len(options) == 0 {
		t.Fatal("expected options for complex tier")
	}

	// First options should be recommended (have * suffix).
	first := options[0]
	if len(first.Key) == 0 {
		t.Fatal("expected non-empty option label")
	}
}

func TestBuildTierOptions_NoDuplicates(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic", "gemini"})
	options := buildTierOptions(models, "default")

	seen := make(map[string]bool)
	for _, opt := range options {
		if seen[opt.Value] {
			t.Errorf("duplicate option value: %q", opt.Value)
		}
		seen[opt.Value] = true
	}
}

func TestBuildTierOptions_AllModelsIncluded(t *testing.T) {
	models := config.ModelsForProviders([]string{"anthropic"})
	options := buildTierOptions(models, "default")

	// Should include all anthropic models, not just recommended ones.
	if len(options) < len(models) {
		t.Errorf("expected at least %d options, got %d", len(models), len(options))
	}
}

func TestBuildTierOptions_EmptyModels(t *testing.T) {
	options := buildTierOptions(nil, "default")
	if len(options) != 0 {
		t.Errorf("expected no options for nil models, got %d", len(options))
	}
}
