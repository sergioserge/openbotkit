# Memory System Design

## Problem Statement

The current codebase conflates "memory" with "history." What we call `source/memory/` today is actually **conversation transcript capture** — it stores Claude Code session transcripts as `memory_conversations` + `memory_messages` tables. This is **history**, not memory.

**History** = a record of past conversations (who said what, when).

**Memory** = durable facts about the user — preferences, identity, relationships, context — that persist across conversations and personalize the assistant's behavior.

A personal assistant needs both, and they must be clearly separated.

---

## Research: How Production Systems Handle Memory

### ChatGPT (OpenAI)

ChatGPT's memory is a flat list of short factual sentences injected into every system prompt. Approximately 200-300 facts, capped at ~6,000 tokens.

**Architecture (4 layers in every prompt):**

1. **Session metadata** — environment info (local time, platform, account age)
2. **Saved memories** — free-text factual sentences (e.g., "User is vegetarian", "User prefers Python over JavaScript")
3. **Recent conversation summaries** — lightweight digests of ~40 recent chats (user messages only)
4. **Current conversation sliding window** — the actual message history

**Memory format:** One atomic fact per entry, plain text, not key-value pairs. Examples:
- "User's name is Simon Willison"
- "User is a software engineer who works primarily in Python"
- "User has a dog named Cleo"

**Extraction:** Two paths:
- **Explicit:** User says "Remember that I prefer dark mode" — triggers `bio` tool call
- **Implicit:** A classification model proactively saves facts it deems durable (preferences, biographical facts)

**User management:** View/delete in Settings > Personalization. No direct editing — must delete and re-tell. "Temporary Chat" mode skips memory entirely.

**Post-April 2025 update:** Memories organized under headings (Assistant Response Preferences, Notable Past Conversation Topics, Helpful User Insights, User Interaction Metadata). Recent/frequent topics rise to the top.

**Known failure modes:**
- Over-saving random things with no discernment
- Context collapse (work context bleeds into personal conversations)
- Context rot (stale preferences degrade response quality over time)
- Duplicate entries filling up the ~100-200 slot cap
- Memory wipe incidents (February 2025 backend failure)

### Gemini (Google)

Gemini stores memory as a single structured document called `user_context` with a fixed 4-section schema:

1. **Demographic information** — name, age, location, education, employment
2. **Interests and preferences** — technologies, recurring topics, long-term goals
3. **Relationships** — important people in the user's life
4. **Dated events, projects, and plans** — timestamped activities, ongoing projects

Each memory includes a factual claim + rationale pointing back to the source conversation and date. The schema is intentionally partitioned by volatility: demographics rarely change; dated events shift constantly.

**Key design insight:** By separating memories by update frequency, the system can selectively rewrite volatile sections without touching stable ones.

### MemGPT / Letta — The "Operating System" Paradigm

The most influential research design. Treats memory like a computer's memory hierarchy.

**Two-tier architecture:**

- **Tier 1: Core Memory (always in prompt, "RAM")**
  - Two writeable blocks: `persona` (who the agent is) and `human` (who the user is)
  - Each block defaults to 2,000 characters
  - The agent self-edits these blocks using function calls (`core_memory_append`, `core_memory_replace`)

- **Tier 2: External Memory ("Disk")**
  - **Recall Memory:** Complete conversation history, searchable via `conversation_search` (text) and `conversation_search_date` (date range)
  - **Archival Memory:** Infinite-size vector database for long-term knowledge, accessed via `archival_memory_insert` and `archival_memory_search`

**Key innovation:** The agent manages its own memory. It decides what to promote to core memory, what to archive, and what to retrieve.

### Mem0 — Production Memory Layer

The most popular open-source memory library (19k+ GitHub stars).

**Two-phase pipeline:**

1. **Extraction phase:** Given conversation messages + existing memory context, an LLM extracts candidate "atomic facts" as JSON. Default model: gpt-4.1-nano. Prompt instructs LLM to extract 7 categories: personal preferences, important personal details, plans/intentions, activity preferences, health/wellness, professional details, miscellaneous.

