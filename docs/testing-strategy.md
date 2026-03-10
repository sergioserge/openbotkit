# Testing Strategy

## Why this exists

openbotkit is a personal AI agent that crosses many layers — LLM providers, tool execution, skill discovery, bash commands, SQLite databases, messaging channels. A bug anywhere in this chain silently degrades the user experience. We need confidence at every level: that individual functions work, that components integrate correctly, and that the agent can actually complete a user's task end-to-end.

This document describes how we think about testing across the project, from fast isolated unit tests up to full LLM-driven spec tests.

## The testing pyramid

```
        Spec Tests
        Prove the agent completes real user tasks
        Real LLMs, real databases, LLM-judged assertions

      Provider Conformance Tests
      Each LLM provider drives the agent protocol correctly

    Session Integration Tests
    Message → agent → response → storage pipeline

  Server API Contract Tests
  HTTP API endpoints honor their contracts

Unit Tests (the base)
Individual functions in isolation, fast and deterministic
```

Each level tests something the level below cannot. We don't skip levels — a spec test passing doesn't excuse a missing unit test for the function it exercises.

## Unit tests

**What they cover:** Individual functions with no external dependencies. Mocks, stubs, and in-memory SQLite databases keep them fast and deterministic.

**Examples across the project:**

| Area | What's tested | Pattern |
|------|--------------|---------|
| `agent/agent_test.go` | Agent loop: text responses, tool calls, multi-tool sequences, max iterations, secret scrubbing, history compaction | `mockProvider` returns scripted LLM responses; `mockExecutor` records tool calls |
| `agent/tools/tools_test.go` | Tool registry, bash execution (echo, timeout, stderr), file operations, output truncation (524KB limit) | Direct function calls with temp directories |
| `provider/resilient_test.go` | Retry logic: 429 retries, 401 doesn't retry, max retries respected | Mock HTTP responses |
| `provider/ratelimit_test.go` | Burst allowance, context cancellation | Time-based assertions |
| `provider/credential_test.go` | Keyring parsing, store/load, env var fallback | Isolated credential operations |
| `memory/store_test.go` | CRUD, list, search, count, duplicate handling | In-memory SQLite via `testDB()` |
| `memory/schema_test.go` | Migration idempotency | Run migration twice, verify no error |
| `source/whatsapp/sync_test.go` | Content extraction from protobuf message types, text truncation | Direct function calls with proto builders |
| `source/whatsapp/send_test.go` | JID validation, group detection, message upsert, dedup | In-memory SQLite |
| `source/gmail/store_test.go` | Email CRUD | In-memory SQLite |
| `channel/telegram/telegram_test.go` | Send (Markdown formatting), receive (EOF on close), approval keyboard, owner filtering | `mockBot` captures sent messages |
| `config/config_test.go` | Defaults, path DSN generation, YAML loading | File-based config in temp dirs |
| `internal/skills/` | Manifest parsing, skill indexing, skill installation | Temp directories with fixture files |

**Conventions:**
- All database tests use in-memory SQLite or `t.TempDir()` — no shared state between tests.
- Mocks are defined in the test file that uses them, not in shared packages. `mockProvider`, `mockExecutor`, `mockBot`, and `stubTool` each live next to their tests.
- No test frameworks beyond the standard `testing` package. Assertions are manual `if err != nil` checks with `t.Fatalf`.

## Server API contract tests

**What they prove:** The HTTP API endpoints accept valid requests, reject invalid ones, enforce authentication, and return correct responses.

**Location:** `internal/servertest/suite.go` defines the test suite. `internal/server/api_test.go` runs it against a local httptest server.

**How it works:**

```go
backend := servertest.Backend{
    Client:       authenticatedClient,
    NoAuthClient: unauthenticatedClient,
    SeedDB:       func(source, sql string) error { ... },
}
servertest.Run(t, backend)
```

The suite tests:
- Health checks work without authentication
- Memory CRUD (add, list, filter by category, delete)
- Authentication enforcement (memory operations rejected without auth)
- Apple Notes push and query (upsert behavior)
- Database proxy (rejects non-SELECT queries, rejects unknown sources)
- Gmail/WhatsApp send validation
- Raw database seeding and querying

The `SeedDB` callback lets the suite inject test data directly into SQLite, independent of API endpoints. This means the suite tests the API layer's behavior, not the database layer's.

## Provider conformance tests

**What they prove:** Each LLM provider implementation correctly drives the agent protocol — tool calls are emitted, tool results are consumed, streaming works.

**Location:** `agent/provider_test.go`

**Tests:**
- `TestProvider_AgentToolExecution` — provider drives the full agent loop (tool call → execution → final response)
- `TestProvider_ToolUseRoundtrip` — tool_use → tool_result → text_response cycle
- `TestProvider_Streaming` — text deltas arrive followed by a done event

**Multi-provider execution:** Tests run for every available provider (Anthropic, Anthropic Vertex, OpenAI, Gemini) using an `EachProvider` helper. If credentials aren't set, that provider is skipped — not failed.

