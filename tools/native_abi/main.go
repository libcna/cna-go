// Command native_abi compiles CNA-Go's private ABI manifest against the
// canonical CNA headers, measures the manifest's own declarations in the
// compilation environment cgo actually gives them, and independently admits a
// selected shared library.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/openeggbert/cna-go/internal/interop"
)

type report struct {
	SchemaVersion              int            `json:"schema_version"`
	Status                     string         `json:"status"`
	HeaderRoot                 string         `json:"header_root"`
	CanonicalHeaderSHA256      string         `json:"canonical_header_sha256"`
	CanonicalHeaderFiles       int            `json:"canonical_header_files"`
	NativeLibrary              string         `json:"native_library"`
	NativeLibrarySHA256        string         `json:"native_library_sha256"`
	AdmittedABI                string         `json:"ADMITTED_ABI"`
	CanonicalHeaderABI         string         `json:"CANONICAL_HEADER_ABI"`
	LoadedABI                  string         `json:"LOADED_ABI"`
	BoundFunctions             int            `json:"BOUND_FUNCTIONS"`
	CanonicalDeclarations      int            `json:"CANONICAL_DECLARATIONS"`
	DeliberatelyUnboundRoutes  int            `json:"DELIBERATELY_UNBOUND_ROUTES"`
	LibraryExports             int            `json:"LIBRARY_EXPORTS"`
	PrototypeTypePositions     int            `json:"PROTOTYPE_TYPE_POSITIONS"`
	RouteTypePairings          int            `json:"ROUTE_TYPE_PAIRINGS"`
	CGoMeasurements            int            `json:"C_GO_MEASUREMENTS"`
	Layouts                    int            `json:"LAYOUTS"`
	ManifestLayoutAgreements   int            `json:"MANIFEST_LAYOUT_AGREEMENTS"`
	Callbacks                  int            `json:"CALLBACKS"`
	Constants                  int            `json:"CONSTANTS"`
	ManifestSideAssertions     int            `json:"MANIFEST_SIDE_ASSERTIONS"`
	SymbolIdentityVerified     bool           `json:"SYMBOL_IDENTITY_VERIFIED"`
	MissingHeaderSymbols       []string       `json:"MISSING_HEADER_SYMBOLS"`
	MissingLibrarySymbols      []string       `json:"MISSING_LIBRARY_SYMBOLS"`
	ABIMismatches              []string       `json:"ABI_MISMATCHES"`
	Findings                   []string       `json:"FINDINGS"`
	Measurements               map[string]int `json:"measurements"`
	ManifestMeasurements       map[string]int `json:"manifest_measurements"`
	Functions                  []string       `json:"functions"`
	FunctionParameterPositions map[string]int `json:"function_parameter_positions"`
}

// route is one bound CNA C ABI function, read from CNA-Go's own manifest rather
// than from a table maintained beside it. The manifest is what the cgo build
// compiles, so a route added there is measured here without a second edit, and
// a route measured here that the bridge never resolves is impossible by
// construction.
type route struct {
	name       string
	parameters []string
}

var (
	requiredSymbolPattern = regexp.MustCompile(`(?s)#define CNA_GO_REQUIRED_SYMBOLS\(X\)(.*?)\n\n`)
	symbolEntryPattern    = regexp.MustCompile(`X\(([A-Za-z0-9_]+)\)`)
	staticAssertPattern   = regexp.MustCompile(`_Static_assert\s*\(`)
	callbackPinPattern    = regexp.MustCompile(`static\s+CNA_[A-Za-z0-9_]*Callback\s+checked_CNA_[A-Za-z0-9_]+\s*=`)
	declarationPattern    = regexp.MustCompile(`\bcna_[a-z0-9_]+\s*\(`)
)

func main() {
	headers := flag.String("headers", "../../cnanext/modules/c-api/include", "canonical CNA C include root")
	library := flag.String("library", "", "absolute path to an admitted CNA shared library")
	output := flag.String("output", "docs/generated/native-abi-report.json", "JSON report path")
	flag.Parse()
	result, err := verify(*headers, *library)
	if writeErr := writeReport(*output, result); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("BOUND_FUNCTIONS=%d MANIFEST_LAYOUT_AGREEMENTS=%d ABI_MISMATCHES=%d MISSING_LIBRARY_SYMBOLS=%d LOADED_ABI=%s\n",
		result.BoundFunctions, result.ManifestLayoutAgreements, len(result.ABIMismatches),
		len(result.MissingLibrarySymbols), result.LoadedABI)
}

// parseManifestRoutes reads the required-symbol list and each route's own
// function-pointer typedef out of abi_manifest.h.
func parseManifestRoutes(manifest string) ([]route, error) {
	block := requiredSymbolPattern.FindStringSubmatch(manifest)
	if block == nil {
		return nil, errors.New("abi_manifest.h has no CNA_GO_REQUIRED_SYMBOLS list")
	}
	names := symbolEntryPattern.FindAllStringSubmatch(block[1], -1)
	if len(names) == 0 {
		return nil, errors.New("CNA_GO_REQUIRED_SYMBOLS lists no symbols")
	}
	routes := make([]route, 0, len(names))
	for _, match := range names {
		name := match[1]
		typedefPattern := regexp.MustCompile(`typedef\s+[A-Za-z0-9_ ]+\s*\(\s*\*\s*` + regexp.QuoteMeta(name) + `_fn\s*\)\s*\(([^)]*)\)\s*;`)
		typedef := typedefPattern.FindStringSubmatch(manifest)
		if typedef == nil {
			return nil, fmt.Errorf("%s is required but has no %s_fn typedef", name, name)
		}
		routes = append(routes, route{name: name, parameters: parseParameters(typedef[1])})
	}
	return routes, nil
}

func parseParameters(list string) []string {
	trimmed := strings.TrimSpace(list)
	if trimmed == "" || trimmed == "void" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, strings.Join(strings.Fields(part), " "))
	}
	return result
}

