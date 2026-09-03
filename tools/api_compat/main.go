package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	root := flag.String("root", ".", "cna-go repository root")
	contractFile := flag.String("contract", "tools/api_compat/reference/xna40-windows-runtime-contract.json", "XNA reference contract path relative to root")
	mappingFile := flag.String("mapping", "tools/api_compat/mapping-rules.json", "mapping rules path relative to root")
	reportFile := flag.String("report", "docs/generated/api-compat-report.json", "JSON report path relative to root; empty disables writing")
	missingFile := flag.String("missing", "docs/generated/missing-type-inventory.md", "Markdown inventory path relative to root; empty disables writing")
	remainingFile := flag.String("remaining", "docs/generated/remaining-work.md", "Markdown remaining-work table path relative to root; empty disables writing")
	mode := flag.String("mode", "strict", "strict, leak-only, or report")
	flag.Parse()

	if err := run(*root, *contractFile, *mappingFile, *reportFile, *missingFile, *remainingFile, *mode); err != nil {
		fmt.Fprintln(os.Stderr, "api-compat:", err)
		os.Exit(1)
	}
}

func run(root, contractFile, mappingFile, reportFile, missingFile, remainingFile, mode string) error {
	if mode != "strict" && mode != "leak-only" && mode != "report" {
		return fmt.Errorf("unsupported mode %q", mode)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	contractBytes, err := os.ReadFile(resolvePath(absoluteRoot, contractFile))
	if err != nil {
		return err
	}
	var reference contract
	if err := json.Unmarshal(contractBytes, &reference); err != nil {
		return fmt.Errorf("parse contract: %w", err)
	}
	if reference.SchemaVersion != 2 || reference.Profile != "XNA 4.0 Windows runtime" {
		return fmt.Errorf("unexpected reference contract schema/profile: %d %q", reference.SchemaVersion, reference.Profile)
	}
	mappingBytes, err := os.ReadFile(resolvePath(absoluteRoot, mappingFile))
	if err != nil {
		return err
	}
	allowlistEntries, err := mappingAllowlistEntries(mappingBytes)
	if err != nil {
		return err
	}
	expected, err := buildExpected(reference)
	if err != nil {
		return err
	}
	if expected.ReferenceTypes != 257 || expected.ReferenceMembers != 2964 {
		return fmt.Errorf("reference admission failed: got %d types/%d members", expected.ReferenceTypes, expected.ReferenceMembers)
	}
	// The expected Go surface is admitted by its parts rather than by one
	// total, so a change in either provenance class is attributed instead of
	// being absorbed. 3243 is the pinned projection of the 2,964 XNA-declared
	// reference members and never moves; BCL-inherited projections are added
	// on top and are separately pinned by the adapter registry.
	declaredProjections := expected.ExpectedGoMembers - expected.BCLInheritedProjections - expected.XNAInheritedProjections
	if expected.ExpectedGoTypes != 257 || declaredProjections != 3243 {
		return fmt.Errorf("mapping count admission failed: got %d types/%d XNA-declared member projections", expected.ExpectedGoTypes, declaredProjections)
	}
	if expected.BCLInheritedProjections != expectedBCLInheritedProjections(expected) {
		return fmt.Errorf("BCL inherited projection admission failed: got %d, registry implies %d", expected.BCLInheritedProjections, expectedBCLInheritedProjections(expected))
	}
	// The third provenance class is admitted the same way: recomputed from the
	// per-type counts the mapper produced, so the total is checked against an
	// independent derivation rather than against itself.
	xnaInherited := 0
	for _, et := range expected.Types {
		xnaInherited += et.XNAInheritedProjections
	}
	if expected.XNAInheritedProjections != xnaInherited {
		return fmt.Errorf("XNA inherited projection admission failed: got %d, per-type counts imply %d", expected.XNAInheritedProjections, xnaInherited)
	}
	actual, err := extractActual(absoluteRoot)
	if err != nil {
		return err
	}
	if len(actual.TypeErrors) != 0 {
		return fmt.Errorf("Go type-check admission produced %d errors; first: %s", len(actual.TypeErrors), actual.TypeErrors[0])
	}
	result := verify(expected, actual, allowlistEntries, mode, sha256Hex(contractBytes), sha256Hex(mappingBytes))
	if reportFile != "" {
		if err := writeJSON(resolvePath(absoluteRoot, reportFile), result); err != nil {
			return err
		}
	}
	if missingFile != "" {
		if err := writeMissingInventory(resolvePath(absoluteRoot, missingFile), result); err != nil {
			return err
		}
	}
	if remainingFile != "" {
		filename := resolvePath(absoluteRoot, remainingFile)
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filename, []byte(renderRemainingWork(result.Frontier, result.Summary)), 0o644); err != nil {
			return err
		}
	}
	printSummary(result)

	switch mode {
	case "strict":
		if result.Summary["TOTAL_DIAGNOSTICS"] != 0 {
			return errors.New("strict structural verification is red (expected while the binding is incomplete)")
		}
	case "leak-only":
		if result.Summary["INTERNAL_TYPE_LEAK"]+result.Summary["RAW_HANDLE_LEAK"]+result.Summary["PUBLIC_NATIVE_FFI_LEAK"] != 0 {
			return errors.New("public native-state leak verification failed")
		}
	}
	return nil
}

