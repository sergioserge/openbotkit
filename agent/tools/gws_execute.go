package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/oauth/google"
)

const defaultAuthTimeout = 5 * time.Minute

// GWSToolConfig configures a GWSExecuteTool.
type GWSToolConfig struct {
	Interactor    Interactor
	ScopeChecker  ScopeChecker
	Bridge        *TokenBridge
	ScopeWaiter   *google.ScopeWaiter
	Google        *google.Google
	Account       string
	Manifest      *skills.Manifest
	Runner        CommandRunner
	AuthTimeout   time.Duration
	ApprovalRules *ApprovalRuleSet
}

// GWSExecuteTool routes all gws commands through a single tool
// with scope checking, progressive consent, and write approval.
type GWSExecuteTool struct {
	interactor    Interactor
	scopeChecker  ScopeChecker
	bridge        *TokenBridge
	scopeWaiter   *google.ScopeWaiter
	google        *google.Google
	account       string
	manifest      *skills.Manifest
	runner        CommandRunner
	authTimeout   time.Duration
	approvalRules *ApprovalRuleSet
}

func NewGWSExecuteTool(cfg GWSToolConfig) *GWSExecuteTool {
	timeout := cfg.AuthTimeout
	if timeout == 0 {
		timeout = defaultAuthTimeout
	}
	return &GWSExecuteTool{
		interactor:    cfg.Interactor,
		scopeChecker:  cfg.ScopeChecker,
		bridge:        cfg.Bridge,
		scopeWaiter:   cfg.ScopeWaiter,
		google:        cfg.Google,
		account:       cfg.Account,
		manifest:      cfg.Manifest,
		runner:        cfg.Runner,
		authTimeout:   timeout,
		approvalRules: cfg.ApprovalRules,
	}
}