// hashHeaderTree is the canonical header tree's content identity: a SHA-256 over
// every `.h` file under the root, ordered by relative slash-separated path, with
// each path and each file length fed into the digest so a rename or a truncation
// cannot collide with an edit.
func hashHeaderTree(root string) (string, int, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".h") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(paths)
	digest := sha256.New()
	for _, rel := range paths {
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil {
			return "", 0, readErr
		}
		fmt.Fprintf(digest, "%s\n%d\n", rel, len(content))
		digest.Write(content)
	}
	return hex.EncodeToString(digest.Sum(nil)), len(paths), nil
}

func verify(headerRoot, library string) (report, error) {
	result := report{
		SchemaVersion: 3, Status: "FAIL", HeaderRoot: headerRoot, NativeLibrary: library,
		AdmittedABI:  interop.ABIAdmissionPolicy(),
		Measurements: map[string]int{}, ManifestMeasurements: map[string]int{},
		FunctionParameterPositions: map[string]int{},
		MissingHeaderSymbols:       []string{}, MissingLibrarySymbols: []string{},
		ABIMismatches: []string{}, Findings: []string{},
	}
	// The header tree is pinned by CONTENT, for the reason the library already
	// is: a path is ephemeral and unverifiable, and this one is now actively
	// misleading. The default `-headers ../../cnanext/modules/c-api/include`
	// reads a LIVE checkout, and that checkout has moved past the artifact the
	// qualification library was built from -- Milestone 55 measured the two
	// trees differing. A digest makes "which headers were these" a fact rather
	// than a claim about a directory that may since have changed.
	headerDigest, headerFiles, err := hashHeaderTree(headerRoot)
	if err != nil {
		return result, err
	}
	result.CanonicalHeaderSHA256 = headerDigest
	result.CanonicalHeaderFiles = headerFiles

	manifestSource, err := os.ReadFile(filepath.Join("internal", "interop", "abi_manifest.h"))
	if err != nil {
		return result, err
	}
	routes, err := parseManifestRoutes(string(manifestSource))
	if err != nil {
		return result, err
	}
	for _, entry := range routes {
		result.Functions = append(result.Functions, entry.name)
		result.FunctionParameterPositions[entry.name] = len(entry.parameters)
		result.PrototypeTypePositions += 1 + len(entry.parameters)
	}
	sort.Strings(result.Functions)
	result.BoundFunctions = len(routes)
	// Every manifest typedef is assigned its OWN canonical function in probe.c
	// under -Werror=incompatible-pointer-types, so a route's private prototype
	// is checked against the declaration of the same name rather than against a
	// compatible neighbour. Routes that share a shape are separated at load
	// time instead, by cna_go_verify_symbol_identity.
	result.RouteTypePairings = len(routes)

	probeSource, err := os.ReadFile(filepath.Join("tools", "native_abi", "testdata", "probe.c"))
	if err != nil {
		return result, err
	}
	bridgeSource, err := os.ReadFile(filepath.Join("internal", "interop", "bridge.c"))
	if err != nil {
		return result, err
	}
	// Counted from the source rather than restated: a pin added or removed
	// moves the number without a second edit.
	result.Constants = len(staticAssertPattern.FindAllString(string(probeSource), -1))
	result.ManifestSideAssertions = len(staticAssertPattern.FindAllString(string(bridgeSource), -1))
	result.Callbacks = len(callbackPinPattern.FindAllString(string(probeSource), -1))

	root, err := filepath.Abs(headerRoot)
	if err != nil {
		return result, err
	}
	declared, err := canonicalDeclarations(root)
	if err != nil {
		return result, err
	}
	result.CanonicalDeclarations = len(declared)
	boundNames := make(map[string]struct{}, len(routes))
	for _, entry := range routes {
		boundNames[entry.name] = struct{}{}
	}
	result.DeliberatelyUnboundRoutes = verifyUnboundRoutes(&result, declared, boundNames)
	temporary, err := os.MkdirTemp("", "cna-go-native-abi-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(temporary)
	commonCompilerArgs := []string{"-std=c11", "-Wall", "-Wextra", "-Werror", "-Werror=incompatible-pointer-types"}
	canonicalArgs := append(append([]string{}, commonCompilerArgs...), "-I"+root)

	objectArgs := append(append([]string{}, canonicalArgs...), "-c", "tools/native_abi/testdata/probe.c", "-o", filepath.Join(temporary, "probe.o"))
	if output, compileErr := exec.Command("gcc", objectArgs...).CombinedOutput(); compileErr != nil {
		result.ABIMismatches = append(result.ABIMismatches, "canonical-header compile failed: "+strings.TrimSpace(string(output)))
		return finish(result), compileErr
	}
	canonicalBinary := filepath.Join(temporary, "canonical-probe")
	layoutArgs := append(append([]string{}, canonicalArgs...), "-DCNA_GO_LAYOUT_ONLY", "tools/native_abi/testdata/probe.c", "-o", canonicalBinary)
	if output, compileErr := exec.Command("gcc", layoutArgs...).CombinedOutput(); compileErr != nil {
		result.ABIMismatches = append(result.ABIMismatches, "canonical-header layout probe failed: "+strings.TrimSpace(string(output)))
		return finish(result), compileErr
	}
	result.Measurements, err = runProbe(canonicalBinary)
	if err != nil {
		result.ABIMismatches = append(result.ABIMismatches, "compiled canonical ABI probe did not run")
		return finish(result), err
	}

	// The manifest probe compiles CNA-Go's private declarations with NO
	// canonical header, which is the environment cgo gives bridge.c. Without it
	// the manifest's own layouts were never measured against anything.
	manifestBinary := filepath.Join(temporary, "manifest-probe")
	manifestArgs := append(append([]string{}, commonCompilerArgs...), "tools/native_abi/testdata/manifest_probe.c", "-o", manifestBinary)
	if output, compileErr := exec.Command("gcc", manifestArgs...).CombinedOutput(); compileErr != nil {
		result.ABIMismatches = append(result.ABIMismatches, "manifest-only probe failed to compile: "+strings.TrimSpace(string(output)))
		return finish(result), compileErr
	}
	result.ManifestMeasurements, err = runProbe(manifestBinary)
	if err != nil {
		result.ABIMismatches = append(result.ABIMismatches, "compiled manifest ABI probe did not run")
		return finish(result), err
	}

	result.CanonicalHeaderABI = decodeMeasuredABI(result.Measurements)
	// LAYOUTS counts the aggregate measurements, which is every measurement the
	// shared list carries: the ABI version parts are reported separately.
	for key := range result.Measurements {
		if !strings.HasPrefix(key, "abi_") {
			result.Layouts++
		}
	}
	agreements, divergences := compareMeasurements(result.Measurements, result.ManifestMeasurements)
	result.ManifestLayoutAgreements = agreements
	result.ABIMismatches = append(result.ABIMismatches, divergences...)
	if agreements == 0 {
		result.ABIMismatches = append(result.ABIMismatches, "no measurement is shared between the canonical and manifest probes")
	}
	result.CGoMeasurements = len(result.Measurements) + result.PrototypeTypePositions

	if library == "" {
		return finish(result), errors.New("-library is required")
	}
	absLibrary, err := filepath.Abs(library)
	if err != nil {
		return finish(result), err
	}
	// Reports retain the content identity, not an ephemeral qualification path.
	// Consumers select their own absolute runtime path with CNA_NATIVE_LIBRARY.
	result.NativeLibrary = filepath.Base(absLibrary)
	data, err := os.ReadFile(absLibrary)
	if err != nil {
		return finish(result), err
	}
	hash := sha256.Sum256(data)
	result.NativeLibrarySHA256 = hex.EncodeToString(hash[:])
	result.LibraryExports = countLibraryExports(absLibrary)
	verification, err := interop.VerifyNativeLibrary(absLibrary)
	if err != nil {
		result.ABIMismatches = append(result.ABIMismatches, err.Error())
		return finish(result), err
	}
	result.LoadedABI = interop.FormatABIVersion(verification.ABIVersion)
	result.MissingLibrarySymbols = append(result.MissingLibrarySymbols, verification.MissingSymbols...)
	result.SymbolIdentityVerified = verification.SymbolIdentityVerified
	if !verification.SymbolIdentityVerified {
		result.ABIMismatches = append(result.ABIMismatches, "resolved symbol identity: "+verification.SymbolIdentityDetail)
	}
	if !interop.ABIAdmits(verification.ABIVersion) {
		result.ABIMismatches = append(result.ABIMismatches,
			fmt.Sprintf("loaded ABI %s is outside the admitted range (%s)", result.LoadedABI, result.AdmittedABI))
	}
	if result.CanonicalHeaderABI != "" && result.CanonicalHeaderABI != result.LoadedABI {
		result.ABIMismatches = append(result.ABIMismatches,
			fmt.Sprintf("canonical headers declare ABI %s but the library reports %s", result.CanonicalHeaderABI, result.LoadedABI))
	}
	if result.LibraryExports > 0 && result.CanonicalDeclarations > 0 && result.LibraryExports != result.CanonicalDeclarations {
		result.ABIMismatches = append(result.ABIMismatches,
			fmt.Sprintf("canonical headers declare %d cna_* routes but the library exports %d", result.CanonicalDeclarations, result.LibraryExports))
	}
	if len(verification.BoundSymbols) != result.BoundFunctions {
		result.ABIMismatches = append(result.ABIMismatches, fmt.Sprintf("bridge bound %d symbols; manifest has %d", len(verification.BoundSymbols), result.BoundFunctions))
	}
	if len(result.ABIMismatches) == 0 && len(result.MissingHeaderSymbols) == 0 && len(result.MissingLibrarySymbols) == 0 {
		result.Status = "PASS"
		return finish(result), nil
	}
	return finish(result), errors.New("native ABI verification failed")
}

// finish collects every diagnostic into one findings list so a reader does not
// have to union three fields to learn whether anything is wrong.
func finish(result report) report {
	result.Findings = append(result.Findings, result.ABIMismatches...)
	for _, symbol := range result.MissingHeaderSymbols {
		result.Findings = append(result.Findings, "missing header symbol: "+symbol)
	}
	for _, symbol := range result.MissingLibrarySymbols {
		result.Findings = append(result.Findings, "missing library symbol: "+symbol)
	}
	if result.Findings == nil {
		result.Findings = []string{}
	}
	return result
}

func runProbe(binary string) (map[string]int, error) {
	measurements := map[string]int{}
	output, err := exec.Command(binary).Output()
	if err != nil {
		return measurements, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		if value, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
			measurements[parts[0]] = value
		}
	}
	return measurements, scanner.Err()
}

