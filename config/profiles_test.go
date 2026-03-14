package config

import (
	"strings"
	"testing"
)

func TestProfilesHaveValidModelSpecs(t *testing.T) {
	for name, profile := range Profiles {
		for _, spec := range []string{profile.Tiers.Default, profile.Tiers.Complex, profile.Tiers.Fast, profile.Tiers.Nano} {
			if spec == "" {
				continue
			}
			parts := strings.SplitN(spec, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				t.Errorf("profile %q: invalid model spec %q", name, spec)
			}
		}
	}
}

func TestProfilesHaveRequiredProviders(t *testing.T) {
	for name, profile := range Profiles {
		providerSet := make(map[string]bool)
		for _, p := range profile.Providers {
			providerSet[p] = true
		}

		for _, spec := range []string{profile.Tiers.Default, profile.Tiers.Complex, profile.Tiers.Fast, profile.Tiers.Nano} {
			if spec == "" {
				continue
			}
			parts := strings.SplitN(spec, "/", 2)
			provName := parts[0]
			if !providerSet[provName] {
				t.Errorf("profile %q: tier spec %q uses provider %q not in Providers list", name, spec, provName)
			}
		}
	}
}

func TestProfileNamesMatchKeys(t *testing.T) {
	for _, name := range ProfileNames {
		profile, ok := Profiles[name]
		if !ok {
			t.Errorf("ProfileNames contains %q but Profiles map does not", name)
			continue
		}
		if profile.Name != name {
			t.Errorf("profile %q: Name field = %q, want %q", name, profile.Name, name)
		}
	}
}

func TestAllProfilesInProfileNames(t *testing.T) {
	nameSet := make(map[string]bool)
	for _, n := range ProfileNames {
		nameSet[n] = true
	}
	for name := range Profiles {
		if !nameSet[name] {
			t.Errorf("profile %q in Profiles map but not in ProfileNames", name)
		}
	}
}
