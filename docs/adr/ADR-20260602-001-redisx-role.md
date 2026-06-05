# ADR-20260602-001: redisx repository role

## Status

Accepted.

## Decision

Use `redisx` (`https://github.com/ZoneCNH/redisx`) as the unified Standard Source, Go Reference Template, Generator, Harness, and Evidence Runtime repository.

## Consequences

Standard text, template code, generator behavior, Harness gates, and Evidence replay must evolve together. A change that affects one surface must be reviewed for downstream standard impact before release evidence is accepted.