// compareMeasurements is the check that the manifest's OWN declarations, not a
// canonical type that happens to share a name, are what the shipped binding
// uses. Every key both probes emit must agree exactly.
func compareMeasurements(canonical, manifest map[string]int) (int, []string) {
	agreements := 0
	divergences := []string{}
	keys := make([]string, 0, len(canonical))
	for key := range canonical {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		manifestValue, ok := manifest[key]
		if !ok {
			continue
		}
		if manifestValue == canonical[key] {
			agreements++
			continue
		}
		divergences = append(divergences, fmt.Sprintf("%s: canonical %d, manifest %d", key, canonical[key], manifestValue))
	}
	return agreements, divergences
}

func decodeMeasuredABI(measurements map[string]int) string {
	major, hasMajor := measurements["abi_major"]
	minor, hasMinor := measurements["abi_minor"]
	patch, hasPatch := measurements["abi_patch"]
	if !hasMajor || !hasMinor || !hasPatch {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

// countCanonicalDeclarations counts the distinct cna_* routes the canonical
// headers declare, so the report can state whether the selected library and the
// headers describe the same surface.
// ---------------------------------------------------------------------------
// Foundation 57 — the deliberately unbound routes.
// ---------------------------------------------------------------------------

// unboundRoute records a canonical CNA route CNA-Go does NOT bind even though
// the XNA member it could back IS projected.
//
// # Why this registry exists
//
// The pinned artifact declares 4,054 routes and CNA-Go binds 78. Almost all of
// the difference is uninteresting: the XNA surface those routes would back is
// not projected yet, which the missing-type inventory already says.
//
// A route is interesting when the member IS projected and CNA-Go answers it
// managed-side anyway. That is a real decision with a real alternative, and
// until Foundation 57 nothing recorded any of them. Foundation 56 shipped one
// stated backwards -- GraphicsResource's Name was justified with "CNA has no
// such cache and no route that could reach one", and CNA has five.
//
// A recorded entry is checked, not merely written down: the route must exist in
// the canonical headers, and it must NOT also be bound. A registry that could
// name a route CNA never declared, or one the manifest already resolves, would
// be prose with a struct around it.
type unboundRoute struct {
	// Route is the canonical cna_* identifier.
	Route string
	// Member is the projected XNA member the route could back.
	Member string
	// Class is one of unboundRouteClasses.
	Class string
	// Detail is the measured reason, including what was measured and when.
	Detail string
}

// unboundRouteClasses are the only reasons a projected member may answer
// managed-side while CNA offers a route for it.
var unboundRouteClasses = map[string]string{
	// Binding it would change the projected member's observable behaviour away
	// from the reference's.
	"CONTRACT_DIVERGENCE": "binding the route would diverge from the reference contract",
	// The route does not accept every resource kind the projected member covers,
	// so binding it would make one XNA member behave per-kind.
	"KIND_PARTIAL": "the route accepts only some of the kinds the member covers",
	// The route's C type cannot represent the CLR type the member declares.
	"REPRESENTATION": "the C type cannot represent the CLR type",
	// The reference's own implementation reaches no runtime at all, so there is
	// nothing for a native route to be more faithful than.
	"MANAGED_REFERENCE": "the reference implementation is managed and reaches no runtime",
	// The route reports data a route CNA-Go already binds reports, and reading
	// both would be two answers to one question that could disagree.
	"REDUNDANT_READ": "the route re-reports data an already bound route carries",
	// A WIDER bound route's contract contains this one's, and the reference has
	// one body for both, so binding both would give one reference path two
	// native paths that could drift.
	"SUBSUMED": "a wider bound route expresses this one's whole contract",
}

// deliberatelyUnboundRoutes is the closed registry.
//
// # The graphics-resource family, measured against the pinned 0.21.0 artifact
//
// Every row below comes from one probe run against
// `~/deps/cna-c-abi-0.21.0/libcna_c_api.so`, on a live device inside a
// lifecycle callback, over a real Texture2D handle and a real SpriteBatch
// handle:
//
//	fresh texture   get_is_disposed -> false           set_name -> ok
//	                copy_name -> ""                    copy_name -> "probe-name"
//	                set_name("a\0b") -> CNA result 11, "not valid UTF-8"
//	SpriteBatch     get_is_disposed -> CNA result 2, "The handle does not refer
//	                                   to a supported graphics resource"
//	                set_name, copy_name -> the same refusal
//	after cna_graphics_resource_dispose:
//	                get_is_disposed -> true            copy_name -> "probe-name"
//	                the texture STILL ENCODES: cna_texture2d_copy_encoded
//	                returned 70 bytes, so CNA's "dispose" is a flag and a
//	                notification, NOT a release
//	                a repeated dispose -> success, as documented
//
// Two facts decide the whole family. CNA's graphics-resource routes accept a
// texture and REFUSE a SpriteBatch, and XNA's GraphicsResource members are
// uniform across every derived type. And CNA's set_name validates UTF-8 while
// a Go string may hold arbitrary bytes and `set_Name` in the reference accepts
// anything, including null.
var deliberatelyUnboundRoutes = []unboundRoute{
	{
		Route:  "cna_graphics_resource_get_name_byte_count",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Name()",
		Class:  "KIND_PARTIAL",
		Detail: "the route refuses a SpriteBatch handle with CNA result 2, and SpriteBatch is a GraphicsResource in the pinned contract, so binding it would make one XNA member answer for textures and fail for sprite batches",
	},
	{
		Route:  "cna_graphics_resource_copy_name",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Name()",
		Class:  "KIND_PARTIAL",
		Detail: "the second half of the same read; it refuses the same handles",
	},
	{
		Route:  "cna_graphics_resource_set_name",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Name()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "CNA validates the name as UTF-8 and refused an embedded NUL with CNA result 11. GraphicsResource::set_Name is `ldarg.1; stfld` -- or one DeviceResourceManager store -- and validates nothing at all, including null. Binding it would refuse names the reference accepts, and would make an infallible member fallible",
	},
	{
		Route:  "cna_graphics_resource_get_is_disposed",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::IsDisposed()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_IsDisposed is `ldarg.0; ldfld isDisposed; ret` over a managed field the managed ~GraphicsResource sets, so the projection's own flag IS the reference's. The route also refuses a SpriteBatch handle",
	},
	{
		Route:  "cna_graphics_resource_get_tag",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Tag()",
		Class:  "REPRESENTATION",
		Detail: "CNA_GraphicsResourceTag is a uint64 opaque token and GraphicsResource::Tag is System.Object, which projects to Go `any`. A uint64 cannot carry an arbitrary Go value without a side registry, and a registry keyed by a token CNA owns would be a second lifetime to get wrong",
	},
	{
		Route:  "cna_graphics_resource_set_tag",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Tag()",
		Class:  "REPRESENTATION",
		Detail: "the write half of the same token",
	},
	{
		Route:  "cna_graphics_resource_get_string_byte_count",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::ToString()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "GraphicsResource::ToString has an exactly specified managed body -- the resource's Name when non-empty, and otherwise System.Object::ToString, which is the RUNTIME type's full CLR name. CNA's string representation is CNA's own and is not required to be either",
	},
	{
		Route:  "cna_graphics_resource_copy_string",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::ToString()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "the second half of the same read",
	},
	{
		Route:  "cna_graphics_resource_dispose",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Dispose()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "measured to be a FLAG AND A NOTIFICATION rather than a release: after it, cna_texture2d_copy_encoded still produced 70 bytes from the same texture. The reference's Dispose(true) releases the native object and then sets the flag, which is what CNA-Go's per-kind destroy does; calling this one as well would set a CNA flag on a resource CNA-Go is about to destroy anyway",
	},
	{
		Route:  "cna_graphics_resource_subscribe_disposing",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Disposing()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "the settled event rule requires a projected XNA event to have its AUTHORITATIVE raise path, and Disposing's is managed: ~GraphicsResource invokes the delegate field directly. CNA's notification fires from cna_graphics_resource_dispose, a different moment CNA-Go does not reach -- the same shape as Game::Disposed, whose native signal is bound LIFECYCLE_ONLY and raises nothing public",
	},
	{
		Route:  "cna_graphics_resource_unsubscribe_disposing",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::Disposing()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "the removal half of a subscription CNA-Go does not make",
	},
	// Foundation 69 -- SpriteFont, measured with build-probe/f69-spritefont.c
	// against the pinned 0.21.0 artifact on the HEADLESS renderer, over a
	// three-glyph `.cnj` font whose 'B' carries a NEGATIVE bearing on each
	// side:
	//
	//	'?' kerning (1, 4, 2)   crop height 8
	//	'A' kerning (0, 5, 0)   crop height 8
	//	'B' kerning (-3, 6, -2) crop height 12   lineSpacing 10, spacing 1
	//
	//	text     cna_sprite_font_measure_utf8   InternalMeasure from the IL
	//	"A"      (5, 10)                        (5, 10)     agree
	//	"?"      (7, 10)                        (7, 10)     agree
	//	"B"      (4, 12)                        (6, 12)     DIVERGE
	//	"AB"     (7, 12)                        (9, 12)     DIVERGE
	//	"BA"     (10, 12)                       (10, 12)    agree
	//	"AB\nA"  (7, 20)                        (9, 20)     DIVERGE
	//	"Z"      CNA result 1                   ArgumentException
	//
	// Every divergence is the same two pixels and the same cause: the
	// reference's LAST statement over the width is
	//
	//	result.X += Math.Max(rightBearing, 0f);
	//
	// and CNA adds the final glyph's right bearing UNCLAMPED. The two agree
	// wherever the last glyph's right bearing is non-negative, which is why
	// "BA" agrees and "AB" does not. CNA does clamp the FIRST glyph's left
	// bearing, so that half of the algorithm matches.
	{
		Route:  "cna_sprite_font_measure_utf8",
		Member: "Microsoft.Xna.Framework.Graphics.SpriteFont::MeasureString(System.String)",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "measured to disagree with SpriteFont::InternalMeasure whenever the last glyph's right bearing is negative: over a font whose 'B' is kerning (-3, 6, -2), CNA measured \"B\" as 4 and the reference's own algorithm as 6, because the reference's final statement is `result.X += Math.Max(rightBearing, 0f)` and CNA adds the bearing unclamped. It also answers CNA result 1 for a character the font has no glyph and no default character for, where the reference throws ArgumentException carrying FrameworkResources.CharacterNotInFont. CNA-Go runs the reference's algorithm over the glyph table cna_sprite_font_copy_glyphs reports, so the DATA is CNA's and the arithmetic is the reference's",
	},
	// Foundation 72. Foundation 60 bound this one for the two effect-free Begin
	// overloads; Foundation 72 added the other two and moved all four to
	// cna_sprite_batch_begin_with_effect.
	{
		Route:  "cna_sprite_batch_begin_with_states",
		Member: "Microsoft.Xna.Framework.Graphics.SpriteBatch::Begin(Microsoft.Xna.Framework.Graphics.SpriteSortMode,Microsoft.Xna.Framework.Graphics.BlendState)",
		Class:  "SUBSUMED",
		Detail: "cna_sprite_batch_begin_with_effect takes the same four state descriptors plus an effect handle and a transform, and CNA documents CNA_INVALID_HANDLE as selecting the stock sprite effect and a null transform as the identity -- which is exactly what the two effect-free overloads supply in the reference, where EVERY Begin funnels into the seven-argument one. Binding both would give one reference body two native paths that could drift",
	},
	{
		Route:  "cna_effect_parameter_get_value_texture",
		Member: "Microsoft.Xna.Framework.Graphics.EffectParameter::GetValueTexture2D()",
		Class:  "REPRESENTATION",
		Detail: "the route reports \"the retained handle or invalid handle for null\", and the reference's three texture getters return the SAME Texture2D, Texture3D or TextureCube OBJECT the setter was given -- it stores the managed reference alongside the D3DX handle. A handle cannot carry that identity, and building a fresh facade over it would hand back a different object with the same native half, so `p.GetValueTexture2D() == myTexture` would silently become false. The three getters refuse with a message that says so, without reaching CNA: calling the route and discarding its answer would bind it to produce a value nothing can use. The SETTER, cna_effect_parameter_set_value_texture, is bound and used -- the direction that loses nothing",
	},
	{
		Route:  "cna_sprite_font_copy_characters",
		Member: "Microsoft.Xna.Framework.Graphics.SpriteFont::Characters()",
		Class:  "REDUNDANT_READ",
		Detail: "measured to report exactly the character column of cna_sprite_font_copy_glyphs, in the same order and the same count, which CNA's own documentation states and the probe confirmed. The reference's characterMap is ONE list that get_Characters views and GetIndexForCharacter binary-searches, and reading it from two routes could produce two lists whose indices no longer correspond -- which is the invariant every other member depends on",
	},
	// Foundation 83 -- OcclusionQuery. One row: the type binds six of CNA's
	// eight routes and the two it leaves are different cases.
	{
		Route:  "cna_occlusion_query_has_renderer",
		Member: "Microsoft.Xna.Framework.Graphics.OcclusionQuery::IsDisposed()",
		Class:  "MANAGED_REFERENCE",
		Detail: "IsDisposed is GraphicsResource's `ldarg.0; ldfld isDisposed; ret`, a managed flag the reference sets in Dispose and reads with no native call. This route reports whether the native query object is alive, which is a DIFFERENT question: a query CNA-Go has disposed has neither, but a query whose renderer went away without a Dispose would answer false here and false for IsDisposed, and the reference's answer is the flag. Binding it would let a renderer event decide a managed property",
	},
	// cna_occlusion_query_get_is_pixel_count_precise_ext adds no row: it backs
	// no member of the pinned contract. XNA's OcclusionQuery declares
	// PixelCount and says nothing about whether the count is a tally or a flag,
	// so there is no projected member the route could have served.

	// Foundation 81 -- EnvironmentMapEffect and SkinnedEffect, closing the stock
	// effect family. The pattern is unchanged: a getter beside a bound setter is
	// unbound because the reference reads its own field, and a texture getter is
	// unbound because a handle cannot carry object identity.
	//
	// SkinnedEffect's cna_skinned_effect_get_vertex_color_enabled and
	// cna_skinned_effect_set_vertex_color_enabled add no row: the pinned
	// contract declares NO VertexColorEnabled on that type, so they back no
	// projected member and are not routes a projected member could have used.
	{
		Route:  "cna_environment_map_effect_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, and the same alpha divergence every stock effect's diffuse colour has: what OnApply pushes is the colour multiplied by alpha",
	},
	{
		Route:  "cna_environment_map_effect_get_emissive_color",
		Member: "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect::EmissiveColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_environment_map_effect_get_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect::Alpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_environment_map_effect_get_texture",
		Member: "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect::Texture()",
		Class:  "REPRESENTATION",
		Detail: "the handle/object obstacle recorded for cna_basic_effect_get_texture, on the same terms",
	},
	{
		Route:  "cna_environment_map_effect_get_environment_map",
		Member: "Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect::EnvironmentMap()",
		Class:  "REPRESENTATION",
		Detail: "the same obstacle one texture family over: the property's value is a TextureCube OBJECT and the route reports a handle. It is the only TextureCube-valued property in the profile, and the projection holds the object the setter was given for the reason every other stock-effect texture getter does",
	},
	{
		Route:  "cna_skinned_effect_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, with the alpha divergence",
	},
	{
		Route:  "cna_skinned_effect_get_emissive_color",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::EmissiveColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_skinned_effect_get_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::Alpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_skinned_effect_get_prefer_per_pixel_lighting",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::PreferPerPixelLighting()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, and the property is a PREFERENCE: the reference reports what it stored, never what the device could do",
	},
	{
		Route:  "cna_skinned_effect_get_weights_per_vertex",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::WeightsPerVertex()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld` of a value the SETTER has already validated against {1, 2, 4} -- so the projection knows the stored value is legal and CNA's answer could only disagree with it",
	},
	{
		Route:  "cna_skinned_effect_get_texture",
		Member: "Microsoft.Xna.Framework.Graphics.SkinnedEffect::Texture()",
		Class:  "REPRESENTATION",
		Detail: "the handle/object obstacle, on the same terms as every other stock-effect texture getter",
	},
	// Foundation 80 -- AlphaTestEffect and DualTextureEffect, the same pattern
	// Foundation 79 recorded one type over. Every getter CNA declares beside a
	// bound setter is here, because the reference reads its own field; the two
	// texture getters are the object-identity obstacle instead.
	//
	// EffectMaterial adds no row. Its two `_ext` routes back no member of the
	// pinned contract -- XNA's EffectMaterial declares one constructor and
	// nothing else -- so they are not routes a projected member could have
	// used, which is what this registry is about.
	{
		Route:  "cna_alpha_test_effect_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_DiffuseColor is `ldarg.0; ldfld diffuseColor; ret`, and what OnApply pushes is that colour multiplied by alpha -- so the reference's own getter and its own push disagree, and binding the read would answer with the pushed value. The SETTER's push is bound",
	},
	{
		Route:  "cna_alpha_test_effect_get_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::Alpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`; the reference never reads alpha back from the effect",
	},
	{
		Route:  "cna_alpha_test_effect_get_vertex_color_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::VertexColorEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`; the setter selects a shader permutation and nothing reads the flag back",
	},
	{
		Route:  "cna_alpha_test_effect_get_alpha_function",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::AlphaFunction()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld` of a CompareFunction the setter stored WITHOUT validating. The reference reports back an undeclared value unchanged, and CNA's own route would report whatever it clamped or substituted -- which is the one thing this getter must not do",
	},
	{
		Route:  "cna_alpha_test_effect_get_reference_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::ReferenceAlpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, and the same non-validating store: a reference alpha outside 0..255 is kept and reported back",
	},
	{
		Route:  "cna_alpha_test_effect_get_texture",
		Member: "Microsoft.Xna.Framework.Graphics.AlphaTestEffect::Texture()",
		Class:  "REPRESENTATION",
		Detail: "the handle/object obstacle cna_basic_effect_get_texture is recorded for, on the same terms: the property's value is a Texture2D OBJECT, the effect is its sole writer, and holding the object the setter was given reproduces the reference's observable where a fresh facade over a handle would not",
	},
	{
		Route:  "cna_dual_texture_effect_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.DualTextureEffect::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "the same measurement as AlphaTestEffect's, including the alpha divergence",
	},
	{
		Route:  "cna_dual_texture_effect_get_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.DualTextureEffect::Alpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_dual_texture_effect_get_vertex_color_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.DualTextureEffect::VertexColorEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`",
	},
	{
		Route:  "cna_dual_texture_effect_get_texture",
		Member: "Microsoft.Xna.Framework.Graphics.DualTextureEffect::Texture()",
		Class:  "REPRESENTATION",
		Detail: "the same obstacle for BOTH layers: the route takes a layer index and reports a handle, and the two properties' values are objects. The indexed SETTER is bound and backs Texture at index 0 and Texture2 at index 1",
	},
	// Foundation 79 -- the stock-effect family. It bound 28 of the 49 routes the
	// BasicEffect cluster declares and records the other 21 here, and they fall
	// into one pattern with two exceptions.
	//
	// The pattern is that BasicEffect and DirectionalLight are MANAGED STATE in
	// the reference. Almost every getter is `ldarg.0; ldfld <field>; ret`, and
	// the state reaches the effect only when OnApply pushes it -- so the
	// SETTERS have native counterparts worth binding and the GETTERS do not.
	// Reading a value back from CNA where the reference reads a field would let
	// CNA's clamping, or CNA's derived quantity, answer an XNA getter.
	//
	// The two exceptions are get_texture, which is the object-identity obstacle
	// the effect-parameter texture getter already carries, and
	// enable_default_lighting, which is a preset CNA owns and XNA specifies.
	{
		Route:  "cna_effect_matrices_get_world",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::World()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_World is `ldarg.0; ldfld world; ret`, seven bytes over a managed field the setter stored, and set_World is a store plus a dirty-flag `or`. The reference reads nothing back: it PUSHES in OnApply, and what it pushes is world*view*projection rather than the world. Binding this getter would let CNA's stored or clamped matrix answer a getter the reference answers from its own field. The SETTER is bound and is what OnApply pushes through",
	},
	{
		Route:  "cna_effect_matrices_get_view",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::View()",
		Class:  "MANAGED_REFERENCE",
		Detail: "the same measurement one field over; set_View additionally raises the eye-position and fog flags because both are computed from the view",
	},
	{
		Route:  "cna_effect_matrices_get_projection",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::Projection()",
		Class:  "MANAGED_REFERENCE",
		Detail: "the same measurement; set_Projection is 22 bytes against the other two setters' 23 because it ORs one flag fewer",
	},
	{
		Route:  "cna_basic_effect_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_DiffuseColor is `ldarg.0; ldfld diffuseColor; ret`. What OnApply writes is not this value but this value multiplied by alpha, so the reference's own getter and its own push already disagree -- binding the read would answer with the PUSHED value where the reference answers with the STORED one",
	},
	{
		Route:  "cna_basic_effect_get_emissive_color",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::EmissiveColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "the same measurement and the same alpha divergence",
	},
	{
		Route:  "cna_basic_effect_get_alpha",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::Alpha()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_Alpha is `ldarg.0; ldfld alpha; ret`, and the reference never reads it back from the effect",
	},
	{
		Route:  "cna_basic_effect_get_prefer_per_pixel_lighting",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::PreferPerPixelLighting()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_PreferPerPixelLighting is one `ldfld`, and the property is a PREFERENCE: the reference reports what it stored, never what the device could do. Binding the read would make it report the renderer's answer, which is the one thing the member is documented not to do",
	},
	{
		Route:  "cna_basic_effect_get_vertex_color_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::VertexColorEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`; the setter selects a shader permutation and raises ShaderIndex, and nothing reads the flag back",
	},
	{
		Route:  "cna_basic_effect_get_texture_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::TextureEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, the same permutation shape",
	},
	{
		Route:  "cna_effect_lights_get_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::LightingEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_LightingEnabled is one `ldfld` in all five shipped implementors, which is what Foundation 18 measured when it declared IEffectLights' accessors infallible",
	},
	{
		Route:  "cna_effect_lights_get_ambient_color",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::AmbientLightColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld` in the three implementors that declare it -- BasicEffect, SkinnedEffect and EnvironmentMapEffect",
	},
	{
		Route:  "cna_effect_fog_get_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::FogEnabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`; Foundation 18 already recorded that FogEnabled, FogStart and FogEnd are the managed three of IEffectFog's four properties and FogColor the one that crosses",
	},
	{
		Route:  "cna_effect_fog_get_start",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::FogStart()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`; the reference pushes a computed fog VECTOR rather than the start, so the route and the getter do not carry the same quantity",
	},
	{
		Route:  "cna_effect_fog_get_end",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::FogEnd()",
		Class:  "MANAGED_REFERENCE",
		Detail: "one `ldfld`, the same divergence",
	},
	{
		Route:  "cna_basic_effect_get_texture",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::Texture()",
		Class:  "REPRESENTATION",
		Detail: "the route reports presence plus a HANDLE and the property's value is a Texture2D OBJECT -- the same obstacle cna_effect_parameter_get_value_texture is recorded for, answered differently because the position is different. An EffectParameter is a view anything holding another view can write, so a cache in the view could go stale; BasicEffect::Texture is a property of the effect and the Foundation 79 probe measured CNA publishing no parameters at all, so the object the setter was given IS the current value and holding it reproduces the reference's observable exactly where refusing would not. The SETTER is bound",
	},
	{
		Route:  "cna_effect_lights_enable_default",
		Member: "Microsoft.Xna.Framework.Graphics.BasicEffect::EnableDefaultLighting()",
		Class:  "CONTRACT_DIVERGENCE",
		Detail: "the route applies CNA's own three-point preset and EffectHelpers::EnableDefaultLighting applies a MEASURED one: thirteen calls with `ldc.r4` operands giving light 0 direction (-0.5265408, -0.5735765, -0.6275069) and colour (1, 0.9607844, 0.8078432), light 1 direction (0.7198464, 0.3420201, 0.6040227) with zero specular, light 2 direction (0.4545195, -0.7660444, 0.4545195), and an ambient return of (0.05333332, 0.09882354, 0.1819608). Those vectors are the contract, and calling CNA's preset would make a native default answer for an XNA behaviour -- nothing learned from CNA may become an XNA behaviour golden",
	},
	{
		Route:  "cna_directional_light_get_direction",
		Member: "Microsoft.Xna.Framework.Graphics.DirectionalLight::Direction()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_Direction is `ldarg.0; ldfld cachedDirection; ret`. All four of DirectionalLight's getters read a CACHE and all four of its setters write through an EffectParameter, so the type reads managed and writes native -- the setters are bound and the getters are not",
	},
	{
		Route:  "cna_directional_light_get_diffuse_color",
		Member: "Microsoft.Xna.Framework.Graphics.DirectionalLight::DiffuseColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_DiffuseColor reads cachedDiffuseColor, and the divergence is observable rather than theoretical: set_Enabled(false) writes Vector3.Zero into the PARAMETER and leaves the cache alone, so a disabled light's property still reports the colour it was given while the light itself holds zero",
	},
	{
		Route:  "cna_directional_light_get_specular_color",
		Member: "Microsoft.Xna.Framework.Graphics.DirectionalLight::SpecularColor()",
		Class:  "MANAGED_REFERENCE",
		Detail: "the same measurement and the same disabled-light divergence",
	},
	{
		Route:  "cna_directional_light_get_enabled",
		Member: "Microsoft.Xna.Framework.Graphics.DirectionalLight::Enabled()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_Enabled is `ldarg.0; ldfld enabled; ret`, and the flag exists only managed-side in the reference: the shader has no enable, which is why set_Enabled expresses disabling as two zero colours",
	},
	{
		Route:  "cna_directional_light_create",
		Member: "Microsoft.Xna.Framework.Graphics.DirectionalLight::.ctor(Microsoft.Xna.Framework.Graphics.EffectParameter,Microsoft.Xna.Framework.Graphics.EffectParameter,Microsoft.Xna.Framework.Graphics.EffectParameter,Microsoft.Xna.Framework.Graphics.DirectionalLight)",
		Class:  "MANAGED_REFERENCE",
		Detail: "the route makes a free-standing native light, and a free-standing light is exactly the case in which the reference's constructor reaches nothing: every setter it calls is guarded by `brfalse` on its EffectParameter, and a publicly constructed light's parameters are whatever the caller passed. The Foundation 79 probe measured CNA's stock BasicEffect publishing PARAMETER_COUNT 0 on both qualified artifacts, so a caller has none to pass and the object is a pure cache -- which the projection already is. A native light no effect reads would change nothing observable",
	},
	{
		Route:  "cna_graphics_resource_get_graphics_device",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::GraphicsDevice()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_GraphicsDevice is `ldarg.0; ldfld _parent; ret`, a stored reference with no disposal check, and it must answer with the SAME GraphicsDevice object the resource was created on. CNA's route returns a fresh callback-scoped handle, which is only valid inside a lifecycle callback and would not be the identity the reference returns",
	},

	// Foundation 84. The dynamic buffers bound exactly one new route,
	// cna_vertex_buffer_set_data_raw_at_with_options, and its offsetless
	// sibling is what that leaves behind.
	{
		Route:  "cna_vertex_buffer_set_data_raw_with_options",
		Member: "Microsoft.Xna.Framework.Graphics.DynamicVertexBuffer::SetData(!!0[],System.Int32,System.Int32,Microsoft.Xna.Framework.Graphics.SetDataOptions)",
		Class:  "SUBSUMED",
		Detail: "the bound cna_vertex_buffer_set_data_raw_at_with_options is the same upload with a buffer offset, and CNA's own header says the offsetless route \"matches\" the windowed one at offset zero. The reference has ONE body for both overloads -- SetData(T[],int,int,SetDataOptions) is `ldarg.0; ldc.i4.0; ldarg.1; ... call SetData(int,T[],int,int,int,SetDataOptions)` -- so binding both would give one reference path two native paths that could drift",
	},
}

// verifyUnboundRoutes checks the registry against the canonical headers and the
// manifest. It returns the number of recorded routes.
func verifyUnboundRoutes(result *report, declared map[string]struct{}, bound map[string]struct{}) int {
	seen := make(map[string]bool, len(deliberatelyUnboundRoutes))
	for _, entry := range deliberatelyUnboundRoutes {
		if seen[entry.Route] {
			result.Findings = append(result.Findings,
				fmt.Sprintf("deliberately unbound route %s is recorded twice", entry.Route))
			continue
		}
		seen[entry.Route] = true
		if _, known := unboundRouteClasses[entry.Class]; !known {
			result.Findings = append(result.Findings,
				fmt.Sprintf("deliberately unbound route %s has unrecorded class %q", entry.Route, entry.Class))
		}
		if strings.TrimSpace(entry.Detail) == "" || strings.TrimSpace(entry.Member) == "" {
			result.Findings = append(result.Findings,
				fmt.Sprintf("deliberately unbound route %s records no member or no detail", entry.Route))
		}
		if len(declared) > 0 {
			if _, exists := declared[entry.Route]; !exists {
				result.Findings = append(result.Findings,
					fmt.Sprintf("deliberately unbound route %s is not declared by the canonical headers", entry.Route))
			}
		}
		if _, isBound := bound[entry.Route]; isBound {
			result.Findings = append(result.Findings,
				fmt.Sprintf("route %s is recorded as deliberately unbound and the manifest binds it", entry.Route))
		}
	}
	return len(deliberatelyUnboundRoutes)
}

func countCanonicalDeclarations(root string) (int, error) {
	names, err := canonicalDeclarations(root)
	return len(names), err
}

// canonicalDeclarations is every cna_* route the canonical headers declare.
func canonicalDeclarations(root string) (map[string]struct{}, error) {
	directory := filepath.Join(root, "CNA", "C")
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	names := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".h") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		for _, match := range declarationPattern.FindAllString(string(data), -1) {
			names[strings.TrimRight(strings.TrimSuffix(strings.TrimSpace(match), "("), " \t")] = struct{}{}
		}
	}
	return names, nil
}

// countLibraryExports reads the dynamic symbol table. It reports zero when the
// platform tool is unavailable, and zero is not treated as a mismatch.
func countLibraryExports(path string) int {
	output, err := exec.Command("nm", "-D", "--defined-only", path).Output()
	if err != nil {
		return 0
	}
	names := map[string]struct{}{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[1] != "T" {
			continue
		}
		name := fields[2]
		if index := strings.Index(name, "@"); index >= 0 {
			name = name[:index]
		}
		if strings.HasPrefix(name, "cna_") {
			names[name] = struct{}{}
		}
	}
	return len(names)
}

func writeReport(path string, result report) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
