# memory

Personal memory system for OpenBotKit. Stores durable facts about the user — preferences, identity, relationships, projects — that persist across conversations and personalize the assistant.

This is distinct from `source/history/`, which stores conversation transcripts.

## Data Model

Each memory is an atomic fact with a category:

| Category       | Examples                                    | Update Frequency |
|----------------|---------------------------------------------|------------------|
| `identity`     | Name, location, profession, education       | Rarely           |
| `preference`   | Likes dark mode, prefers Go, vegetarian     | Occasionally     |
| `relationship` | Wife is Sarah, dog named Max                | Occasionally     |
| `project`      | Building OpenBotKit, learning Rust          | Frequently       |

```go
type Memory struct {
    ID        int64
    Content   string    // "User prefers Go over Python"
    Category  Category  // identity, preference, relationship, project
    Source    string    // "history", "manual", "whatsapp", "gmail", "applenotes"
    SourceRef string    // optional reference (session_id, etc.)
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Store (CRUD)

```go
// Write
id, err := memory.Add(db, "User prefers dark mode", memory.CategoryPreference, "manual", "")
err := memory.Update(db, id, "User prefers light mode")
err := memory.Delete(db, id)

// Read
m, err := memory.Get(db, id)
all, err := memory.List(db)
prefs, err := memory.ListByCategory(db, memory.CategoryPreference)
results, err := memory.Search(db, "dark mode")
count, err := memory.Count(db)
```

## Extraction Pipeline

Extracts personal facts from conversation messages using an LLM.

```
Messages → preFilter (skip acks, short messages) → LLM extraction → []CandidateFact
```

```go
facts, err := memory.Extract(ctx, llm, messages)
```

Pre-filtering removes trivial messages (< 5 words, acknowledgements like "ok", "thanks", "yes") before sending to the LLM. The LLM returns a JSON array of `{content, category}` pairs.

## Reconciliation

Reconciles extracted facts against existing memories to avoid duplicates. For each candidate fact:

1. Search existing memories by keyword
2. If no matches found → ADD directly
3. If matches found → ask LLM to decide: ADD, UPDATE, DELETE, or NOOP

```go
result, err := memory.Reconcile(ctx, db, llm, candidates)
// result.Added, result.Updated, result.Deleted, result.Skipped
```

## Prompt Injection

Formats all memories as markdown for inclusion in the agent's system prompt:

```go
text := memory.FormatForPrompt(memories)
```

Output:
```
## About the user

### Identity
- User's name is Priyanshu
- User is a software engineer

### Preferences
- User prefers Go over Python
- User prefers dark mode

### Relationships
- (none)

### Projects & Context
- User is building OpenBotKit
```

## LLM Interface

Extract and Reconcile depend on an `LLM` interface for testability:

```go
type LLM interface {
    Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error)
}
```

In production, use `RouterLLM` which adapts a `provider.Router` with a fixed model tier:

```go
llm := &memory.RouterLLM{Router: router, Tier: provider.TierFast}
```

In tests, use a mock:

```go
llm := &mockLLM{response: `[{"content": "User likes Go", "category": "preference"}]`}
```

## CLI Commands

```
obk memory list [--category <cat>]    # list all memories
obk memory add "fact" --category preference --source manual
obk memory delete <id>
obk memory extract [--last <n>]       # extract from recent history sessions
```

## End-to-End Flow

```
Conversation ends
  → hook runs: obk history capture && obk memory extract &
    → history capture: parses transcript, saves to history DB
    → memory extract (background):
        1. Load recent messages from history DB
        2. Extract candidate facts via LLM
        3. Reconcile against existing memories
        4. Print summary: "Added 3, Updated 1, Deleted 0, Skipped 2"
