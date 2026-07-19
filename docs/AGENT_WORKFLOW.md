# StreamForge Agent Development Workflow

This document defines the required workflow for implementing any new feature in the StreamForge repository.

Every feature implementation must follow these steps strictly.

---

# 0. Determine Next Feature

Before starting any development work, identify the next feature from the project roadmap.

The single source of truth for feature priority is:

```
docs/ROADMAP.md
```

Process:

1. Read:

```
docs/ROADMAP.md
```

2. Identify:

- Current active phase
- Completed features
- Remaining features
- Feature order

3. Select the first unfinished feature according to roadmap order.

Example:

```md
Phase 3 — Download Engine

[x] Downloader abstraction
[x] yt-dlp Adapter
[x] Async execution pipeline

[ ] Playlist Support
[ ] Progress Reporting
```

The next feature is:

```
Playlist Support
```

Before implementation, report:

```
Current Phase:
Phase X

Selected Feature:
Feature Name

Reason:
This is the next incomplete item in docs/ROADMAP.md

Branch:
feature/<feature-name>
```

Do not start implementation if:

- roadmap priority is unclear
- feature dependencies are missing
- multiple features have equal priority

Ask for clarification first.

---

# 1. Repository State Check

Before making changes:

Check git state:

```bash
git status
```

Check current branch:

```bash
git branch --show-current
```

Requirements:

- Working tree must be clean.
- Never overwrite existing changes.
- Report uncommitted changes before continuing.

---

# 2. Start From Latest Main

Every new feature must start from the latest `main`.

Execute:

```bash
git checkout main
git pull origin main
```

Verify:

```bash
git log --oneline -5
```

Never start development from:

- old feature branches
- previous PR branches
- roadmap branches
- temporary branches

---

# 3. Create Feature Branch

Branch naming:

```
feature/<feature-name>
```

Examples:

```
feature/playlist-support
feature/progress-reporting
feature/job-retry-system
```

Create:

```bash
git checkout -b feature/<feature-name>
```

All implementation must happen only on this branch.

---

# 4. Understand Current Project State

Before coding read:

```
README.md
docs/ROADMAP.md
docs/
```

Understand:

- architecture
- current phase
- completed work
- remaining work
- existing constraints

Do not duplicate completed features.

---

# 5. Update ROADMAP Before Development

Before implementation update:

```
docs/ROADMAP.md
```

Mark the selected feature:

Example:

```md
### Playlist Support

Status:
🚧 In Progress

Started:
YYYY-MM-DD
```

After completion:

```md
Status:
✅ Completed
```

ROADMAP.md must always represent the real project state.

---

# 6. Define Feature Scope

Before coding provide:

## Goal

What problem this feature solves.

## Scope

What will be implemented.

## Out of Scope

What will not be changed.

## Expected Files

Which areas should change.

Do not expand scope without approval.

---

# 7. Implementation Rules

Follow existing architecture.

Rules:

- Keep packages modular.
- Respect interfaces.
- Prefer dependency injection.
- Avoid unrelated refactoring.
- Avoid unnecessary dependencies.
- Do not modify database schema unless required.
- Keep changes reviewable.

---

# 8. Architecture Boundaries

## API

Responsible for:

- HTTP handling
- authentication
- validation
- job creation
- queue publishing

API should NOT:

- execute downloads
- process queue messages
- run worker tasks


## Worker

Responsible for:

- consuming queue messages
- executing jobs
- calling downloader
- updating job state

Worker should NOT:

- handle HTTP
- create user requests
- bypass queue flow


## Downloader

Responsible for:

- media downloading
- metadata extraction
- external downloader integrations

Downloader should NOT:

- access database directly
- manage jobs

---

# 9. Testing Requirements

Every feature requires tests.

Cover:

- normal behavior
- edge cases
- failures

Run:

```bash
go test ./...
```

```bash
go build ./...
```

```bash
golangci-lint run ./...
```

If unavailable:

- report it
- do not claim success

---

# 10. Commit Strategy

Never create one huge commit.

Separate by responsibility.

Examples:

Feature:

```
feat(component): implement feature
```

Tests:

```
test(component): add tests
```

Documentation:

```
docs(roadmap): update status
```

Fix:

```
fix(component): fix issue
```

Refactor:

```
refactor(component): improve structure
```

---

# 11. Pull Request Preparation

Before PR:

Check:

```bash
git status
```

```bash
git log --oneline --decorate --graph
```

Provide:

- Summary
- Architecture changes
- Files changed
- Tests
- Technical debt
- Commit list

---

# 12. Completion Checklist

Before marking feature complete:

- [ ] Started from latest main
- [ ] Feature branch created
- [ ] ROADMAP updated
- [ ] Scope respected
- [ ] Tests added
- [ ] go test passes
- [ ] go build passes
- [ ] golangci-lint checked
- [ ] Commits separated
- [ ] PR prepared

Follow this workflow for every StreamForge feature.