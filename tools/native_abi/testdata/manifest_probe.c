// SPDX-License-Identifier: MS-PL
//
// The MANIFEST-side ABI probe. It includes CNA-Go's private declarations and
// deliberately NO canonical CNA header, which is exactly the compilation
// environment cgo gives bridge.c: the shipped binding's struct layouts,
// callback shapes and event identities come from abi_manifest.h alone.
//
// tools/native_abi runs this alongside the canonical probe and compares the two
// measurement sets key by key. Before that comparison existed, a manifest
// struct could have carried a wrong field width or a missing reserved byte and
// every existing check would still have passed: the canonical probe measures
// canonical types, the prototype pairing suppresses the manifest's private
// definitions whenever a CNA header is present, and the cgo build never sees a
// CNA header at all.
#include <stddef.h>
#include <stdio.h>

#include "../../../internal/interop/abi_manifest.h"
#include "../../../internal/interop/bridge.h"

// The manifest MUST be the thing being measured here. If a CNA header ever
// leaked into this translation unit the guards in abi_manifest.h would suppress
// its private definitions and this probe would silently measure canonical types
// instead -- reporting perfect agreement while proving nothing.
#if defined(CNA_C_ABI_H) || defined(CNA_C_CORE_H) || defined(CNA_C_RUNTIME_H) || \
    defined(CNA_C_GRAPHICS_H) || defined(CNA_C_GRAPHICS_DEVICE_H) || \
    defined(CNA_C_TEXTURE_H) || defined(CNA_C_INPUT_H)
#error "the manifest probe must not see a canonical CNA header"
#endif

int main(void) {
    printf("policy_major=%u\n", (unsigned)CNA_GO_ABI_MAJOR);
    printf("policy_minimum_minor=%u\n", (unsigned)CNA_GO_ABI_MINIMUM_MINOR);
    printf("policy_qualified_version=%u\n", (unsigned)CNA_GO_ABI_QUALIFIED_VERSION);
#define CNA_GO_MEASURE(key, expression) printf(#key "=%zu\n", (size_t)(expression));
#include "measurements.inc"
#undef CNA_GO_MEASURE
    return 0;
}
