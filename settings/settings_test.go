package settings

import (
	"testing"

	"github.com/73ai/openbotkit/config"
)

func testService(cfg *config.Config) *Service {
	creds := make(map[string]string)
	return New(cfg,
		WithSaveFn(func(*config.Config) error { return nil }),
		WithStoreCred(func(ref, value string) error {
			creds[ref] = value
			return nil
		}),
		WithLoadCred(func(ref string) (string, error) {
			v, ok := creds[ref]
			if !ok {
				return "", nil
			}
			return v, nil
		}),
	)
}

func TestGetSetMode(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "mode")
	if field == nil {
		t.Fatal("mode field not found")
	}

	got := svc.GetValue(field)
	if got != "local" {
		t.Errorf("default mode = %q, want %q", got, "local")
	}

	if err := svc.SetValue(field, "remote"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "remote" {
		t.Errorf("after set, mode = %q, want %q", got, "remote")
	}
}

func TestGetSetTimezone(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "timezone")
	if field == nil {
		t.Fatal("timezone field not found")
	}

	if err := svc.SetValue(field, "Asia/Ho_Chi_Minh"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "Asia/Ho_Chi_Minh" {
		t.Errorf("timezone = %q, want %q", got, "Asia/Ho_Chi_Minh")
	}
}

func TestTimezoneValidation(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "timezone")
	if field == nil {
		t.Fatal("timezone field not found")
	}

	if err := svc.SetValue(field, "Not/A/Timezone"); err == nil {
		t.Error("expected validation error for invalid timezone")
	}

	if err := svc.SetValue(field, ""); err != nil {
		t.Errorf("empty timezone should be valid: %v", err)
	}
}

func TestGetSetNilModels(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "models.default")
	if field == nil {
		t.Fatal("models.default field not found")
	}

	got := svc.GetValue(field)
	if got != "" {
		t.Errorf("nil models default = %q, want empty", got)
	}

	if err := svc.SetValue(field, "anthropic/claude-sonnet-4-6"); err != nil {
		t.Fatal(err)
	}
	if cfg.Models == nil {
		t.Fatal("Models should be initialized after set")
	}
	if got := svc.GetValue(field); got != "anthropic/claude-sonnet-4-6" {
		t.Errorf("default model = %q, want %q", got, "anthropic/claude-sonnet-4-6")
	}
}

func TestGetSetBool(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "gmail.download_attachments")
	if field == nil {
		t.Fatal("gmail.download_attachments field not found")
	}

	if err := svc.SetValue(field, "true"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "true" {
		t.Errorf("download_attachments = %q, want %q", got, "true")
	}
}

func TestGetSetNumber(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "gmail.sync_days")
	if field == nil {
		t.Fatal("gmail.sync_days field not found")
	}

	if err := svc.SetValue(field, "30"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "30" {
		t.Errorf("sync_days = %q, want %q", got, "30")
	}

	if err := svc.SetValue(field, "abc"); err == nil {
		t.Error("expected validation error for non-numeric value")
	}
}

func TestPasswordField(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.providers.anthropic.api_key")
	if field == nil {
		t.Fatal("anthropic api_key field not found")
	}

	got := svc.GetValue(field)
	if got != "not configured" {
		t.Errorf("initial api key status = %q, want %q", got, "not configured")
	}

	if err := svc.SetValue(field, "sk-test-key"); err != nil {
		t.Fatal(err)
	}
	got = svc.GetValue(field)
	if got != "configured" {
		t.Errorf("after set, api key status = %q, want %q", got, "configured")
	}
}

func TestNilChannels(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "channels.telegram.bot_token")
	if field == nil {
		t.Fatal("telegram bot_token field not found")
	}

	got := svc.GetValue(field)
	if got != "not configured" {
		t.Errorf("nil channels bot_token = %q, want %q", got, "not configured")
	}

	if err := svc.SetValue(field, "123:ABC"); err != nil {
		t.Fatal(err)
	}
	if cfg.Channels == nil || cfg.Channels.Telegram == nil {
		t.Fatal("Channels.Telegram should be initialized after set")
	}
}

func TestTreeStructure(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	tree := svc.Tree()
	if len(tree) != 6 {
		t.Fatalf("tree has %d top-level nodes, want 6", len(tree))
	}

	labels := []string{"General", "Models", "Channels", "Data Sources", "Integrations", "Advanced"}
	for i, n := range tree {
		if n.Category == nil {
			t.Errorf("tree[%d] is not a category", i)
			continue
		}
		if n.Category.Label != labels[i] {
			t.Errorf("tree[%d].Label = %q, want %q", i, n.Category.Label, labels[i])
		}
	}
}

func findField(svc *Service, key string) *Field {
	return findFieldInNodes(svc.Tree(), key)
}

func findFieldInNodes(nodes []Node, key string) *Field {
	for _, n := range nodes {
		if n.Field != nil && n.Field.Key == key {
			return n.Field
		}
		if n.Category != nil {
			if f := findFieldInNodes(n.Category.Children, key); f != nil {
				return f
			}
		}
	}
	return nil
}