func resolvePath(root, name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(root, name)
}

func mappingAllowlistEntries(data []byte) (int, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, fmt.Errorf("parse mapping rules: %w", err)
	}
	raw, ok := value["allowlist"]
	if !ok {
		return 0, nil
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return 0, fmt.Errorf("parse mapping allowlist: %w", err)
	}
	return len(entries), nil
}

func writeJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0o644)
}

func writeMissingInventory(filename string, result report) error {
	var output strings.Builder
	output.WriteString("# Generated missing-type inventory\n\n")
	output.WriteString("Generated by `go run ./tools/api_compat --mode report`. Do not edit by hand.\n\n")
	fmt.Fprintf(&output, "- Complete types: %d\n- Partial types: %d\n- Missing types: %d\n\n", len(result.CompleteTypes), len(result.PartialTypes), len(result.MissingTypes))
	output.WriteString("## Complete types\n\n")
	if len(result.CompleteTypes) == 0 {
		output.WriteString("None.\n\n")
	} else {
		for _, name := range result.CompleteTypes {
			fmt.Fprintf(&output, "- `%s`\n", name)
		}
		output.WriteByte('\n')
	}
	output.WriteString("## Partial types\n\n")
	if len(result.PartialTypes) == 0 {
		output.WriteString("None.\n\n")
	} else {
		for _, item := range result.PartialTypes {
			fmt.Fprintf(&output, "### `%s`\n\n", item.XNA)
			for _, missing := range item.MissingMembers {
				fmt.Fprintf(&output, "- `%s`\n", missing)
			}
			output.WriteByte('\n')
		}
	}
	output.WriteString("## Missing types\n\n")
	for _, name := range result.MissingTypes {
		fmt.Fprintf(&output, "- `%s`\n", name)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filename, []byte(output.String()), 0o644)
}