This catches provider-specific quirks: Gemini needing function names on tool results, OpenAI streaming differently, Anthropic Vertex requiring project IDs. Same behavioral contract, different implementations.

## Session integration tests

**What they prove:** The full message → agent → response → storage pipeline works for each messaging channel.

**Locations:** `channel/telegram/integration_test.go`, `source/whatsapp/integration_test.go`

**Telegram examples:**
- `TestSession_MessageAndHistorySaved` — message goes in, Gemini produces a response, the response is sent via bot, and conversation history is saved to SQLite.
- `TestSession_MemoryInjectedIntoPrompt` — memories from the database appear in the system prompt the LLM receives.
- `TestSession_ToolUseViaBash` — agent executes bash commands during a session and the results flow back.

**Setup pattern:** Each test creates a temp config directory, migrates history and memory databases, creates a channel with a mock bot, and wires a real LLM provider. The mock bot captures `Send()` calls so we can verify what the user would see.

These tests use real LLM APIs (currently Gemini) and skip if credentials are absent.

## Spec tests

**What they prove:** The agent can complete real user tasks end-to-end — discover skills, query databases, combine information from multiple sources, and produce useful answers.

**Location:** `spectest/`

This is the highest-value, highest-cost testing layer. Unit tests can't catch "the agent asks the LLM for a SQL query but the skill prompt is wrong." Integration tests can't catch "the agent finds emails but fails to also check WhatsApp when asked about all communications." Spec tests can.

### Core ideas

**Test user behavior, not implementation.** The spec is "find emails from Alice" or "summarize communications across sources" — things the user does. Tests don't know how the agent routes through tools internally.

**Declarative state setup (Given pattern).** `fixture.GivenEmails(t, emails)` reads like a story. The fixture knows how to produce that state (insert into SQLite, install skills, build the CLI binary). The test doesn't care.

**Same spec, multiple providers.** `EachProvider(t, func(t *testing.T, fx *LocalFixture) { ... })` runs the same behavioral test against Gemini, Anthropic Vertex, and OpenAI. Same assertions, different LLM backends.

### Fixture design

```go
type Fixture interface {
    Agent(t *testing.T) *agent.Agent
    GivenEmails(t *testing.T, emails []Email)
    GivenWhatsAppMessages(t *testing.T, messages []Message)
    GivenMemories(t *testing.T, memories []Memory)
}
```

**Local fixture** (`local_fixture.go`):
- `OBK_CONFIG_DIR` → `t.TempDir()` with real SQLite databases, migrated
- Skills embedded and installed from source tree
- `obk` binary built via `go build` and placed in PATH
- `GivenEmails` → inserts rows directly into `gmail_emails` table
- Agent wired with real tools (bash, skills) pointing at test databases
- Provider → real LLM (env-gated)

### Assertion strategy

The agent is non-deterministic — the LLM may phrase responses differently each run. But the data is deterministic: if the agent found the right email, the sender address and subject line appear verbatim.

Three layers, applied progressively:

**Layer 1 — Structural (free, deterministic).** Check what the agent did, not what it said. Record tool calls. For send operations, check a recorder: was a message sent to the right recipient?

**Layer 2 — Substring on data (free, deterministic).** The response should contain `alice@example.com` or `Meeting Tomorrow`. These are database values — the LLM retrieves them, it doesn't rephrase them.

**Layer 3 — LLM judge (costs money, flexible).** A cheap model evaluates: "Does this response answer the question about emails from Alice? Does it include the meeting email?" This handles semantic assertions that substring matching can't express.

```go
AssertJudge(t, fx, response, []string{
    "mentions at least one email from alice",
    "includes the subject line about the meeting",
})
```

We use Layer 1+2 by default and add Layer 3 when the assertion is genuinely semantic.

### Current specs

| Spec | What it proves |
|------|---------------|
| `TestSpec_FindEmailsBySender` | Agent discovers email-read skill, queries database, filters by sender |
| `TestSpec_SummarizeCommunicationsAcrossSources` | Agent autonomously queries both email and WhatsApp, synthesizes results |
| `TestSpec_RecallMemoryAndCorrelateEmails` | Agent combines personal memories with email search |

## Graceful skipping

Tests that need external resources (LLM API keys, sqlite3 binary, Docker) skip with `t.Skip()` rather than fail. This means:

- `go test ./...` always passes on a fresh checkout (unit tests run, integration tests skip)
- CI with credentials set runs the full suite
- A developer without a Gemini key can still run and trust all non-LLM tests

Helper functions like `testutil.RequireGeminiKey(t)` standardize this pattern.

## CI configuration

`.github/workflows/ci.yml` runs:
- `go test ./...` on Ubuntu, macOS, and Windows
- Cross-compilation for linux/amd64, linux/arm64, linux/riscv64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64

## When to write which test

| Situation | Test level |
|-----------|-----------|
| New function or method | Unit test |
| New API endpoint | Add to server contract suite |
| New LLM provider | Provider conformance tests |
| New messaging channel | Session integration test |
| New user-facing capability ("the agent can now do X") | Spec test |
| Bug fix | Test at the lowest level that reproduces the bug |
