// Command resource_strings verifies that every exact Microsoft resource message
// CNA-Go claims appears VERBATIM in a retained XNA 4.0 assembly.
//
// # Why it exists
//
// CNA-Go reproduces exception messages byte for byte, and the IL disassembly in
// ~/deps/xna-il-cache/ cannot supply them: it shows the `Resources::get_<Key>()`
// call and never the value the key names. So a message has to be read out of the
// assembly's .resources stream -- and until Foundation 49 nothing checked that
// one had been.
//
// Foundation 48 inferred `BackBufferDimMustBePositive` from its key and wrote
// "The back buffer dimension must be positive." The real string is
// "BackBufferWidth and BackBufferHeight must be greater than zero." It named the
// two properties rather than "the dimension" and said "greater than zero" rather
// than "positive", and nothing in the repository could tell the difference,
// because a plausible sentence looks exactly like a measured one.
//
// This tool closes that. It is deliberately a SUBSTRING search over the raw
// assembly bytes rather than a .resources parser: the streams store
// length-prefixed UTF-8, so a message that is present is present as its own
// bytes, and a parser would be a second thing that can be wrong.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// claimedString is one exact Microsoft message the binding reproduces.
type claimedString struct {
	// Key is the Resources property the reference's throw site calls.
	Key string
	// Assembly is the retained file the string must appear in.
	Assembly string
	// Value is the message, exactly as CNA-Go spells it.
	Value string
	// Placeholders records a deliberate, documented substitution: the CLR uses
	// String.Format's {0}/{1} and Go uses fmt's %s, so the message is compared
	// with the CLR spelling restored.
	Placeholders bool
}

