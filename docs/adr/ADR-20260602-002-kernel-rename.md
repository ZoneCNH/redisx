# ADR-20260602-002: default downstream name moves to kernel

## Status

Accepted.

## Decision

Use `kernel` as the default L0 downstream target name for rendered template examples and downstream synchronization records.

## Consequences

Documentation and generator examples should prefer `kernel` for the L0 downstream target. Historical names may remain only in migration context and must not be treated as the current release identity.
