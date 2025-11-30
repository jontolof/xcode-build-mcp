# Documentation

This directory contains architectural decision records and supporting documentation for the xcode-build-mcp project.

## Documentation Philosophy

We follow the **Three Document Approach**:

1. **[CHANGELOG.md](../CHANGELOG.md)** - What changed (per release)
2. **[ADRs](adr/)** - Why we decided (architectural decisions)
3. **`git log`** - When it happened (automatic history)

Everything else is either in the code or can be recovered from git history.

## Structure

```
docs/
├── README.md           # This file
└── adr/                # Architectural Decision Records
    ├── README.md       # ADR index and guidelines
    ├── template.md     # Template for new ADRs
    └── 000X-*.md       # Individual ADRs
```

## Where to Find What

| Looking For | Location |
|-------------|----------|
| **Project overview** | [../README.md](../README.md) |
| **AI assistant guidelines** | [../CLAUDE.md](../CLAUDE.md) |
| **Version history** | [../CHANGELOG.md](../CHANGELOG.md) |
| **Why we made decisions** | [adr/](adr/) |
| **When changes happened** | `git log` |
| **How code works** | Code comments + tests |

## Principles

- **Delete over archive** - Trust git to preserve history
- **Stable filenames** - Versions inside files, not in names
- **Living documentation** - Update or delete, don't create snapshots
- **Decision records** - Capture why, not just what

## Contributing Documentation

- **Fixing docs**: Just update and commit
- **New ADR**: Follow [adr/README.md](adr/README.md)
- **Release notes**: Update [../CHANGELOG.md](../CHANGELOG.md)
