# Architectural Decision Records (ADRs)

This directory contains records of architectural decisions made in this project.

## What is an ADR?

An Architectural Decision Record (ADR) captures an important architectural decision made along with its context and consequences.

## When to Write an ADR

Write an ADR when you:
- Make a significant technical decision with multiple viable alternatives
- Choose between competing approaches or technologies
- Establish a pattern or standard for the project
- Make a decision that affects the project's architecture

## When NOT to Write an ADR

Don't write an ADR for:
- Obvious choices with no alternatives
- Temporary decisions or experiments
- Implementation details covered by code comments
- Decisions that can be easily reversed

## Format

All ADRs follow the template in `template.md`:
- **Context**: Why we needed to make a decision
- **Decision**: What we decided
- **Alternatives**: What we considered
- **Consequences**: What happens as a result

## Naming Convention

ADRs are numbered sequentially:
- `0001-crash-detection-architecture.md`
- `0002-output-filtering-strategy.md`
- `0003-mcp-response-size-limits.md`

**Important:** Use stable filenames. Version numbers and dates go **inside** the file, not in the filename.

## Status Lifecycle

- **Proposed**: Under discussion
- **Accepted**: Decision made and implemented
- **Deprecated**: No longer recommended but still in use
- **Superseded**: Replaced by a newer ADR

## Index of ADRs

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-crash-detection-architecture.md) | Crash Detection Architecture | Accepted | 2025-11-30 |
| [0002](0002-output-filtering-strategy.md) | Output Filtering Strategy | Accepted | 2025-11-30 |
| [0003](0003-mcp-response-size-limits.md) | MCP Response Size Limits | Accepted | 2025-11-30 |

## Contributing

1. Copy `template.md` to a new file with the next sequential number
2. Fill in all sections thoughtfully
3. Update this README's index
4. Commit with message: `docs: add ADR-XXXX [title]`

## References

- [ADR GitHub Organization](https://adr.github.io/)
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) - Original blog post by Michael Nygard
