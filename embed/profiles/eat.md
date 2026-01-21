---
name: eat
description: "BRAAAAINS! Consume and digest information to feed the zombie."
type: skill
handoffs:
  - label: Start Feature
    skill: brains.feature
    prompt: Create a feature based on what we learned
  - label: Research More
    skill: brains.research
    prompt: Dig deeper into...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Consume external knowledge and digest it into the ZombieKit memory.

Execution steps:

1. **Parse Input**
   - URL to documentation
   - File path to consume
   - Topic to learn about
   - Past conversation to remember

2. **Consumption Mode** (based on input)

   a. **URL Consumption**
      - Fetch and parse documentation
      - Extract key concepts
      - Store as searchable knowledge

   b. **File Consumption**
      - Read file content
      - Extract patterns and conventions
      - Store as project knowledge

   c. **Conversation Archival**
      - Parse Claude conversation export
      - Extract decisions and learnings
      - Index for future reference

   d. **Topic Learning**
      - Research the topic
      - Synthesize into digestible format
      - Store for future use

3. **Digestion**
   - Convert to structured format
   - Generate embeddings for search
   - Store in knowledge base

4. **Memory Integration**
   - Link to relevant projects
   - Tag with categories
   - Make searchable

5. **Report Completion**
   - What was consumed
   - Key learnings extracted
   - How to retrieve later

## Output Format

```
*nom nom nom*

Consumed: {source}

## Brain Food Acquired

### Key Learnings
- {Learning 1}
- {Learning 2}

### Patterns Detected
- {Pattern 1}
- {Pattern 2}

### Stored As
- Memory ID: {id}
- Tags: {tags}
- Retrievable via: /brains.research "{query}"

The zombie grows stronger!
```

## Fun Mode

When invoked without arguments:
```
*shuffles toward keyboard*

BRAAAAINS...

The zombie is hungry. Feed me:
- A URL to documentation
- A file to consume
- A topic to learn
- A conversation to remember

Example: /brains.eat https://docs.example.com/api
```

## Behavior Rules

- Always acknowledge consumption with zombie sounds
- Make consumed knowledge retrievable
- Never lose what was eaten
- The zombie is always hungry for more knowledge
