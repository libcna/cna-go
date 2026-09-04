package framework

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/openeggbert/cna-go/internal/interop"
)

// TitleContainer is the XNA static title-content type identity, `public
// abstract sealed` in the reference and stateless apart from one private
// character table. Its one member maps to a type-prefixed package declaration.
type TitleContainer struct{}

// TitleContainerOpenStream is
// Microsoft.Xna.Framework.TitleContainer::OpenStream(String), the only member
// of `public abstract sealed class TitleContainer`.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # The reference's 213 bytes are four guards and one open
//
//	if (string.IsNullOrEmpty(name)) throw new ArgumentNullException("name");
//	name = GetCleanPath(name);
//	if (IsCleanPathAbsolute(name))
//	    throw new ArgumentException(FrameworkResources.InvalidTitleContainerName);
//	try { new Uri(name.Replace('\\', '/'), UriKind.Relative); }
//	catch (Exception inner) {
//	    throw new ArgumentException(FrameworkResources.InvalidTitleContainerName, inner);
//	}
//	try { return File.OpenRead(Path.Combine(TitleLocation.Path, name)); }
//	catch (Exception error) {
//	    if (error is FileNotFoundException || error is DirectoryNotFoundException ||
//	        error is ArgumentException)
//	        throw new FileNotFoundException(Format(OpenStreamNotFound, name));
//	    throw new IOException(Format(OpenStreamError, name), error);
//	}
//
// The FIRST guard's ArgumentNullException carries no message, unlike most of
// the profile's, and it fires for the EMPTY string as well as null -- so an
// empty name is an argument-null failure rather than a not-found one.
//
// Everything down to the open is pure managed string work, and all of it is
// reproduced here, character for character, because it decides which of three
// refusals a caller sees.
//
// # What CNA does with the rest, and the narrowing it brings
//
// `cna_title_container_read_ext` resolves the title path itself and hands back
// the WHOLE FILE. The canonical header states the trade: "This ABI has no
// stream handle for title content, and a title asset is read to use it, so the
// count/copy pair delivers the whole file instead. That is a deliberate
// narrowing: incremental reads over a title stream are not available."
//
// So the returned io.Reader is a reader over bytes already in memory. A
// consumer that reads it sequentially cannot tell; one that opened a very large
// asset expecting to stream it would pay for the whole file at once, and that
// is the narrowing rather than a bug. The alternative -- resolving the title
// path in Go and calling os.Open -- is not available: CNA exposes the title
// path only to itself here, and guessing a directory would be inventing the one
// thing this member exists to resolve.
func TitleContainerOpenStream(name string) (io.Reader, error) {
	// `string.IsNullOrEmpty`, and the ArgumentNullException it throws carries
	// no message.
	if name == "" {
		return nil, fmt.Errorf("%w: name", errTitleContainerArgumentNull)
	}
	clean := titleContainerCleanPath(name)
	if titleContainerCleanPathIsAbsolute(clean) {
		return nil, fmt.Errorf("%w: %s", errTitleContainerArgument, invalidTitleContainerName)
	}
	// The reference builds a RELATIVE System.Uri purely to see whether it
	// throws. Go has no equivalent parse with the same accept set, and the
	// characters a relative Uri rejects that the guard above does not are the
	// control characters -- so those are checked directly, which is the part of
	// the Uri constructor this call site actually depends on.
	if strings.ContainsFunc(clean, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return nil, fmt.Errorf("%w: %s", errTitleContainerArgument, invalidTitleContainerName)
	}
	runtime, ok := interop.CurrentRuntime()
	if !ok {
		return nil, errNoRunningGame
	}
	content, err := runtime.TitleContainerRead(clean)
	if err != nil {
		// The reference sorts the open's failure into two: the three "it is not
		// there" exception types become FileNotFoundException with
		// OpenStreamNotFound, and anything else becomes OpenStreamError. CNA
		// reports the first as CNA_RESULT_IO -- the header says it "reports
		// [the canonical failure for a missing file] as CNA_RESULT_IO rather
		// than letting it reach a caller as an internal failure" -- and gives
		// no way to tell a missing file from an unreadable one, so the two
		// messages cannot be told apart here.
		//
		// The not-found message is the one used, because CNA_RESULT_IO is
		// documented as "when the file cannot be opened", which is the case
		// OpenStreamNotFound names. The other message is retained and verified
		// so the pair is recorded rather than half-forgotten.
		return nil, fmt.Errorf("%w: %s", err, fmt.Sprintf(openStreamNotFound, clean))
	}
	return bytes.NewReader(content), nil
}

// The three Microsoft messages this member carries, read from
// Microsoft.Xna.Framework.dll. Two of them carry the requested name through
// String.Format's {0}.
const (
	invalidTitleContainerName = "Invalid filename. TitleContainer.OpenStream requires a relative URI."
	openStreamNotFound        = "Error loading \"%s\". File not found."
	openStreamError           = "Error loading \"%s\". Cannot open file."
)

