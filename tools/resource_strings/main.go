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
// Foundation 49 closed that with a SUBSTRING search over the raw assembly
// bytes. Foundation 50 replaced it with a real read of the assembly's resource
// sets, keyed by name -- see resources.go -- because a substring search checks
// only half the claim. It proves the sentence is Microsoft's; it cannot prove
// it is filed under the key the reference's throw site calls. The audio-emitter
// message was recorded under `DopplerScaleMustBeGreaterThanOrEqualToZero`, a
// key that exists nowhere; the string is real and its key is
// `InvalidEmitterDopplerScale`. Four milestones of substring searches passed
// it.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	// Foundation 74 and 76. The four .NET Framework 4.0 BCL messages CNA-Go
	// reproduces, read from the pinned mscorlib the retained XNA assemblies
	// bind against -- the same binary every "sha256 5634668d..." claim in this
	// repository names, and admitted here by that hash.
	//
	// Exception_WasThrown carries the exception's class name through
	// String.Format's {0}, which CNA-Go spells %s.
	{Key: "Arg_KeyNotFound", Assembly: "mscorlib.dll",
		Value: "The given key was not present in the dictionary."},
	{Key: "Argument_AddingDuplicate", Assembly: "mscorlib.dll",
		Value: "An item with the same key has already been added."},
	{Key: "Exception_WasThrown", Assembly: "mscorlib.dll",
		Value: "Exception of type '%s' was thrown.", Placeholders: true},
	{Key: "Exception_EndOfInnerExceptionStack", Assembly: "mscorlib.dll",
		Value: "--- End of inner exception stack trace ---"},
	{Key: "Arg_ExternalException", Assembly: "mscorlib.dll",
		Value: "External component has thrown an exception."},

	// Foundation 75. GraphicsDeviceInformation::set_Adapter, and the two
	// NoSuitableGraphicsDeviceException messages FindBestPlatformDevice throws.
	// NoCompatibleDevices carries the GraphicsProfile through String.Format's
	// {0}, and its line breaks are CRLF.
	{Key: "NoNullUseDefaultAdapter", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Adapter cannot be null.  Try using GraphicsAdapter.DefaultAdapter instead."},
	{Key: "NoCompatibleDevices", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Could not find a Direct3D device that supports the XNA Framework %s profile.\r\n\r\n" +
			"Verify that a suitable graphics device is installed.\r\n\r\n" +
			"Make sure the desktop is not locked, and that no other application is running in full screen mode.\r\n\r\n" +
			"Avoid running under Remote Desktop or as a Windows service.\r\n\r\n" +
			"Check the display properties to make sure hardware acceleration is set to Full.", Placeholders: true},
	{Key: "NoCompatibleDevicesAfterRanking", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "The process of ranking devices removed all compatible devices."},
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
	// Foundation 50. SpriteBatch's four throw sites, all in
	// Microsoft.Xna.Framework.dll rather than the Graphics assembly the type
	// lives in: FrameworkResources is in the shared one.
	//
	// The last two read alike and are different sentences at different throw
	// sites -- one is about calling End without a Begin and the other about
	// calling Begin twice -- which is exactly the pair a substring search could
	// not have told apart.
	{Key: "NullNotAllowed", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "This method does not accept null for this parameter."},
	{Key: "BeginMustBeCalledBeforeDraw", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Begin must be called successfully before a Draw can be called."},
	{Key: "BeginMustBeCalledBeforeEnd", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Begin must be called successfully before End can be called."},
	{Key: "EndMustBeCalledBeforeBegin", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Begin cannot be called again until End has been successfully called."},
	// Foundation 52. The one guard Texture2D's constructors reproduce before
	// they reach CNA; everything after it is CNA's own refusal, because
	// reproducing D3D9's format-capability messages would mean asserting a
	// support decision CNA-Go did not make.
	// The state-object freeze. Its value carries `{0}` TWICE and Placeholders is
	// deliberately NOT set: that flag converts Go's `%s` into `{0}`, `{1}`, and
	// this message substitutes the SAME argument at both positions. CNA-Go
	// therefore keeps the CLR's own spelling in the source constant and formats
	// it with strings.ReplaceAll, so the registry compares the exact bytes the
	// assembly holds.
	{Key: "BoundStateObject", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Cannot change read-only {0}. State objects become read-only the first time they are bound to a GraphicsDevice. To change property values, create a new {0} instance."},
	{Key: "DeviceCannotBeNullOnResourceCreate", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The GraphicsDevice must not be null when creating new resources."},
	// Foundation 64. VertexElementValidator's five messages, which are the
	// whole failure surface of VertexDeclaration's two constructors.
	//
	// Placeholders is deliberately NOT set on the four formatted ones, for the
	// reason BoundStateObject's is not: each substitutes a usage and a usage
	// index with NO separator between them, and the overlap message names two
	// elements with four placeholders. CNA-Go keeps the CLR spelling in the
	// source constant and substitutes positionally, so the registry compares
	// the exact bytes the assembly holds.
	//
	// The first is thrown at TWO sites for TWO quantities -- the stride and one
	// element's offset -- and the sentence names both, which is why one message
	// covers both throws rather than the projection inventing a second.
	{Key: "VertexElementOffsetNotMultipleFour", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid VertexDeclaration. Vertex stride and VertexElement.Offset must be multiples of four."},
	{Key: "VertexElementBadUsage", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid VertexDeclaration. Usage {0}{1} is out of range."},
	{Key: "VertexElementOutsideStride", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid VertexDeclaration. Element {0}{1} does not fit within the specified vertex stride."},
	{Key: "DuplicateVertexElement", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid VertexDeclaration. Duplicate element {0}{1}."},
	{Key: "VertexElementsOverlap", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid VertexDeclaration. Elements {0}{1} and {2}{3} are overlapping."},
	// Foundation 65. IndexBuffer's four. MustBeValidIndex is the shared one:
	// Helpers::ValidateCopyParameters throws it three times with three
	// different PARAMETER names and one sentence, and every buffer transfer in
	// the reference goes through that helper.
	{Key: "ResourcesMustBeGreaterThanZeroSize", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Resource size must be greater than zero."},
	{Key: "WriteOnlyGetNotSupported", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Calling GetData on a resource that was created with BufferUsage.WriteOnly is not supported."},
	{Key: "ResourceDataMustBeCorrectSize", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The array is not the correct size for the amount of data requested."},
	{Key: "MustBeValidIndex", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "This parameter must be a valid index within the array."},
	// Foundation 66. VertexBuffer's stride refusal and FromType's four. The
	// four are formatted with the type's name at `{0}`, so Placeholders is not
	// set for the same reason the vertex-element messages do not set it.
	{Key: "VertexStrideTooSmall", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The vertex stride is too small for the type of data requested. This is not allowed."},
	{Key: "VertexTypeNotValueType", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid vertex type. {0} is not a value type."},
	{Key: "VertexTypeNotIVertexType", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid vertex type. {0} does not implement the IVertexType interface."},
	{Key: "VertexTypeNullDeclaration", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid vertex type. {0} returned a null VertexDeclaration."},
	{Key: "VertexTypeWrongSize", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Invalid vertex type. The size of {0} does not match the stride of its vertex declaration."},
	// Foundation 73. The one message GetBackBufferData adds; its other guard,
	// the ProfileCapabilities one, is not reproduced.
	{Key: "CannotGetBackBufferActiveRenderTargets", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Cannot use GetBackBufferData when a render target is active."},
	// Foundation 73. The one message the six DrawUser* generics add. The other
	// three they throw -- NullNotAllowed, MustDrawSomething and
	// MustBeValidIndex -- are already here, claimed by earlier members.
	{Key: "OffsetNotValid", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The offset must be within the valid range for this resource."},
	// Foundation 72. The one message the Effect cluster throws with a resource
	// string. Everything else it throws is parameterless -- all 49
	// InvalidCastException sites in EffectParameter are
	// `newobj InvalidCastException::.ctor()`, the array getters'
	// ArgumentOutOfRangeException names no parameter, and
	// set_CurrentTechnique's parent check is a bare InvalidOperationException.
	{Key: "NotCurrentTechnique", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Cannot Apply an EffectPass that is not from the CurrentTechnique."},
	// Foundation 69. SpriteFont's one message, thrown from TWO sites with two
	// exception shapes: set_DefaultCharacter's ArgumentException(message) and
	// GetIndexForCharacter's ArgumentException(message, "character").
	//
	// Placeholders is deliberately NOT set, for the reason BoundStateObject's
	// is not: both placeholders take the SAME argument -- boxed as Char at {0}
	// and as Int32 at {1} -- and {1} carries the `x4` format specifier Go has
	// no `%`-verb equivalent for. CNA-Go keeps the CLR spelling in the source
	// constant and substitutes positionally, so the registry compares the exact
	// bytes the assembly holds.
	{Key: "CharacterNotInFont", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The character '{0}' (0x{1:x4}) is not available in this SpriteFont. If applicable, adjust the font's start and end CharacterRegions to include this character."},
	// Foundation 67. The two the draw members throw. The second is the one a
	// consumer can trip without a shader: it refuses a NON-instanced draw while
	// any bound stream carries a non-zero instance frequency.
	{Key: "MustDrawSomething", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "When drawing, at least one primitive must be drawn."},
	{Key: "NonZeroInstanceFrequency", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "Non-instanced draw calls are not valid when a vertex buffer is bound with a non-zero instance frequency."},
	// Foundation 50 corrected this KEY. The value was right and its key was
	// invented from the sentence; the resource-set reader found no
	// DopplerScaleMustBeGreaterThanOrEqualToZero anywhere, and the real key is
	// InvalidEmitterDopplerScale.
	{Key: "InvalidEmitterDopplerScale", Assembly: "Microsoft.Xna.Framework.dll",
		Value: "The doppler scale of an audio emitter must be greater than or equal to zero."},
	{Key: "GameCannotBeNull", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "Game cannot be null."},
	{Key: "GraphicsDeviceManagerAlreadyPresent", Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value: "A graphics device manager is already registered.  The graphics device manager cannot be changed once it is set."},
}

// constantStringValue folds one constant expression to its string value.
//
// It handles a `+` chain of string literals as well as a single one, because a
// long Microsoft message is spelled across several source lines and would
// otherwise be invisible to this scan -- which is the one way an unverified
// claim could get in. The messageShape filter then applies to the WHOLE folded
// sentence, so a multi-line message is checked exactly as a one-line one is.
func constantStringValue(expression ast.Expr) (string, bool) {
	switch typed := expression.(type) {
	case *ast.BasicLit:
		if typed.Kind != token.STRING {
			return "", false
		}
		text, err := strconv.Unquote(typed.Value)
		if err != nil {
			return "", false
		}
		return text, true
	case *ast.BinaryExpr:
		if typed.Op != token.ADD {
			return "", false
		}
		left, leftOK := constantStringValue(typed.X)
		right, rightOK := constantStringValue(typed.Y)
		if !leftOK || !rightOK {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return constantStringValue(typed.X)
	default:
		return "", false
	}
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
	bcl := flag.String("bcl", filepath.Join(os.Getenv("HOME"), "deps", "bcl-4.0-pinned"),
		"directory holding the pinned .NET Framework 4.0 BCL the XNA assemblies bind against")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	result, err := run(*assemblies, *bcl, *root)
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

func run(assemblyRoot, bclRoot, repositoryRoot string) (report, error) {
	result := report{Claimed: len(registry)}
	blobs, err := loadAssemblies(assemblyRoot, bclRoot)
	if err != nil {
		return result, err
	}
	result.Assemblies = len(blobs)
	result.Findings = append(result.Findings, verifyAdmittedHashes(blobs)...)
	for _, entry := range registry {
		blob, present := blobs[entry.Assembly]
		if !present {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s names assembly %s, which is not retained", entry.Key, entry.Assembly))
			continue
		}
		set, err := resourceStrings(blob)
		if err != nil {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s names assembly %s, whose resource sets could not be read: %v", entry.Key, entry.Assembly, err))
			continue
		}
		actual, present := set[entry.Key]
		if !present {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s is not a resource key in %s", entry.Key, entry.Assembly))
			continue
		}
		if actual != clrSpelling(entry) {
			result.Findings = append(result.Findings,
				fmt.Sprintf("%s in %s is %q, and the source claims %q", entry.Key, entry.Assembly, actual, entry.Value))
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

func loadAssemblies(roots ...string) (map[string][]byte, error) {
	blobs := map[string][]byte{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("retained assemblies are not available at %s: %w", root, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dll") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, entry.Name()))
			if err != nil {
				return nil, err
			}
			if existing, present := blobs[entry.Name()]; present && !bytes.Equal(existing, data) {
				return nil, fmt.Errorf("two admitted roots carry different %s", entry.Name())
			}
			blobs[entry.Name()] = data
		}
	}
	return blobs, nil
}

// admittedAssemblyHashes pins the identity of every retained binary a claimed
// message may be read from that is NOT one of the XNA assemblies -- which are
// pinned by their own provenance file.
//
// mscorlib is here because Foundation 76 needs it: the exception family's
// default message and its inner-exception separator are BCL resource strings,
// not XNA ones, and every "read from the pinned mscorlib" claim in this
// repository names this exact sha256. Admitting the binary without checking its
// hash would make the claim unfalsifiable.
var admittedAssemblyHashes = map[string]string{
	"mscorlib.dll": "5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63",
}

// verifyAdmittedHashes rejects a retained binary whose identity is pinned and
// whose content does not match it.
func verifyAdmittedHashes(blobs map[string][]byte) []string {
	var findings []string
	for name, want := range admittedAssemblyHashes {
		data, present := blobs[name]
		if !present {
			findings = append(findings, fmt.Sprintf("%s is admitted by hash but is not retained", name))
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != want {
			findings = append(findings, fmt.Sprintf("%s has sha256 %s, and the admitted identity is %s", name, got, want))
		}
	}
	return findings
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
						text, ok := constantStringValue(expression)
						if !ok {
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