func (g *GWSExecuteTool) Name() string        { return "gws_execute" }
func (g *GWSExecuteTool) Description() string { return "Execute a Google Workspace CLI (gws) command" }
func (g *GWSExecuteTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The gws command to execute (e.g. 'calendar events.list --maxResults 10')"
			}
		},
		"required": ["command"]
	}`)
}

type gwsInput struct {
	Command string `json:"command"`
}

func (g *GWSExecuteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in gwsInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	slog.Info("gws_execute called", "command", in.Command)
	args := splitGWSCommand(in.Command)
	// Strip leading "gws" if present — skill examples include it but the runner adds it.
	if len(args) > 0 && args[0] == "gws" {
		args = args[1:]
	}
	service := gwsServiceFromCommand(args)
	isWrite := g.isWriteCommand(args)

	// Scope check + progressive consent.
	if service != "" {
		requiredScopes := g.scopesForService(service)
		if len(requiredScopes) > 0 {
			has, err := g.scopeChecker.HasScopes(g.account, requiredScopes)
			if err != nil {
				return "", fmt.Errorf("check scopes: %w", err)
			}
			if !has {
				if err := g.requestConsent(ctx, requiredScopes); err != nil {
					return "", err
				}
			}
		}
	}

	// Write approval.
	if isWrite {
		var opts []GuardOption
		if g.approvalRules != nil {
			opts = append(opts, WithApprovalRules(g.approvalRules, "gws_execute", input))
		}
		return GuardedAction(ctx, g.interactor, RiskHigh, fmt.Sprintf("Run gws command: %s", in.Command), func() (string, error) {
			return g.run(ctx, args)
		}, opts...)
	}

	return g.run(ctx, args)
}

func (g *GWSExecuteTool) run(ctx context.Context, args []string) (string, error) {
	env, err := g.bridge.Env(ctx)
	if err != nil {
		slog.Warn("gws_execute: token error, attempting re-auth", "error", err)
		// Token expired or refresh failed — trigger re-consent and retry.
		service := gwsServiceFromCommand(args)
		scopes := g.scopesForService(service)
		if len(scopes) == 0 {
			return "", fmt.Errorf("get token: %w", err)
		}
		if cerr := g.requestConsent(ctx, scopes); cerr != nil {
			return "", fmt.Errorf("get token: %w (re-auth also failed: %v)", err, cerr)
		}
		env, err = g.bridge.Env(ctx)
		if err != nil {
			return "", fmt.Errorf("get token after re-auth: %w", err)
		}
	}
	return g.runner.Run(ctx, args, env)
}

func (g *GWSExecuteTool) requestConsent(ctx context.Context, scopes []string) error {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Errorf("generate state: %w", err)
	}
	state := "gws-" + hex.EncodeToString(b[:])
	url, err := g.google.AuthURL(g.account, scopes, state)
	if err != nil {
		return fmt.Errorf("generate auth URL: %w", err)
	}
	if err := g.interactor.Notify("I need additional Google access to complete this request."); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	if err := g.interactor.NotifyLink("Tap to grant access", url); err != nil {
		return fmt.Errorf("notify link: %w", err)
	}

	if err := g.scopeWaiter.Wait(state, g.authTimeout, scopes, g.account); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	// After first-time auth, discover the account from the token store.
	if g.account == "" {
		accounts, err := g.google.Accounts(ctx)
		if err == nil && len(accounts) > 0 {
			g.account = accounts[0]
			g.bridge.SetAccount(accounts[0])
		}
	}

	if err := g.interactor.Notify("Access granted, thanks!"); err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	return nil
}

func (g *GWSExecuteTool) isWriteCommand(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "+") {
			return true
		}
	}
	return false
}

// serviceToScope maps gws short service names to full Google API scope URLs.
var serviceToScope = map[string]string{
	"calendar": "https://www.googleapis.com/auth/calendar",
	"drive":    "https://www.googleapis.com/auth/drive",
	"docs":     "https://www.googleapis.com/auth/documents",
	"sheets":   "https://www.googleapis.com/auth/spreadsheets",
	"tasks":    "https://www.googleapis.com/auth/tasks",
	"people":   "https://www.googleapis.com/auth/contacts",
}

func (g *GWSExecuteTool) scopesForService(service string) []string {
	if g.manifest == nil {
		return nil
	}
	for _, entry := range g.manifest.Skills {
		if entry.Source == "gws" && len(entry.Scopes) > 0 && entry.Scopes[0] == service {
			scope, ok := serviceToScope[service]
			if !ok {
				return nil
			}
			return []string{scope}
		}
	}
	return nil
}

// splitGWSCommand splits a gws command string into args, handling --params
// and --json flags whose values are JSON objects containing spaces and quotes.
// LLMs often wrap JSON in single quotes (shell convention) which standard
// splitters cannot handle reliably because the JSON also contains double quotes.
func splitGWSCommand(cmd string) []string {
	var args []string
	cmd = strings.TrimSpace(cmd)
	for len(cmd) > 0 {
		if cmd[0] == '\'' || cmd[0] == '{' {
			// JSON or quoted JSON value — extract using brace matching.
			s := cmd
			if s[0] == '\'' {
				s = s[1:] // skip opening quote
			}
			if len(s) > 0 && s[0] == '{' {
				jsonVal, rest := extractJSONObject(s)
				args = append(args, jsonVal)
				rest = strings.TrimLeft(rest, "'") // strip trailing quote
				cmd = strings.TrimSpace(rest)
				continue
			}
		}
		// Regular token — read until next whitespace.
		end := strings.IndexByte(cmd, ' ')
		if end < 0 {
			args = append(args, cmd)
			break
		}
		args = append(args, cmd[:end])
		cmd = strings.TrimSpace(cmd[end:])
	}
	return args
}

// extractJSONObject returns the substring from the opening { to its matching },
// counting brace depth. It handles braces inside JSON string literals.
func extractJSONObject(s string) (string, string) {
	depth := 0
	inString := false
	escaped := false
	for i, c := range s {
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[:i+1], s[i+1:]
			}
		}
	}
	return s, ""
}

// gwsServiceFromCommand extracts the service name from gws command args.
func gwsServiceFromCommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