2. **Update phase:** Each candidate fact is embedded and compared via vector similarity search (top-K most similar existing memories). An LLM then chooses one of four operations:
   - **ADD** — new fact, no semantic equivalent exists
   - **UPDATE** — augment an existing memory with complementary info
   - **DELETE** — new info contradicts an existing memory
   - **NOOP** — fact already captured, no change needed

**Dual storage:** Vector store (embeddings for similarity search) + optional graph store (entities as nodes, relationships as edges).

### LangMem (LangChain)

**Three memory types:**
- **Semantic Memory** — facts, preferences, knowledge (like ChatGPT's saved memories)
- **Episodic Memory** — specific events/interactions with temporal context
- **Procedural Memory** — learned behaviors, prompt optimizations

Uses schema-based extraction: memories are structured objects defined by developer-provided schemas. Supports `enable_updates=True` for conflict resolution.

### OpenAI Agents SDK Cookbook Pattern

Three-stage pipeline:
1. **Memory injection** — at session start, inject relevant state into context
2. **Memory distillation** — during conversation, a tool captures session notes
3. **Memory consolidation** — at session end, merge session notes into global memory with deduplication

Uses dual format: structured keys (predefined fields) + unstructured notes (free-form text).

---

## Research: Memory Extraction Approaches

### LLM-Based Extraction

All production systems use LLM-based extraction. The task is relatively constrained (structured output, clear categories), so cheap/small models work well.

**Mem0's pipeline:** Two LLM calls per conversation:
1. Extract candidate facts (JSON output)
2. Reconcile each fact against existing memories (ADD/UPDATE/DELETE/NOOP)

**Key finding:** Mem0's entire benchmark suite (26% improvement over OpenAI's memory) ran on GPT-4o-mini. Their current default is gpt-4.1-nano. Extraction quality is sufficient with cheap models.

### Small/Cheap Model Quality

Research confirms small models are sufficient for memory extraction:

- Mem0 defaults to gpt-4.1-nano ($0.10/$0.40 per 1M tokens)
- A March 2026 paper ("Beyond the Context Window") used GPT-5-nano at $0.0435 per conversation
- The paper found that **retrieval method is the dominant factor** — accuracy spans 20 points across retrieval methods but only 3-8 points across write strategies

This means **how you search memories matters more than how you extract them**, validating cheap extraction models.

### Deterministic/Rule-Based Extraction

Rule-based approaches (regex, NER, pattern matching) catch only **20-30% of extractable facts** — specifically explicit patterns like "My name is X", "I prefer Y", "I work at Z."

They miss the most valuable implicit information:
- "I had sushi again last night" (implies they like sushi)
- Context-dependent facts spread across multiple turns
- Negation and nuance ("I used to like X but not anymore")

### Cost Analysis

Assumptions: 50 messages per conversation (~5,000 tokens), 5 conversations per day.

| Model | Per Conversation | Monthly (150 conv) |
|---|---|---|
| GPT-5-nano ($0.05/$0.40) | $0.0007 | $0.105 |
| GPT-4.1-nano ($0.10/$0.40) | $0.0011 | $0.165 |
| GPT-4o-mini ($0.15/$0.60) | $0.0016 | $0.240 |
| Claude Haiku 4.5 ($0.25/$1.25) | $0.0029 | $0.435 |

**Cost is negligible.** Under $0.50/month even with the most expensive option.

---

## Research: Extraction From Different Source Types

### Conversations (User-Assistant Chat) — HIGH VALUE
- Signal-to-noise: High. Users explicitly state preferences, facts, goals.
- Extraction difficulty: Easy. Information is directed at the assistant.
- Priority: Ship first.

### Notes — HIGH VALUE, DENSE
- Signal-to-noise: High. Everything in a note was important enough for the user to write down.
- Extraction difficulty: Medium. Notes can be fragmentary or use shorthand.
- Priority: Ship second.