func printSummary(result report) {
	order := []string{
		"REFERENCE_TYPES", "REFERENCE_MEMBERS", "REFERENCE_XNA_MEMBERS",
		"BCL_INHERITED_PUBLIC_MEMBERS", "BCL_INHERITED_MEMBER_PROJECTIONS",
		"XNA_INHERITED_PUBLIC_MEMBERS", "XNA_INHERITED_PUBLIC_MEMBERS_OVERRIDDEN",
		"XNA_INHERITED_MEMBER_PROJECTIONS",
		"EXPECTED_GO_TYPES", "EXPECTED_GO_MEMBERS",
		"TARGET_TYPES", "TARGET_MEMBERS", "TOTAL_DIAGNOSTICS", "MISSING_TYPE", "MISSING_MEMBER",
		"COMPLETE_TYPES", "PARTIAL_TYPES", "MISSING_TYPES",
		"FRONTIER_FAMILIES", "GLOBAL_ACTIONABLE_LOCAL", "GLOBAL_UNREVIEWED",
		"GLOBAL_BLOCKED_UPSTREAM_CNA", "GLOBAL_BLOCKED_PLATFORM", "GLOBAL_BLOCKED_HARDWARE",
		"GLOBAL_BLOCKED_FIXTURE", "GLOBAL_BLOCKED_REFERENCE_ASSET",
		"GLOBAL_LANGUAGE_MAPPING_LIMITATION", "GLOBAL_BCL_PROJECTION_BLOCKED_EXTERNAL",
		"GLOBAL_DELIBERATE_NON_BINDING",
		"INTERFACE_WITNESS_PROJECTIONS", "PACKFROMVECTOR4_WITNESS_PROJECTIONS", "TOVECTOR4_WITNESS_PROJECTIONS",
		"BCL_BASE_ADAPTERS", "BCL_BASE_ADAPTER_CONSUMERS",
		"BCL_SIGNATURE_ADAPTERS", "BCL_SIGNATURE_ADAPTER_CARRIERS", "BCL_DEFERRED_BASE_BLOCKERS",
		"GAME_BASE_CALL_ADAPTERS", "GAME_BASE_CALL_DEFERRED_STEPS",
		"GAME_NATIVE_SIGNALS", "GAME_NATIVE_SIGNAL_RAISE_SITES",
		"GAME_NATIVE_SIGNALS_RUNTIME_DEFERRED",
		"GAME_NATIVE_SIGNALS_LIFECYCLE_ONLY", "GAME_MANAGED_EVENT_RAISE_SITES",
		"GAME_FRAME_HOOKS", "GAME_FRAME_HOOKS_NEVER_INSTALLED",
		"GAME_FRAME_HOOKS_INSTALLED_ON_OVERRIDE", "GAME_FRAME_HOOK_OVERRIDE_CAPABILITIES",
		"GAME_CALLBACKS_MEMBERS", "GAME_FRAME_HOOK_DEFERRED_STEPS",
		"DECLARED_INTERFACE_CONFORMANCE",
		"XNA_BASE_RELATIONSHIPS", "XNA_BASE_DERIVED_TYPES",
		"XNA_DEFERRED_BASE_BLOCKERS", "XNA_INHERITED_PUBLIC_MEMBERS_UNPROJECTED",
		"XNA_BASE_TYPED_SIGNATURE_POSITIONS", "XNA_BASE_SUBSTITUTABILITY_NONE",
		"XNA_BASE_SUBSTITUTABILITY_LATENT", "XNA_BASE_SUBSTITUTABILITY_LIVE",
		"XNA_BASE_SUBSTITUTABILITY_REGISTERED",
		"XNA_COMPOSED_BASE_RELATIONSHIPS", "XNA_COMPOSED_DERIVED_TYPES",
		"XNA_COMPOSED_DERIVED_TYPES_PROJECTED", "XNA_INHERITED_ATTRIBUTED_MEMBERS",
		"XNA_COMPOSED_IDENTITY_SITES", "XNA_COMPOSED_IDENTITY_USES",
		"XNA_COMPOSED_IDENTITY_FORWARDS", "XNA_COMPOSED_IDENTITY_BINDINGS",
	}
	order = append(order, diagnosticCategories[2:]...)
	seen := make(map[string]bool)
	for _, key := range order {
		if seen[key] {
			continue
		}
		seen[key] = true
		fmt.Printf("%s=%d\n", key, result.Summary[key])
	}
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func sortedKeys(values map[string]int) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

// expectedBCLInheritedProjections recomputes the BCL-inherited projection
// count straight from the adapter registry and the pinned contract, so the
// number the mapper produced is admitted against an independent derivation
// rather than against itself. A CLR property with two accessors projects two
// Go members; every other inherited member projects one.
func expectedBCLInheritedProjections(expected *expectedSurface) int {
	total := 0
	for _, et := range expected.Types {
		if et.BCLInheritedCLRMembers == 0 {
			continue
		}
		adapter := bclBaseAdapters[baseIdentityWithoutArguments(et.BaseType)]
		for _, entry := range adapter.Members {
			switch {
			case entry.Member.Kind == "property" && entry.Member.Get && entry.Member.Set:
				total += 2
			default:
				total++
			}
		}
	}
	return total
}
