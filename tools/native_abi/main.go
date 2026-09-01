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
	{
		Route:  "cna_graphics_resource_get_graphics_device",
		Member: "Microsoft.Xna.Framework.Graphics.GraphicsResource::GraphicsDevice()",
		Class:  "MANAGED_REFERENCE",
		Detail: "get_GraphicsDevice is `ldarg.0; ldfld _parent; ret`, a stored reference with no disposal check, and it must answer with the SAME GraphicsDevice object the resource was created on. CNA's route returns a fresh callback-scoped handle, which is only valid inside a lifecycle callback and would not be the identity the reference returns",
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