### Emails — MEDIUM-HIGH VALUE, STRUCTURED
- Signal-to-noise: Medium. Professional emails have clear topics but lots of boilerplate.
- Extraction difficulty: Medium. Emails are structured (subject, sender, recipient).
- Recommendation: Focus on sent emails (user's own words). Pre-filter by importance.
- Priority: Ship third.

### WhatsApp — MEDIUM VALUE, HIGH NOISE
- Signal-to-noise: Low-to-medium. Most messages are coordination ("ok", "coming", "lol").
- Extraction difficulty: Hard. Requires conversation-level context, not message-level.
- Recommendation: Process in conversation chunks (20-50 messages), not individual messages.
- Priority: Ship last.

---

## Research: Storage and Retrieval Strategies

### Inject-All Approach (ChatGPT Style)

All memories stuffed into system prompt every time. No retrieval needed.

**Token overhead:**

| Memory count | Estimated tokens | % of 200K context | Per-query cost (Sonnet @ $3/M input) |
|---|---|---|---|
| 50 facts | ~1,000 | 0.5% | $0.003 |
| 200 facts | ~4,000 | 2% | $0.012 |
| 500 facts | ~10,000 | 5% | $0.030 |

**Where it breaks down:** Research on the "lost in the middle" effect shows LLMs are more likely to recall information at the beginning or end of long prompts. Facts buried in the middle of 500+ memories may be effectively invisible.

**ChatGPT caps at ~200-300 memories (~6,000 tokens)**, suggesting OpenAI found a practical ceiling there.

### Embedding-Based Retrieval (RAG Style)

Store memories as vector embeddings, retrieve top-K relevant ones per query.

**Fundamental problem for personal memory:** RAG retrieves by semantic similarity, but personal facts are often needed for reasons that aren't semantically obvious. "Recommend a movie" should surface "I prefer sci-fi," but "movie" and "sci-fi preference" may not have high cosine similarity.

**Vector DB options for local-first SQLite:**

| Option | Go Compatibility | CGO Required? |
|---|---|---|
| viant/sqlite-vec | Works with modernc.org/sqlite | No (pure Go) |
| asg017/sqlite-vec (CGO) | mattn/go-sqlite3 only | Yes |
| asg017/sqlite-vec (WASM) | ncruces/go-sqlite3 only | No |

`viant/sqlite-vec` is the only option compatible with our existing `modernc.org/sqlite` driver without CGO.

### Scaling Thresholds

- **Under 100 memories (~2K tokens):** Inject-all is optimal. No quality degradation.
- **100-300 memories (~2K-6K tokens):** Inject-all still works well. ChatGPT's cap lives here.
- **300-500 memories (~6K-10K tokens):** Lost-in-the-middle effects begin. Consider tiered approach.
- **500+ memories (~10K+ tokens):** Retrieval becomes necessary.

---

## Research: Memory Reconciliation

### Mem0's Reconciliation Pipeline

For each extracted fact:
1. Embed the fact using embedding model
2. Search vector store for top-5 semantically similar existing memories
3. Send all retrieved old memories + all new facts to LLM
4. LLM returns JSON with `event` field: ADD, UPDATE, DELETE, or NONE

Rules from the prompt:
- **ADD:** New fact, no semantic equivalent. LLM generates a new ID.
- **UPDATE:** Same topic but new info is richer/different. Keep same ID.
- **DELETE:** New info contradicts an existing memory.
- **NONE:** Already present or same meaning (e.g., "Likes cheese pizza" vs "Loves cheese pizza").

### Failure Modes (Independent Evaluation)

- 53% accuracy on correct answers (no system exceeds 55%)
- Best recall only 43% — more than half of important memories lost
- Update accuracy below 26% — memory evolution is fundamentally broken
- Degradation with conversation length (29.69% accuracy on medium, 0.92% on long)
- Basic RAG on raw chunks (77.9%) matches or exceeds Mem0-style extracted facts

### Memory Categories

Gemini's 4-category approach is the most elegant:
- Categories map to natural update frequencies
- Demographics = rarely change, events = change constantly
- Enables selective rewriting without full re-summarization

### Deduplication

Two-stage approach: embedding-based candidate detection (cosine similarity > 0.7-0.8 threshold), then LLM-based confirmation picking the richer/more accurate version. Running dedup as batch consolidation is more reliable than per-fact inline dedup.

### Memory Quality Guidelines

A good memory statement is:
- 1-2 sentences max, short and specific
- Durable (likely true across sessions)
- Actionable (changes future responses)
- Explicit (stated or confirmed by user, not inferred)
- Normalized (use "User prefers X", not "User said they prefer X")

### Forgetting / Decay

No production system implements automatic decay. MemoryBank (research) uses Ebbinghaus forgetting curves, but Mem0, ChatGPT, and Gemini all skip this. Automatic decay is risky — you might delete "User is allergic to peanuts" because it hasn't been accessed recently.

**Practical approach:** Store `created_at` and `last_accessed_at` timestamps. Use recency as a retrieval boost, not a deletion signal. Let users explicitly forget. Periodically consolidate to merge related memories and prune duplicates.

---

## Research: Mid-Conversation vs Post-Conversation Memory

### The "Remember This" Moment

When a user says "remember that I prefer dark mode," all production systems agree: **immediate save + visible confirmation.** Deferred saving feels unreliable.

### Proactive vs Reactive Saving

**Proactive (ChatGPT's approach):** Saves facts without being asked. Well-documented failure modes:
- Saves wrong things, over-indexes on casual mentions
- Creates context rot over time
- Feels creepy/surveillance-like
- Leads users to turn the feature off entirely

**Reactive only:** Only saves when explicitly asked. Predictable, no surprise saves. Misses implicit facts.

**Emerging consensus (OpenAI Agents SDK Cookbook):** Hybrid approach:
1. During conversation: reactive tool for explicit saves
2. After conversation: post-session distillation for implicit facts

### Duplicate Handling Between Both Paths

ChatGPT's duplicate problem is severe and well-documented (same memory duplicated dozens of times). Mem0 solves this with vector-similarity-based dedup at reconciliation time.

**If you have both mid-conversation saves AND post-conversation extraction, you MUST have a deduplication/merge step.**

---

## Design Decisions

### 1. Memory Sources — Extract From All, But Phased

Extract from all sources, prioritized by signal-to-noise ratio:
1. **Conversations (history)** — ship first
2. **Notes** — ship second
3. **Emails** — ship third
4. **WhatsApp** — ship last

Cost is negligible ($0.07-$0.44/month). The constraint is engineering effort, not money.

### 2. Extraction Approach — Cheap LLM, Not Deterministic

Use the configured "fast" model from the user's existing model config. Pre-filter trivial messages. Two LLM calls: extract facts, then reconcile against existing memories.

Rule-based approaches catch only 20-30% of facts. The cost difference between cheap and expensive models is pennies/month. Quality matters more.

### 3. Storage — Inject-All Now, Hybrid Later

Start with inject-all (all memories in system prompt). For a single user, this will stay under 200 memories for months. Inject-all has **100% recall** — no retrieval misses.

Schema includes an optional `embedding` column from day one for future vector search via `viant/sqlite-vec` (pure Go, works with modernc.org/sqlite).

Upgrade path: at 300+ memories, add tiered injection (core facts always + retrieval for the rest).

### 4. Memory Categories — Lightweight Fixed Set

Four categories inspired by Gemini, mapped to natural update frequencies:

| Category | Examples | Update Frequency |
|---|---|---|
| `identity` | Name, location, profession, education | Rarely |
| `preference` | Likes dark mode, prefers Go, vegetarian | Occasionally |
| `relationship` | Wife is Sarah, dog named Max, works with Bob | Occasionally |
| `project` | Building OpenBotKit, learning Rust, trip to Japan | Frequently |

### 5. Extraction Model — The "Fast" Model From Config

Use the user's configured fast model. Cost is negligible regardless of tier. This respects the existing model routing system.

### 6. Automatic Extraction — Yes, Async After History Capture

The hook command:
```
obk history capture && obk memory extract &
```

History capture is fast (JSONL parsing, <100ms). Memory extraction runs in the background with LLM calls (~5-10 seconds). The `&` backgrounds it so the hook returns immediately.

### 7. Mid-Conversation Save — Reactive Only, Never Proactive

The agent gets a `save_memory` tool for explicit "remember this" requests. It does NOT proactively save facts it notices — that's what post-conversation extraction is for.

Two complementary paths:

| Path | When | What It Catches | Speed |
|---|---|---|---|
| `save_memory` tool | Mid-conversation | Explicit "remember this" | Instant + confirmation |
| `obk memory extract` | Post-conversation | Implicit facts, preferences | Background, ~10s |

Post-conversation extraction reconciles against all existing memories (including mid-conversation saves), preventing duplicates.

---

## Architecture

### Memory Is NOT a Source

Sources are external data ingestors (Gmail, WhatsApp, Notes, History). Memory is a **core system feature** that:
1. Extracts facts **from** sources
2. Stores them as structured facts
3. Injects them into the agent's context

Memory lives as its own top-level package, not under `source/`.

### Package Layout

```
memory/                    # Core memory system
  types.go                 # Memory struct, Category enum, LLM interface
  schema.go                # DB tables (SQLite + Postgres)
  store.go                 # DB operations (CRUD + search)
  extract.go               # LLM-based fact extraction from text
  reconcile.go             # ADD/UPDATE/DELETE/NOOP against existing memories
  format.go                # FormatForPrompt — markdown for system prompt
  *_test.go

source/history/            # Renamed from source/memory/
  history.go               # Source interface (was memory.go)
  types.go                 # Conversation, Message types
  schema.go                # history_conversations, history_messages tables
  store.go                 # DB operations
  capture.go               # Transcript parsing
  *_test.go

internal/cli/memory/       # CLI commands for memory
  memory.go                # obk memory (parent)
  list.go                  # obk memory list
  add.go                   # obk memory add "fact"
  delete.go                # obk memory delete <id>
  extract.go               # obk memory extract

internal/cli/history/      # CLI commands for history (renamed)
  history.go               # obk history (parent)
  capture.go               # obk history capture

skills/memory-save/        # Save memory skill for the agent
skills/history-read/       # Renamed from skills/memory-read/
```

### Data Model

```go
type Memory struct {
    ID        int64
    Content   string    // Atomic fact: "User's name is Priyanshu"
    Category  Category  // identity, preference, relationship, project
    Source    string    // "history", "whatsapp", "gmail", "applenotes", "manual"
    SourceRef string    // Optional: session_id, message_id, etc.
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS memories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    content     TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'preference',
    source      TEXT NOT NULL DEFAULT 'manual',
    source_ref  TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_memories_category ON memories(category);
CREATE INDEX idx_memories_source ON memories(source);
```

### Extraction Pipeline

```
Conversation ends (hook triggers obk history capture)
    |
    v
obk memory extract (runs in background)
    |
    v
1. Load recent history messages from DB
    |
    v
2. Pre-filter: skip trivial messages (<5 words, acks, greetings)
    |
    v
3. LLM call (fast model): "Extract personal facts about the user"
   Returns: [{"content": "...", "category": "..."}]
    |
    v
4. For each candidate fact:
   a. Text-search existing memories for similar content
   b. LLM call (fast model): Given existing + new, decide ADD/UPDATE/DELETE/NOOP
    |
    v
5. Execute decisions against DB
    |
    v
6. Return ExtractResult{Added: 3, Updated: 1, Deleted: 0, Skipped: 2}
```

### Agent Integration

Memories injected into system prompt at conversation start:

```
## About the user

### Identity
- Name is Priyanshu
- Based in [city]
- Software engineer

### Preferences
- Prefers Go over Python
- Likes concise responses

### Relationships
- [none yet]

### Projects & Context
- Building OpenBotKit, a personal assistant CLI tool
- Uses Claude Code for development
```

### CLI Commands

```
obk memory list                      # Show all memories, grouped by category
obk memory list --category identity  # Filter by category
obk memory add "fact" --category identity
obk memory delete 42
obk memory extract                   # Run extraction on recent history

obk history capture                  # Capture conversation transcript (renamed)
```

### Rename: memory -> history

| Component | Before | After |
|---|---|---|
| Package | `source/memory/` | `source/history/` |
| Tables | `memory_conversations`, `memory_messages` | `history_conversations`, `history_messages` |
| CLI | `obk memory capture` | `obk history capture` |
| Skill | `skills/memory-read/` | `skills/history-read/` |
| Config | `config.Memory` | `config.History` |
| Source name | `"memory"` | `"history"` |

---

## Cross-System Comparison

| Dimension | ChatGPT | Gemini | MemGPT/Letta | Mem0 | OpenBotKit |
|---|---|---|---|---|---|
| Storage format | Flat list of sentences | Structured document with 4 sections | Blocks + vector DB | Vector embeddings + graph | SQLite table with categories |
| Extraction | Rules + classifier | LLM inference | Agent self-decides | LLM two-phase | Fast LLM two-phase |
| Injection | All memories every prompt | Relevant subset | Core always; archival on-demand | Retrieved via similarity | All memories (phase 1) |
| Conflict resolution | Model replaces via tool | Overwrite by section | Agent self-edits | LLM: ADD/UPDATE/DELETE/NOOP | LLM: ADD/UPDATE/DELETE/NOOP |
| User management | View/delete in settings | Limited visibility | Full API control | Full API | CLI: list/add/delete |
| Capacity | ~200-300 facts (~6K tokens) | Single document | Core: 2K chars; Archival: unlimited | Unlimited (vector DB) | ~200 facts (phase 1), unlimited (phase 2) |
| Mid-conversation saves | Yes (proactive + reactive) | No | Yes (agent self-edits) | Via API | Yes (reactive only) |

---

## References

### ChatGPT Memory
- [Memory and new controls for ChatGPT — OpenAI](https://openai.com/index/memory-and-new-controls-for-chatgpt/)
- [How ChatGPT Memory Works, Reverse Engineered — LLMRefs](https://llmrefs.com/blog/reverse-engineering-chatgpt-memory)
- [I Reverse Engineered ChatGPT's Memory System — Manthan Gupta](https://manthanguptaa.in/posts/chatgpt_memory/)
- [Reverse Engineering ChatGPT's Updated Memory System — Julian Fleck](https://medium.com/@j0lian/reverse-engineering-chatgpts-updated-memory-system-3cb9e82e5d21)
- [I really don't like ChatGPT's new memory dossier — Simon Willison](https://simonwillison.net/2025/May/21/chatgpt-new-memory/)
- [Comparing the memory implementations of Claude and ChatGPT — Simon Willison](https://simonwillison.net/2025/Sep/12/claude-memory/)
- [How ChatGPT Remembers You — Embrace The Red](https://embracethered.com/blog/posts/2025/chatgpt-how-does-chat-history-memory-preferences-work/)
- [Why I Turned Off ChatGPT's Memory — Every.to](https://every.to/also-true-for-humans/why-i-turned-off-chatgpt-s-memory)
- [Memory FAQ — OpenAI Help Center](https://help.openai.com/en/articles/8590148-memory-faq)
- [What is Memory? — OpenAI Help Center](https://help.openai.com/en/articles/8983136-what-is-memory)
- [Inside ChatGPT's Memory — AI Monks](https://medium.com/aimonks/inside-chatgpts-memory-how-the-most-sophisticated-memory-system-in-ai-really-works-f2b3f32d86b3)
- [ChatGPT Memory Issues: Duplicates — OpenAI Community](https://community.openai.com/t/instead-of-saving-a-new-entry-into-memory-chatgpt-instead-repeatedly-duplicates-a-few-of-the-latest-memories-instead-fix-please/1103489)
- [ChatGPT's Fading Recall: Inside the 2025 Memory Wipe Crisis — WebProNews](https://www.webpronews.com/chatgpts-fading-recall-inside-the-2025-memory-wipe-crisis/)

### Gemini Memory
- [Google Has Your Data. Gemini Barely Uses It. — Shlok Khemani](https://www.shloked.com/writing/gemini-memory)
- [Gemini's New Memory Feature Update — Kai](https://kaiwritesornot.medium.com/geminis-new-memory-feature-update-58c2872689a6)
- [Memory for AI Code Reviews using Gemini Code Assist — Google Cloud](https://cloud.google.com/blog/products/ai-machine-learning/memory-for-ai-code-reviews-using-gemini-code-assist)

### MemGPT / Letta
- [MemGPT: Towards LLMs as Operating Systems — arXiv](https://arxiv.org/abs/2310.08560)
- [Intro to Letta — Letta Docs](https://docs.letta.com/concepts/memgpt/)
- [Agent Memory: How to Build Agents that Learn and Remember — Letta](https://www.letta.com/blog/agent-memory)
- [RAG is not Agent Memory — Letta](https://www.letta.com/blog/rag-vs-agent-memory)

### Mem0
- [Mem0 GitHub Repository](https://github.com/mem0ai/mem0)
- [Mem0: Building Production-Ready AI Agents with Scalable Long-Term Memory — arXiv](https://arxiv.org/abs/2504.19413)
- [Mem0: How Three Prompts Created a Viral AI Memory Layer](https://blog.lqhl.me/mem0-how-three-prompts-created-a-viral-ai-memory-layer)
- [Mem0 Benchmark: OpenAI Memory vs LangMem vs MemGPT vs Mem0](https://mem0.ai/blog/benchmarked-openai-memory-vs-langmem-vs-memgpt-vs-mem0-for-long-term-memory-here-s-how-they-stacked-up)
- [Graph Memory — Mem0 Docs](https://docs.mem0.ai/open-source/features/graph-memory)
- [RAG vs AI Memory — Mem0](https://mem0.ai/blog/rag-vs-ai-memory)

### LangMem
- [LangMem Conceptual Guide](https://langchain-ai.github.io/langmem/concepts/conceptual_guide/)
- [How to Extract Semantic Memories — LangMem](https://langchain-ai.github.io/langmem/guides/extract_semantic_memories/)

### OpenAI Agents SDK
- [Context Engineering for Personalization — OpenAI Cookbook](https://cookbook.openai.com/examples/agents_sdk/context_personalization)
- [Short-Term Memory Management with Sessions — OpenAI Cookbook](https://cookbook.openai.com/examples/agents_sdk/session_memory)

### Research Papers
- [Beyond the Context Window: Fact-Based Memory vs Long-Context LLMs — arXiv](https://arxiv.org/html/2603.04814)
- [MemoryBank: Enhancing Large Language Models with Long-Term Memory — arXiv](https://arxiv.org/abs/2305.10250)
- [LoCoMo Benchmark — SNAP Research](https://snap-research.github.io/locomo/)

### Memory Architecture Patterns
- [Design Patterns for Long-Term Memory in LLM-Powered Architectures — Serokell](https://serokell.io/blog/design-patterns-for-long-term-memory-in-llm-powered-architectures)
- [Memory for AI Agents: A New Paradigm — The New Stack](https://thenewstack.io/memory-for-ai-agents-a-new-paradigm-of-context-engineering/)
- [Building Smarter AI Agents: AgentCore Long-Term Memory Deep Dive — AWS](https://aws.amazon.com/blogs/machine-learning/building-smarter-ai-agents-agentcore-long-term-memory-deep-dive/)
- [A Comprehensive Review of the Best AI Memory Systems — Pieces](https://pieces.app/blog/best-ai-memory-systems)
- [The AI Memory Crisis: Why 62% of Your AI Agent's Memories Are Wrong](https://medium.com/@mohantaastha/the-ai-memory-crisis-why-62-of-your-ai-agents-memories-are-wrong-792d015b71a4)

### Implementation References
- [claude-mem: Hook Lifecycle](https://docs.claude-mem.ai/architecture/hooks)
- [claude-mem GitHub](https://github.com/thedotmack/claude-mem)
- [viant/sqlite-vec (Pure Go)](https://github.com/viant/sqlite-vec)
- [asg017/sqlite-vec Go Bindings](https://github.com/asg017/sqlite-vec-go-bindings)
- [Anthropic Embeddings Documentation](https://platform.claude.com/docs/en/build-with-claude/embeddings)

### Pricing / Cost
- [LLM API Pricing Comparison 2026 — IntuitionLabs](https://intuitionlabs.ai/articles/ai-api-pricing-comparison-grok-gemini-openai-claude)
- [LLM Pricing Comparison 2026 — CloudIDR](https://www.cloudidr.com/blog/llm-pricing-comparison-2026)
- [OpenAI Embeddings Pricing — CostGoat](https://costgoat.com/pricing/openai-embeddings)
