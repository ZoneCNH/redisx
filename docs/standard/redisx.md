# redisx L2 Standard

`redisx` is the standard authority source for `https://github.com/ZoneCNH/redisx` and the Go module `github.com/ZoneCNH/redisx`. This repository owns the redisx standard text, the generated Go library template, the generator, Harness gates, and Evidence replay contracts for the L2 Redis standard factory.

## Repository role

This repository is the redisx Standard Source and 模板 authority. It must keep the Redis-facing API, contracts, template output, generator behavior, Harness checks, and Evidence artifacts synchronized before any downstream adoption is marked complete.

## Layer boundary

- Public package: `pkg/redisx`.
- Provider implementations belong behind `internal/provider`; a go-redis implementation belongs under `internal/provider/goredis`.
- Public APIs must not expose go-redis concrete types or import business/private packages.
- L2 redisx must not encode business key policy, application cache strategy, business schemas, or runtime secret paths.

## Target contracts

The L2 Redis factory target includes:

- explicit configuration through `redisx.Options` or an approved config binder, with no implicit environment lookup or global client;
- Redis lifecycle APIs for construct, close, ping, and health with context timeout handling;
- KV operations: Get, Set, Del, Exists, Expire, TTL, MGet, MSet, Incr, and Decr;
- health output with component, status, latency, error class, and checked_at fields;
- metrics named `redisx_operations_total`, `redisx_operation_duration_seconds`, `redisx_errors_total`, `redisx_pool_connections`, and `redisx_health_status`;
- Redis error classes including nil, timeout, canceled, network, auth, readonly, loading, try-again, moved, ask, closed, invalid config, and provider errors;
- testkit coverage with a fake Redis provider, contract assertions, and environment-gated real Redis integration tests.

## Harness and Evidence obligations

A redisx release candidate must pass the repository Harness before release evidence is accepted:

```bash
GOWORK=off go test ./...
GOWORK=off make boundary
GOWORK=off make contracts
GOWORK=off make docs-check
GOWORK=off make dependency-check
GOWORK=off make standard-impact-check
GOWORK=off make score
```

Evidence must remain replayable from tracked files. Generator archive behavior must not depend on untracked workspace state, and release-ready status must not be asserted without manifest, checksum, downstream, and retrospective evidence.

## Current implementation state

As of 2026-06-05, the tracked code is still a generic template scaffold for redisx. It has the repository skeleton and governance Harness surface, but provider-backed Redis APIs, Redis-specific schemas, fake Redis testkit helpers, release manifest output, downstream adoption proof, and retrospective patch assets remain incomplete. Until those gaps are closed, redisx must not be treated as release-ready or downstream adopted.
