# CNA-Go implementation plan

**Status:** foundation scaffold in place

**Date:** 2026-08-16

**Sources:** `../cnabinding/analysis_binding.md`,
`../cnabinding/analysis_binding_sharp_runtime.md`, and
`../cna/analysis_binding_languages.md`

## Goal

Provide an idiomatic Go frontend for CNA's canonical C++ implementation. The
binding should feel familiar to XNA/FNA/MonoGame developers without simulating
C# inheritance or copying the engine into Go.

## Phase 0 — repository scaffold (this commit)

- [x] README, plan, architecture, license, notices, editor settings, ignores.
- [x] Dependency-free Go module and local `Vector2`, `Color`, and `GameTime`.
- [x] `Game` interface, no-op `GameAdapter`, and explicit unavailable-runtime
      error while the native ABI is missing.
- [x] Reserved private `internal/interop` boundary.
- [x] Unit tests for the first local value types.

## Phase 1 — canonical native ABI

- [ ] Wait for the C ABI headers and implementation in `openeggbert/cna`;
      do not make this repository the ABI authority.
- [ ] Add the minimal cgo declarations for runtime initialization, ABI-version
      validation, structured errors, opaque handles, and shutdown.
- [ ] Convert every `CNA_Result` to a Go `error` carrying native error detail.
- [ ] Add pure-C and Go smoke tests for wrong ABI versions, stale handles,
      double close, UTF-8, and missing native libraries.

## Phase 2 — first playable loop

- [ ] Bridge `Game` callbacks with stable rooted context and documented thread
      rules.
- [ ] Add `GraphicsDevice`, `Texture2D`, `SpriteBatch`, `ContentManager`, and
      keyboard snapshots.
- [ ] Make native owners idempotently closable; keep borrowed handles distinct.
- [ ] Run a HelloGame that clears Cornflower Blue, loads and draws a texture,
      exits on Escape, and shuts down cleanly.

## Phase 3 — performance and packaging

- [ ] Buffer SpriteBatch commands and transfer arrays in bulk.
- [ ] Package supported CNA binaries without committing build outputs.
- [ ] Test at least Linux and Windows and report renderer/runtime limitations.
- [ ] Publish a pre-1.0 Go module only after the end-to-end sample works.

## Phase 4 — broader CNA/XNA concepts

- [ ] Complete local math, geometry, colors, and input value types.
- [ ] Add audio, effects, render targets, fonts, models, and 3D resources in
      increments backed by parity and lifecycle tests.
- [ ] Publish an honest compatibility matrix; do not promise universal XNA
      source or binary compatibility.

## Invariants

1. CNA C++ is the only engine implementation and its C ABI is the only FFI.
2. C++ exceptions and Sharp Runtime types never cross the boundary.
3. ABI primitives are fixed-width, strings are UTF-8, and ownership is explicit.
4. Math stays local; input uses snapshots; repetitive calls and buffers batch.
5. Public Go code never exposes raw C pointers, CNA handles, or result codes.
