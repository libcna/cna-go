# CNA-Go implementation plan

**Status:** XNA namespace scaffold in place

**Date:** 2026-08-16

## Phase 0 — repository scaffold

- [x] Establish `Microsoft/Xna/Framework` plus `Graphics`, `Input`, and
      `Content` compatibility packages.
- [x] Keep ABI code under private `internal/interop`.
- [x] Add initial `Game`, `GameTime`, `Vector2`, and `Color` shapes.
- [x] Remove the invalid invented `CNA/Framework` package tree.

## Phase 1 — canonical ABI

- [ ] Generate or audit interop only from headers owned by `openeggbert/cna`.
- [ ] Add version checks, errors, UTF-8, opaque handles, callbacks, ownership,
      threading, and shutdown.

## Phase 2 — playable compatibility slice

- [ ] Add graphics device, texture, sprite batch, content, and keyboard types.
- [ ] Run a CNA-backed XNA-style game loop.

## Invariants

1. XNA types follow the `Microsoft.Xna.Framework` hierarchy.
2. No public namespace/package is invented without a native counterpart.
3. CNA C++ remains canonical and only its stable C ABI crosses the boundary.
