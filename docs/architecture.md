# StreamForge Architecture

## Overview

StreamForge is designed as a modular monolith backend system built with
Go.

The main goal is to process long-running media tasks asynchronously
while demonstrating Go concurrency patterns and scalable backend design.

------------------------------------------------------------------------

## Architecture Style

StreamForge follows a modular monolith architecture.

Each domain has clear boundaries:

-   Authentication
-   Job Management
-   Media Processing
-   Worker Processing
-   Queue Management
-   Cache Management

This approach keeps the system simple while allowing future service
extraction if needed.

------------------------------------------------------------------------

## System Flow

    Client
     |
    Gin API
     |
    Create Job
     |
    RabbitMQ Queue
     |
    Worker Pool
     |
    Media Processing
     |
    PostgreSQL + Redis
     |
    Realtime Updates

------------------------------------------------------------------------

## Components

### API Layer

Responsible for:

-   HTTP communication
-   Request validation
-   Authentication
-   Job creation

### Queue Layer

RabbitMQ handles asynchronous job distribution.

Benefits:

-   Decouples API from heavy processing
-   Enables retry mechanisms
-   Allows horizontal worker scaling

### Worker Pool

Workers process jobs concurrently using:

-   Goroutines
-   Channels
-   Context cancellation
-   Synchronization patterns

### Redis

Redis is used for:

-   Progress tracking
-   Temporary processing state
-   Rate limiting

### PostgreSQL

PostgreSQL stores:

-   Users
-   Jobs
-   Media items
-   Processing history

------------------------------------------------------------------------

## Design Principles

-   Clear separation of concerns
-   Maintainable code structure
-   Explicit dependencies
-   Idiomatic Go practices
-   Production-oriented design decisions

------------------------------------------------------------------------

## Future Improvements

-   Distributed workers
-   Kubernetes deployment
-   Prometheus metrics
-   OpenTelemetry tracing
-   Advanced monitoring
-   Service extraction when required

------------------------------------------------------------------------

## Architectural Decisions

### Why Go?

Go is selected because of its simplicity, performance, strong standard
library, and excellent support for concurrent programming.

### Why RabbitMQ?

RabbitMQ provides reliable asynchronous job processing and decouples API
requests from long-running operations.

### Why Redis?

Redis provides fast state management for real-time progress tracking and
temporary processing information.

### Why Modular Monolith?

The project avoids unnecessary microservice complexity while keeping
boundaries clear enough for future scalability.
