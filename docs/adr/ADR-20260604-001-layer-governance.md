# ADR-20260604-001: redisx layer governance boundary

## Status

Accepted.

## Context

`redisx` is the public Standard Source for the L2 Redis factory. It must define reusable standard, template, generator, Harness, and Evidence behavior without importing L3 private business systems or production runtime policy.

The repository also documents downstream consumers and private business systems, so the governance boundary must be explicit enough for `docs-check`, dependency checks, and release Evidence gates to fail closed when L3 concerns leak into public artifacts.

## Decision

Adopt the layer direction `L3 -> L2 -> L1 -> L0 -> Standard` and keep `redisx` as a public L2/Standard authority surface. `x.go` and business repositories remain L3 私有 and may consume released public libraries, but public repositories must not consume L3 code, schemas, strategies, production secret paths, or customer-data semantics.

The machine-checkable source of this rule is split across:

- `docs/standard/layer-governance-rules.md` for human policy;
- `.agent/policies/layer-governance.yaml` for registry facts;
- `contracts/layer-governance.schema.json` for schema validation;
- `scripts/check_docs.sh` and `GOWORK=off make docs-check` for documentation completeness;
- boundary, contracts, dependency, standard-impact, release, and Evidence gates for executable verification.

## Consequences

- P0 violations have no temporary exception path.
- L3 私有 systems configure `GOPRIVATE`, inject secrets, and own production wiring outside this repository.
- redisx can document private downstream expectations only as boundary guidance; it must not commit private implementation details or secret values.
- Any future change to layer direction or release responsibility must update the standard docs, policy registry, schema, Harness gates, docs-check requirements, and downstream sync records together.