// registry is the closed list. A message constant in the projection packages
// that is not here is a verifier failure, and so is an entry whose value is not
// in its assembly.
var registry = []claimedString{
	{Key: "MissingGraphicsDeviceService", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Drawable components require a graphics device service in the game service container."},
	{Key: "NoGraphicsDeviceService", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "This property requires a graphics device service in the game service container."},
	{Key: "PropertyCannotBeCalledBeforeInitialize", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The GraphicsDevice property cannot be used before Initialize has been called."},
	{Key: "BackBufferDimMustBePositive", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "BackBufferWidth and BackBufferHeight must be greater than zero."},
	{Key: "InactiveSleepTimeCannotBeZero", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The inactive sleep time must be greater than or equal to zero.  Specify zero or a positive value."},
	{Key: "TargetElaspedCannotBeZero", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The target elapsed time must be greater than zero.  Specify a non-zero positive value."},
	{Key: "ServiceAlreadyPresent", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Container already contains a service of this type."},
	{Key: "ServiceMustBeAssignable", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Service provider object of type %s must be assignable to service type %s.", Placeholders: true},
	{Key: "ServiceProviderCannotBeNull", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The service provider instance cannot be null."},
	{Key: "ServiceTypeCannotBeNull", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The service type cannot be null."},
	{Key: "CannotAddSameComponentMultipleTimes", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Cannot add the same game component to a game component collection multiple times."},
	{Key: "CannotSetItemsIntoGameComponentCollection", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Cannot set a value using operator[] on GameComponentCollection.  Use Add/Remove instead."},
	{Key: "DopplerScaleMustBeGreaterThanOrEqualToZero", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The doppler scale of an audio emitter must be greater than or equal to zero."},
	{Key: "GameCannotBeNull", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Game cannot be null."},
	{Key: "GraphicsDeviceManagerAlreadyPresent", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "A graphics device manager is already registered.  The graphics device manager cannot be changed once it is set."},
}

// messageShape is what a claimed reference message looks like in the source: a
// string constant that starts with a capital and reads like a sentence. It is
// deliberately loose, because a false positive costs one registry entry and a
// false negative costs an unverified claim.
var messageShape = regexp.MustCompile(`^[A-Z][^"]{24,}[.]$`)

// projectionPackages are the directories whose string constants are scanned.
var projectionPackages = []string{"Microsoft"}

type report struct {
	Assemblies int
	Claimed    int
	Verified   int
	Scanned    int
	Findings   []string
}

func main() {
	assemblies := flag.String("assemblies", filepath.Join(os.Getenv("HOME"), "deps", "xna40-windows-assemblies"),
		"directory holding the retained XNA 4.0 Windows assemblies")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	result, err := run(*assemblies, *root)
	fmt.Printf("RESOURCE_STRINGS_CLAIMED=%d\n", result.Claimed)
	fmt.Printf("RESOURCE_STRINGS_VERIFIED=%d\n", result.Verified)
	fmt.Printf("RESOURCE_STRINGS_SOURCE_CONSTANTS=%d\n", result.Scanned)
	fmt.Printf("RESOURCE_STRINGS_ASSEMBLIES=%d\n", result.Assemblies)
	fmt.Printf("RESOURCE_STRINGS_FINDINGS=%d\n", len(result.Findings))
	for _, finding := range result.Findings {
		fmt.Fprintln(os.Stderr, "resource-strings:", finding)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "resource-strings:", err)
		os.Exit(1)
	}
	if len(result.Findings) != 0 {
		os.Exit(1)
	}
	fmt.Println("RESOURCE_STRINGS_STATUS=PASS")
}

func run(assemblyRoot, repositoryRoot string) (report, error) {
	result := report{Claimed: len(registry)}
	blobs, err := loadAssemblies(assemblyRoot)
	if err != nil {
		return result, err
	}
	result.Assemblies = len(blobs)
	for _, entry := range registry {
		blob, present := blobs[entry.Assembly]
		if !present {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s names assembly %s, which is not retained", entry.Key, entry.Assembly))
			continue
		}
		if !strings.Contains(string(blob), clrSpelling(entry)) {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s is claimed as %q but that text is not in %s", entry.Key, entry.Value, entry.Assembly))
			continue
		}
		result.Verified++
	}

	claimed := make(map[string]bool, len(registry))
	for _, entry := range registry {
		claimed[entry.Value] = true
	}
	constants, err := scanMessageConstants(repositoryRoot)
	if err != nil {
		return result, err
	}
	result.Scanned = len(constants)
	for _, value := range constants {
		if !claimed[value] {
			result.Findings = append(result.Findings,
				fmt.Sprintf("the source claims %q, which is in no registry entry, so nothing checked it against an assembly", value))
		}
	}
	return result, nil
}

// clrSpelling restores the CLR's own placeholders for a message CNA-Go spells
// with Go's. The substitution is deliberate and documented; comparing the Go
// spelling against the assembly would fail for a reason that is not a defect.
func clrSpelling(entry claimedString) string {
	if !entry.Placeholders {
		return entry.Value
	}
	value := entry.Value
	for index := 0; strings.Contains(value, "%s"); index++ {
		value = strings.Replace(value, "%s", "{"+strconv.Itoa(index)+"}", 1)
	}
	return value
}

func loadAssemblies(root string) (map[string][]byte, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("retained assemblies are not available at %s: %w", root, err)
	}
	blobs := map[string][]byte{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dll") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		blobs[entry.Name()] = data
	}
	return blobs, nil
}

// scanMessageConstants finds every message-shaped string constant in the
// projected packages, so a new claim cannot be added without a registry entry.
// Test files are excluded: a test's expected value is checked by the constant
// it compares against.
func scanMessageConstants(root string) ([]string, error) {
	found := map[string]bool{}
	for _, pkg := range projectionPackages {
		err := filepath.Walk(filepath.Join(root, pkg), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			ast.Inspect(file, func(node ast.Node) bool {
				declaration, ok := node.(*ast.GenDecl)
				if !ok || declaration.Tok != token.CONST {
					return true
				}
				for _, spec := range declaration.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, expression := range value.Values {
						literal, ok := expression.(*ast.BasicLit)
						if !ok || literal.Kind != token.STRING {
							continue
						}
						text, unquoteErr := strconv.Unquote(literal.Value)
						if unquoteErr != nil {
							continue
						}
						if messageShape.MatchString(text) {
							found[text] = true
						}
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	result := make([]string, 0, len(found))
	for value := range found {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}
