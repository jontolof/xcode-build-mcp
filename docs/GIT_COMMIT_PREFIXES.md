# Git Commit Message Prefixes Reference

## Core Types

- **feat**: New feature or functionality
- **fix**: Bug fix
- **docs**: Documentation only changes
- **style**: Code style changes (formatting, missing semicolons, etc.) - no logic changes
- **refactor**: Code restructuring without changing functionality
- **perf**: Performance improvements
- **test**: Adding or modifying tests
- **chore**: Maintenance tasks, dependency updates, configs
- **build**: Changes to build system or external dependencies
- **ci**: Changes to CI/CD configuration files and scripts

## Additional Common Types

- **revert**: Reverting a previous commit
- **wip**: Work in progress (avoid in main branch)
- **hotfix**: Emergency fix for production
- **release**: Version release commits
- **deps**: Dependency updates (alternative to chore)
- **security**: Security fixes or improvements
- **config**: Configuration file changes
- **init**: Initial commit or project setup
- **merge**: Merge commits (often automated)
- **cleanup**: Code cleanup without logic changes

## Breaking Changes

Add `!` after prefix for breaking changes:
- **feat!**: Breaking feature change
- **fix!**: Breaking bug fix
- **refactor!**: Breaking refactor

## Scope Examples

Optional scope in parentheses:
- `feat(auth): Add OAuth2 support`
- `fix(recording): Resolve audio capture crash`
- `test(viewmodel): Add RecordingViewModel tests`
- `docs(api): Update API documentation`

## Example Commit Messages

```
feat: add voice memo recording with hold gesture
fix: resolve test regression in RecordingPerformanceTests
refactor: simplify agent naming to kebab-case
docs: update framework deployment instructions
test: add coverage for queue persistence
chore: update SwiftLint configuration
perf: optimize RecordingView rendering
style: format code according to SwiftFormat rules
build: upgrade to Xcode 15.2 toolchain
ci: add automated test runs on PR
```

## Best Practices

1. Use present tense ("add" not "added")
2. Keep first line under 50 characters
3. Capitalize first word after prefix
4. No period at end of subject line
5. Use scope for clarity when helpful
6. Add body for complex changes (blank line after subject)

## This standardization helps with:
- Automated changelog generation
- Clear git history
- Easy filtering of commit types
- Semantic versioning decisions