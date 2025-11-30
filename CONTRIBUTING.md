# Contributing to xcode-build-mcp

Thank you for your interest in contributing to xcode-build-mcp! This document outlines our development workflow and guidelines.

## Branching Strategy

We use **GitHub Flow** — a simple, trunk-based workflow:

- `master` is always production-ready and deployable
- All changes are made through feature branches and Pull Requests
- PRs are squash-merged to maintain a clean, linear history

## Getting Started

### Prerequisites

- Go 1.24.4 or higher
- Xcode 14.0 or higher
- macOS 12.0 or higher

### Setup

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/xcode-build-mcp.git
cd xcode-build-mcp

# Add upstream remote
git remote add upstream https://github.com/jontolof/xcode-build-mcp.git

# Verify remotes
git remote -v
```

## Development Workflow

### 1. Create a Feature Branch

Always branch from an up-to-date `master`:

```bash
git checkout master
git pull upstream master
git checkout -b <type>/<description>
```

### Branch Naming Convention

Use these prefixes to categorize your work:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `feat/` | New features | `feat/add-code-signing` |
| `fix/` | Bug fixes | `fix/simulator-timeout` |
| `docs/` | Documentation | `docs/update-api-examples` |
| `chore/` | Maintenance | `chore/update-dependencies` |
| `refactor/` | Code restructuring | `refactor/extract-parser` |
| `test/` | Test improvements | `test/add-integration-tests` |
| `style/` | Code formatting | `style/fix-formatting` |

### 2. Make Your Changes

```bash
# Make changes to the code
# ...

# Ensure code is formatted
make fmt

# Run tests
make test

# Run linter
make lint
```

### 3. Commit Your Changes

Follow our commit message convention:

```
<type>: <description>
```

**Types:** `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `style`

**Rules:**
- Use present tense: "add feature" not "added feature"
- Keep under 50 characters
- No period at the end
- Be descriptive but concise

**Examples:**
```bash
git commit -m "feat: add watchOS simulator support"
git commit -m "fix: resolve timeout in boot sequence"
git commit -m "docs: update installation instructions"
```

### 4. Push and Create a Pull Request

```bash
git push -u origin <branch-name>
```

Then open a Pull Request on GitHub:

1. Go to the repository on GitHub
2. Click "Compare & pull request"
3. Fill out the PR template
4. Ensure CI checks pass

### 5. After Merge

Clean up your local branches:

```bash
git checkout master
git pull upstream master
git branch -d <branch-name>
```

## Pull Request Guidelines

### PR Title

Use the same format as commit messages:

```
<type>: <description>
```

### PR Description

Include:
- **Summary**: What does this PR do?
- **Motivation**: Why is this change needed?
- **Test plan**: How was this tested?

### Review Process

1. All PRs require CI checks to pass
2. PRs are squash-merged to maintain clean history
3. The squash commit message will use your PR title

## Code Standards

### Go Code

- Run `gofmt` before committing (or `make fmt`)
- Follow standard Go conventions
- Add tests for new functionality
- Maintain existing test coverage

### Testing

```bash
# Run all tests
make test

# Run tests with race detection
go test -race ./...

# Run tests with coverage
make test-coverage
```

### Documentation

- Update README.md if adding user-facing features
- Add ADRs for significant architectural decisions (see `docs/adr/`)
- Keep code comments focused on "why" not "what"

## Reporting Issues

### Bug Reports

Include:
- Go version (`go version`)
- macOS version
- Xcode version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

Include:
- Use case description
- Proposed solution (if any)
- Alternative approaches considered

## Questions?

- Open a [GitHub Discussion](https://github.com/jontolof/xcode-build-mcp/discussions)
- Check existing [Issues](https://github.com/jontolof/xcode-build-mcp/issues)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
