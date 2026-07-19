# StreamForge Agent Development Workflow

This document defines the required workflow for implementing any new feature in the StreamForge repository.

Every feature implementation must follow these steps.

---

# 1. Repository State Check

Before making any changes:

Check current git state:

```bash
git status
```

Check current branch:

```bash
git branch --show-current
```

Requirements:

- Working tree must be clean before starting a new feature.
- Never overwrite existing uncommitted changes.
- If uncommitted changes exist, report them before continuing.

---

# 2. Always Start From Latest Main

Every new feature must start from the latest `main` branch.

Execute:

```bash
git checkout main
git pull origin main
```

Verify:

```bash
git log --oneline -5
```

Never start feature development from:

- old feature branches
- roadmap branches
- temporary branches
- previous PR branches

---

# 3. Create Feature Branch

After updating main, create a dedicated feature branch.

Branch naming convention:

```
feature/<feature-name>
```

Examples:

```
feature/playlist-support
feature/progress-reporting
feature/job-retry-system
```

Create branch:

```bash
git checkout -b feature/<feature-name>
```

All feature work must happen only on this branch.

Never commit feature code directly to:

```
main
update-roadmap-*
chore/*
```

---

# 4. Understand Current Project State

Before implementation, read:

```
README.md
docs/ROADMAP.md
docs/
```

Understand:

- current architecture
- completed phases
- active development phase
- remaining features
- existing technical decisions

Do not implement already completed features.

---

# 5. Update ROADMAP.md

Before writing code, update:

```
docs/ROADMAP.md
```

Add the feature status.

Example:

```md
## Phase X

### Feature Name

Status:
🚧 In Progress

Started:
YYYY-MM-DD
```

After completing the feature:

```md
Status:
✅ Completed
```

The roadmap must always represent the real project state.

---

# 6. Define Feature Scope

Before coding, provide:

## Feature Goal

What problem this feature solves.

## Scope

What will be implemented.

## Out of Scope

What will intentionally not be changed.

## Expected Files

Which areas of the repository should change.

Avoid expanding scope without confirmation.

---

# 7. Implementation Rules

Follow existing architecture.

Requirements:

- Keep packages modular.
- Respect existing interfaces.
- Prefer dependency injection.
- Avoid unnecessary refactoring.
- Avoid unrelated cleanup.
- Do not change public APIs without reason.
- Do not introduce new dependencies without justification.
- Keep backward compatibility when possible.

---

# 8. Testing Requirements

Every feature must include tests.

Required coverage:

- normal behavior
- edge cases
- failure scenarios

Before committing run:

```bash
go test ./...
```

```bash
go build ./...
```

```bash
golangci-lint run ./...
```

If a command is unavailable:

- report it clearly
- never claim success

---

# 9. Commit Strategy

Never create one large commit containing unrelated changes.

Commits must be separated by responsibility.

Examples:

Feature implementation:

```
feat(component): implement feature
```

Bug fix:

```
fix(component): fix issue
```

Tests:

```
test(component): add feature tests
```

Documentation:

```
docs(roadmap): update feature status
```

Refactor:

```
refactor(component): improve structure
```

Each commit must be independently reviewable.

---

# 10. Pull Request Preparation

Before opening PR:

Check:

```bash
git status
```

```bash
git log --oneline --decorate --graph
```

Provide:

- Feature summary
- Architecture changes
- Files changed
- Tests executed
- Remaining technical debt
- Commit list

---

# 11. Pull Request Rules

PR title format:

```
<type>(<scope>): <description>
```

Examples:

```
feat(download): add playlist support
fix(worker): improve job execution flow
docs(roadmap): update phase status
```

PR description must include:

## Overview

What changed.

## Changes

Detailed implementation summary.

## Testing

Commands executed and results.

## Out of Scope

What was intentionally excluded.

## Follow-ups

Future improvements.

---

# 12. StreamForge Architecture Rules

Respect these boundaries:

## API Service

Responsible for:

- HTTP handling
- authentication
- validation
- creating jobs
- publishing messages

API should NOT:

- execute downloads
- run worker tasks
- process queue messages


## Worker Service

Responsible for:

- consuming queue messages
- executing jobs
- calling downloader
- updating job/media status

Worker should NOT:

- handle HTTP
- create business requests
- bypass queue flow


## Downloader Layer

Responsible for:

- downloading media
- metadata extraction
- external downloader integrations

Downloader should not:

- modify database directly
- manage jobs


---

# 13. Completion Checklist

Before considering a feature complete:

- [ ] Feature branch created from latest main
- [ ] ROADMAP.md updated
- [ ] Scope respected
- [ ] Tests added
- [ ] go test ./... passes
- [ ] go build ./... passes
- [ ] golangci-lint passes
- [ ] Commits separated by responsibility
- [ ] PR description prepared

---

Follow this workflow for every future StreamForge feature.