// The two refusals the guards report. They are the projection's channel for the
// reference's ArgumentNullException and ArgumentException.
var (
	errTitleContainerArgumentNull = errors.New("argument is null")
	errTitleContainerArgument     = errors.New("argument is invalid")
)

// titleContainerCleanPath is TitleContainer::GetCleanPath, 256 bytes of string
// work transcribed operation by operation:
//
//	path = path.Replace('/', '\\');
//	path = path.Replace("\\.\\", "\\");
//	while (path.StartsWith(".\\"))  path = path.Substring(2);
//	while (path.EndsWith("\\."))
//	    path = path.Length > 2 ? path.Substring(0, path.Length - 2) : "\\";
//	for (int at = 1; at < path.Length; ) {
//	    at = path.IndexOf("\\..\\", at);
//	    if (at < 0) break;
//	    at = CollapseParentDirectory(ref path, at, 4);
//	}
//	if (path.EndsWith("\\..")) {
//	    int at = path.Length - 3;
//	    if (at > 0) CollapseParentDirectory(ref path, at, 3);
//	}
//	if (path == ".") path = string.Empty;
//
// Three details a reader would guess wrong. The forward-slash replacement comes
// FIRST, so a caller may write either separator. The `\.` trim collapses a path
// that is nothing but `\.` to `\` rather than to the empty string. And the
// trailing `\..` is collapsed only when there is something before it -- `\..`
// alone is left, which the absoluteness check then rejects.
func titleContainerCleanPath(path string) string {
	path = strings.ReplaceAll(path, "/", "\\")
	path = strings.ReplaceAll(path, "\\.\\", "\\")
	for strings.HasPrefix(path, ".\\") {
		path = path[len(".\\"):]
	}
	for strings.HasSuffix(path, "\\.") {
		if len(path) > len("\\.") {
			path = path[:len(path)-len("\\.")]
		} else {
			path = "\\"
		}
	}
	for at := 1; at < len(path); {
		found := strings.Index(path[at:], "\\..\\")
		if found < 0 {
			break
		}
		at, path = titleContainerCollapseParent(path, at+found, len("\\..\\"))
	}
	if strings.HasSuffix(path, "\\..") {
		if at := len(path) - len("\\.."); at > 0 {
			_, path = titleContainerCollapseParent(path, at, len("\\.."))
		}
	}
	if path == "." {
		path = ""
	}
	return path
}

// titleContainerCollapseParent is TitleContainer::CollapseParentDirectory,
// forty bytes over a `string&`:
//
//	int start = path.LastIndexOf('\\', position - 1) + 1;
//	path = path.Remove(start, position - start + removeLength);
//	return Math.Max(start - 1, 1);
//
// The reference mutates the string through a reference parameter and returns
// the position to resume from; Go returns both. The `Max(start - 1, 1)` is what
// keeps the caller's loop from rescanning position zero, and it is the reason
// the loop starts at 1 rather than 0.
func titleContainerCollapseParent(path string, position, removeLength int) (int, string) {
	start := strings.LastIndexByte(path[:position], '\\') + 1
	end := position + removeLength
	if end > len(path) {
		end = len(path)
	}
	path = path[:start] + path[end:]
	if resume := start - 1; resume > 1 {
		return resume, path
	}
	return 1, path
}

// titleContainerCleanPathIsAbsolute is TitleContainer::IsCleanPathAbsolute, 87
// bytes and six tests. Its NAME is misleading and its body is what matters: it
// answers true for anything that is not safely relative, which includes a
// filename character Windows forbids.
//
//	if (path.IndexOfAny(badCharacters) >= 0) return true;   // : * ? " < > |
//	if (path.StartsWith("\\"))   return true;
//	if (path.StartsWith("..\\")) return true;
//	if (path.Contains("\\..\\")) return true;
//	if (path.EndsWith("\\.."))   return true;
//	if (path == "..")            return true;
//	return false;
//
// The last four are what stops a caller escaping the title directory, and they
// run AFTER GetCleanPath has collapsed every `..` it can -- so what reaches
// them is a `..` that would have escaped.
func titleContainerCleanPathIsAbsolute(path string) bool {
	// badCharacters, read from the assembly's static blob: 3A 00 2A 00 3F 00
	// 22 00 3C 00 3E 00 7C 00 -- seven UTF-16 code units.
	if strings.ContainsAny(path, ":*?\"<>|") {
		return true
	}
	return strings.HasPrefix(path, "\\") ||
		strings.HasPrefix(path, "..\\") ||
		strings.Contains(path, "\\..\\") ||
		strings.HasSuffix(path, "\\..") ||
		path == ".."
}
