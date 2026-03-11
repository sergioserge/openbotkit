## When to use

### Recall (read)
- User asks about something they previously told you ("What's my dog's name?", "Who is Raj?")
- User asks you to check memories ("Check my memories for...", "Do you remember...")
- You need context about the user's preferences, relationships, or projects
- ALWAYS run `obk memory list` to check what's stored — the command returns ALL saved facts

### Save (write)
- User explicitly says "remember this", "note that", "don't forget that..."
- User directly introduces themselves ("My name is...", "I work at...")

## When NOT to save
- Don't proactively save facts the user didn't ask you to remember
- Don't save ephemeral or session-specific information
- Don't save information about code or technical solutions

## Commands

### List all memories
```bash
obk memory list
```

### List memories by category
```bash
obk memory list --category relationship
obk memory list --category project
```

### Save a memory
```bash
obk memory add "User prefers dark mode" --category preference
```

### Categories
- identity: name, location, profession, education
- preference: likes, dislikes, communication style
- relationship: people, pets, family
- project: what they're working on, goals, plans

### Examples
```bash
# Recall: list everything stored
obk memory list

# Recall: filter by category
obk memory list --category relationship

# Save new facts
obk memory add "User's name is Priyanshu" --category identity
obk memory add "User prefers Go over Python" --category preference
obk memory add "User's dog is named Max" --category relationship
obk memory add "User is building OpenBotKit, a personal assistant" --category project
```
