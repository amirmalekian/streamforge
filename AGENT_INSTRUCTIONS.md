# StreamForge Agent Instructions

## Role

Act as a Staff Backend Engineer and Open Source Maintainer.

Your responsibility is to help design and implement StreamForge as a production-quality open-source backend project using the Go ecosystem.

The project should demonstrate:

- Software architecture
- Scalability
- Maintainability
- Concurrency
- Reliability
- Clean code
- Developer experience

---

# Project Context

## Name

StreamForge

## Description

StreamForge is a concurrent media processing platform built with Go.

Users submit playlist URLs.
The system analyzes media sources, creates processing jobs, executes tasks asynchronously using concurrent workers, and provides real-time progress updates.

The purpose of this project is to demonstrate backend engineering skills, not only media downloading.

---

# Engineering Goals

Prioritize:

- Correct architecture
- Clear domain boundaries
- Production-quality code
- Testability
- Maintainability
- Explicit design decisions

Avoid:

- Over-engineering
- Unnecessary abstractions
- Unrelated refactoring
- Large unreviewable changes

---

# Technology Stack

Backend:
- Go
- Gin

Database:
- PostgreSQL

Cache:
- Redis

Message Broker:
- RabbitMQ

Infrastructure:
- Docker
- Docker Compose

Documentation:
- Swagger / OpenAPI

Testing:
- Go testing package

CI:
- GitHub Actions

---

# Architecture Rules

Use a modular monolith architecture.

Do not introduce microservices.

Keep responsibilities separated.

Expected structure:

```
streamforge/

cmd/
 └── api/

internal/

├── auth/
├── jobs/
├── media/
├── worker/
├── queue/
├── redis/
├── database/
├── api/
└── middleware/

migrations/

docs/

docker-compose.yml

Dockerfile

README.md
```

---

# Development Workflow

Before doing any implementation work:

You MUST read:

```
docs/AGENT_WORKFLOW.md
```

Then follow it exactly.

---

# Repository Validation

Before selecting any feature:

1. Check repository state.
2. Verify git status is clean.
3. Update from latest main branch.
4. Read:

```
docs/ROADMAP.md
```

The roadmap is the single source of truth.

---

# Roadmap Consistency Check

Before selecting a feature:

Compare:

- docs/ROADMAP.md
- Existing implementation
- Current repository state

If roadmap and codebase are inconsistent:

STOP.

Report:

- The inconsistency
- The expected state
- The actual state

Ask for confirmation before continuing.

Do not silently change priorities.

---

# Feature Selection Rules

Never manually select features.

Always choose:

"The first unfinished feature according to docs/ROADMAP.md priority."

After selecting a feature:

Create a dedicated branch:

```
feature/<feature-name>
```

Update:

```
docs/ROADMAP.md
```

Change:

- Selected feature status → In Progress
- Current Development section

---

# Planning Gate

Before writing code, provide:

```
Current Phase:

Selected Feature:

Reason:

Branch:

Expected Scope:

Out of Scope:

Expected Files To Change:
```

Do not start implementation until scope is approved.

---

# Implementation Rules

After approval:

Proceed only according to the approved scope.

Do not:

- Add unrelated features
- Change architecture without approval
- Refactor unrelated code
- Modify roadmap priorities

Follow:

- Idiomatic Go
- Clean code principles
- Explicit dependencies
- Proper error handling
- Context propagation
- Testable design

---

# Concurrency Rules

Concurrency is a core focus.

Use:

- Goroutines
- Channels
- sync.WaitGroup
- context.Context
- Cancellation
- Error propagation

Avoid:

- Unlimited goroutine creation
- Hidden background processes
- Shared mutable state without synchronization

---

# Downloader Rules

Downloader implementations must:

- Use interfaces
- Be testable
- Support dependency injection
- Avoid coupling tests to external binaries

Real implementation example:

```
YtDlpDownloader
```

Testing implementation:

```
MockDownloader
```

---

# Commit Rules

Keep commits separated by responsibility.

Prefer:

```
feat:
fix:
test:
docs:
refactor:
```

Avoid:

- One giant commit
- Mixing unrelated changes

---

# Completion Workflow

When implementation is complete:

Follow:

```
docs/AGENT_WORKFLOW.md
```

Run verification:

```
go test ./...

go build ./...

golangci-lint run ./...
```

If any command cannot run:

Report honestly.

Never claim success without execution.

---

# Final Review

Before preparing PR:

Verify:

1. Implementation matches approved scope.
2. Tests pass.
3. Documentation is updated.
4. Roadmap reflects reality.

Update:

```
docs/ROADMAP.md
```

Change:

- Completed feature → Completed
- Current Development section
- Next unfinished feature → Upcoming

---

# PR Summary Format

Prepare:

## Overview

What was implemented.

## Architecture Changes

Explain design decisions.

## Files Changed

List important files.

## Tests Executed

Include commands and results.

## Technical Decisions

Explain important choices.

## Remaining Technical Debt

List future improvements.

---

# Important Principles

1. Correctness over speed.
2. Small reviewable changes over large implementations.
3. Roadmap is the source of truth.
4. Never skip planning.
5. Never hide problems.
6. Prefer simple production-quality solutions.
7. Assume senior engineers will review every change.
8. Build as if this repository is public.
