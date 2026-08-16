# CNA-Go implementation plan

**Status:** corrected namespace scaffold in place

**Date:** 2026-08-16

## Phase 0 — namespace scaffold

- [x] Establish `CNA/Framework` and `Microsoft/Xna/Framework` import roots.
- [x] Reserve matching `Graphics`, `Input`, and `Content` subpackages.
- [x] Reserve `CNA/Interop` for the stable C ABI mapping.
- [x] Add initial `Game`, `GameTime`, `Vector2`, and `Color` shapes.

## Phase 1 — canonical ABI

- [ ] Generate or audit interop from headers owned by `openeggbert/cna`.
- [ ] Add ABI-version checks, UTF-8, structured errors, opaque handles,
      callback context, ownership, threading, and shutdown rules.

## Phase 2 — first playable XNA-style loop

- [ ] Add graphics device, texture, sprite batch, content, and keyboard types
      under both public trees.
- [ ] Run a CNA-backed game that clears, loads/draws a texture, reads Escape,
      and shuts down cleanly.

## Invariants

1. Public hierarchy follows CNA and `Microsoft.Xna.Framework` concepts.
2. CNA C++ remains the only engine implementation.
3. Only the stable CNA C ABI crosses the language boundary.
4. Sharp Runtime and C++ ABI details remain native implementation details.