```

## Design Decisions

**Why LLM extraction over rules?** Rule-based approaches (regex, NER) catch only ~20-30% of extractable facts. They miss implicit information like "I had sushi again last night" (implies preference). Cost is negligible — under $0.50/month even with expensive models.

**Why inject-all instead of RAG?** For a single user, memories stay under 200 for months. Inject-all has 100% recall with no retrieval misses. RAG can be added later at 300+ memories using `sqlite-vec`.

**Why reactive-only mid-conversation saves?** Proactive saving (ChatGPT's approach) has well-documented failure modes: over-saving random things, context rot, duplicate entries. The `memory-save` skill only triggers on explicit "remember this" requests. Post-conversation extraction handles implicit facts.

**Why these 4 categories?** Inspired by Gemini's approach. Categories map to natural update frequencies — demographics rarely change, projects change constantly. This enables selective reconciliation.

## References

### Production Memory Systems
- [Memory and new controls for ChatGPT — OpenAI](https://openai.com/index/memory-and-new-controls-for-chatgpt/)
- [How ChatGPT Memory Works, Reverse Engineered — LLMRefs](https://llmrefs.com/blog/reverse-engineering-chatgpt-memory)
- [I Reverse Engineered ChatGPT's Memory System — Manthan Gupta](https://manthanguptaa.in/posts/chatgpt_memory/)
- [Reverse Engineering ChatGPT's Updated Memory System — Julian Fleck](https://medium.com/@j0lian/reverse-engineering-chatgpts-updated-memory-system-3cb9e82e5d21)
- [I really don't like ChatGPT's new memory dossier — Simon Willison](https://simonwillison.net/2025/May/21/chatgpt-new-memory/)
- [Comparing the memory implementations of Claude and ChatGPT — Simon Willison](https://simonwillison.net/2025/Sep/12/claude-memory/)
- [How ChatGPT Remembers You — Embrace The Red](https://embracethered.com/blog/posts/2025/chatgpt-how-does-chat-history-memory-preferences-work/)
- [Why I Turned Off ChatGPT's Memory — Every.to](https://every.to/also-true-for-humans/why-i-turned-off-chatgpt-s-memory)
- [Inside ChatGPT's Memory — AI Monks](https://medium.com/aimonks/inside-chatgpts-memory-how-the-most-sophisticated-memory-system-in-ai-really-works-f2b3f32d86b3)
- [Google Has Your Data. Gemini Barely Uses It. — Shlok Khemani](https://www.shloked.com/writing/gemini-memory)
- [Gemini's New Memory Feature Update — Kai](https://kaiwritesornot.medium.com/geminis-new-memory-feature-update-58c2872689a6)
- [Memory for AI Code Reviews using Gemini Code Assist — Google Cloud](https://cloud.google.com/blog/products/ai-machine-learning/memory-for-ai-code-reviews-using-gemini-code-assist)

### Frameworks & Libraries
- [MemGPT: Towards LLMs as Operating Systems — arXiv](https://arxiv.org/abs/2310.08560)
- [Intro to Letta — Letta Docs](https://docs.letta.com/concepts/memgpt/)
- [Agent Memory: How to Build Agents that Learn and Remember — Letta](https://www.letta.com/blog/agent-memory)
- [RAG is not Agent Memory — Letta](https://www.letta.com/blog/rag-vs-agent-memory)
- [Mem0 GitHub Repository](https://github.com/mem0ai/mem0)
- [Mem0: Building Production-Ready AI Agents with Scalable Long-Term Memory — arXiv](https://arxiv.org/abs/2504.19413)
- [Mem0: How Three Prompts Created a Viral AI Memory Layer](https://blog.lqhl.me/mem0-how-three-prompts-created-a-viral-ai-memory-layer)
- [Mem0 Benchmark: OpenAI Memory vs LangMem vs MemGPT vs Mem0](https://mem0.ai/blog/benchmarked-openai-memory-vs-langmem-vs-memgpt-vs-mem0-for-long-term-memory-here-s-how-they-stacked-up)
- [LangMem Conceptual Guide](https://langchain-ai.github.io/langmem/concepts/conceptual_guide/)
- [How to Extract Semantic Memories — LangMem](https://langchain-ai.github.io/langmem/guides/extract_semantic_memories/)

### Research Papers
- [Beyond the Context Window: Fact-Based Memory vs Long-Context LLMs — arXiv](https://arxiv.org/html/2603.04814)
- [MemoryBank: Enhancing Large Language Models with Long-Term Memory — arXiv](https://arxiv.org/abs/2305.10250)
- [LoCoMo Benchmark — SNAP Research](https://snap-research.github.io/locomo/)

### Architecture & Patterns
- [Context Engineering for Personalization — OpenAI Cookbook](https://cookbook.openai.com/examples/agents_sdk/context_personalization)
- [Short-Term Memory Management with Sessions — OpenAI Cookbook](https://cookbook.openai.com/examples/agents_sdk/session_memory)
- [Design Patterns for Long-Term Memory in LLM-Powered Architectures — Serokell](https://serokell.io/blog/design-patterns-for-long-term-memory-in-llm-powered-architectures)
- [Memory for AI Agents: A New Paradigm — The New Stack](https://thenewstack.io/memory-for-ai-agents-a-new-paradigm-of-context-engineering/)
- [Building Smarter AI Agents: AgentCore Long-Term Memory Deep Dive — AWS](https://aws.amazon.com/blogs/machine-learning/building-smarter-ai-agents-agentcore-long-term-memory-deep-dive/)
- [The AI Memory Crisis: Why 62% of Your AI Agent's Memories Are Wrong](https://medium.com/@mohantaastha/the-ai-memory-crisis-why-62-of-your-ai-agents-memories-are-wrong-792d015b71a4)

### Implementation References
- [claude-mem: Hook Lifecycle](https://docs.claude-mem.ai/architecture/hooks)
- [viant/sqlite-vec (Pure Go)](https://github.com/viant/sqlite-vec)
- [asg017/sqlite-vec Go Bindings](https://github.com/asg017/sqlite-vec-go-bindings)
