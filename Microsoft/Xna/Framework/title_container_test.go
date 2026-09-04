package framework

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TitleContainer's guards are pure managed string work -- 256 bytes of
// GetCleanPath, 87 of IsCleanPathAbsolute and 40 of CollapseParentDirectory --
// so all of it is measured here without a device. What needs one is the read
// itself, which is in the native-stress scenario.

// TestTitleContainerCleanPathIsTheReferenceAlgorithm walks every operation
// GetCleanPath performs, in the order it performs them.
func TestTitleContainerCleanPathIsTheReferenceAlgorithm(t *testing.T) {
	for _, probe := range []struct{ name, in, want string }{
		// The forward-slash replacement comes FIRST, so a caller may write
		// either separator and everything after sees backslashes.
		{"forward slashes become backslashes", "a/b/c", `a\b\c`},
		{"mixed separators", `a/b\c`, `a\b\c`},
		// `\.\` collapses to `\`, and it is a REPLACE rather than a loop -- so
		// overlapping runs are left behind by design.
		{"current directory in the middle", `a\.\b`, `a\b`},
		{"two current directories", `a\.\b\.\c`, `a\b\c`},
		// A leading `.\` is trimmed repeatedly.
		{"leading current directory", `.\a`, "a"},
		{"repeated leading current directory", `.\.\.\a`, "a"},
		// A trailing `\.` is trimmed repeatedly, and a path that is nothing but
		// `\.` becomes `\` rather than empty -- which the absoluteness check
		// then rejects.
		{"trailing current directory", `a\.`, "a"},
		{"nothing but a trailing current directory", `\.`, `\`},
		// `\..\` collapses the segment before it.
		{"parent in the middle", `a\..\b`, "b"},
		{"two parents", `a\b\..\..\c`, "c"},
		{"parent after a deeper path", `a\b\c\..\d`, `a\b\d`},
		// A trailing `\..` collapses only when something precedes it.
		// `a\b\..` becomes `a\` and NOT `a`, which is worth tracing because it
		// looks like a bug and is not. CollapseParentDirectory removes from the
		// separator AFTER the collapsed segment: start is
		// LastIndexOf('\\', position-1) + 1 = 2, and Remove(2, 3-2+3) takes
		// indices 2 through 5 out of `a\b\..`, leaving the leading separator.
		{"trailing parent", `a\b\..`, `a\`},
		{"trailing parent with one segment", `a\..`, ""},
		// A path that is exactly "." becomes empty, which is the last statement
		// of the body.
		{"a single dot", ".", ""},
		// And the cases the collapse deliberately does NOT reach, which are
		// what the absoluteness check exists for.
		//
		// The parent loop starts at index ONE, so a `\..\` at index zero is
		// never found and the path survives to be refused. Starting at zero
		// would collapse it and hand back `a`, which is inside the title
		// directory and is exactly the escape this member exists to stop.
		{"parent at index zero", `\..\a`, `\..\a`},
		// A path that is nothing but a trailing parent is left alone, because
		// the collapse is guarded by `at > 0`. Collapsing it would produce the
		// empty string, which the absoluteness check accepts.
		{"a bare trailing parent", `\..`, `\..`},
		// The resume value after a collapse is Math.Max(start - 1, 1) and not
		// zero, and a DOUBLE separator is what tells the two apart: resuming at
		// zero finds the `\..\` the collapse just moved to index zero and
		// eats the rest.
		{"double separator before a parent", `\\..\..\`, `\..\`},
		{"double separator between parents", `\..\\..\`, `\..\`},
		{"leading parent", `..\a`, `..\a`},
		{"a bare parent", "..", ".."},
		{"absolute", `\a\b`, `\a\b`},
		{"plain relative", `a\b`, `a\b`},
		{"empty", "", ""},
	} {
		t.Run(probe.name, func(t *testing.T) {
			if got := titleContainerCleanPath(probe.in); got != probe.want {
				t.Fatalf("GetCleanPath(%q) = %q, want %q", probe.in, got, probe.want)
			}
		})
	}
}

// TestTitleContainerAbsolutenessIsSixTests pins IsCleanPathAbsolute, whose name
// is misleading and whose body is six tests -- the first of which is the
// forbidden-character table.
func TestTitleContainerAbsolutenessIsSixTests(t *testing.T) {
	// The seven characters read from the assembly's static blob.
	for _, bad := range []string{":", "*", "?", "\"", "<", ">", "|"} {
		if !titleContainerCleanPathIsAbsolute("a" + bad + "b") {
			t.Fatalf("a path containing %q was accepted", bad)
		}
	}
	// A backslash and a forward slash are NOT in that table: they are
	// separators, and the tests below are what govern them.
	if titleContainerCleanPathIsAbsolute(`a\b`) {
		t.Fatal("an ordinary relative path was rejected")
	}
	for _, absolute := range []string{`\a`, `..\a`, `a\..\b`, `a\..`, ".."} {
		if !titleContainerCleanPathIsAbsolute(absolute) {
			t.Fatalf("%q was accepted", absolute)
		}
	}
	for _, relative := range []string{"a", `a\b\c`, "", ".", `a..b`, `a\..b`} {
		if titleContainerCleanPathIsAbsolute(relative) {
			t.Fatalf("%q was rejected", relative)
		}
	}
}

// TestTitleContainerEscapeAttemptsAreRefused is the two helpers working
// together, which is the pair's whole purpose: GetCleanPath collapses every
// `..` it can, and what survives is a `..` that would have escaped the title
// directory.
func TestTitleContainerEscapeAttemptsAreRefused(t *testing.T) {
	for _, escape := range []string{
		`..\secret`,
		`\..\secret`,
		`\..`,
		`\\..\..\`,
		`a\..\..\secret`,
		"../secret",
		`a\b\..\..\..\secret`,
		`\etc\passwd`,
		"/etc/passwd",
		"..",
	} {
		clean := titleContainerCleanPath(escape)
		if !titleContainerCleanPathIsAbsolute(clean) {
			t.Fatalf("%q cleaned to %q and was accepted", escape, clean)
		}
		if _, err := TitleContainerOpenStream(escape); !errors.Is(err, errTitleContainerArgument) {
			t.Fatalf("OpenStream(%q) = %v, want the invalid-name refusal", escape, err)
		}
	}
	// And a path that merely LOOKS like an escape but collapses to something
	// inside the directory is accepted by the guards -- it fails later, on the
	// read, because no game is running.
	for _, inside := range []string{`a\..\b`, `.\a`, "a/b"} {
		clean := titleContainerCleanPath(inside)
		if titleContainerCleanPathIsAbsolute(clean) {
			t.Fatalf("%q cleaned to %q and was refused", inside, clean)
		}
		if _, err := TitleContainerOpenStream(inside); !errors.Is(err, errNoRunningGame) {
			t.Fatalf("OpenStream(%q) = %v, want the guards to pass and the read to refuse", inside, err)
		}
	}
}

// TestTitleContainerRefusesAnEmptyNameAsArgumentNull pins the first guard,
// which is `string.IsNullOrEmpty` and throws ArgumentNullException with NO
// message -- so an empty name is an argument-null failure and not a not-found
// one.
func TestTitleContainerRefusesAnEmptyNameAsArgumentNull(t *testing.T) {
	reader, err := TitleContainerOpenStream("")
	if !errors.Is(err, errTitleContainerArgumentNull) {
		t.Fatalf("OpenStream(\"\") = %v, want the argument-null refusal", err)
	}
	if reader != nil {
		t.Fatal("a refused OpenStream handed back a reader")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("the refusal does not name the parameter: %v", err)
	}
	// It is NOT the invalid-name refusal, which is a different exception type
	// in the reference and carries a message this one does not.
	if strings.Contains(err.Error(), invalidTitleContainerName) {
		t.Fatal("the empty-name refusal carried InvalidTitleContainerName")
	}
}

// TestTitleContainerRefusesControlCharacters covers the Uri guard's one
// contribution beyond the character table.
func TestTitleContainerRefusesControlCharacters(t *testing.T) {
	for _, name := range []string{"a\x00b", "a\nb", "a\tb", "a\x7fb"} {
		if _, err := TitleContainerOpenStream(name); !errors.Is(err, errTitleContainerArgument) {
			t.Fatalf("OpenStream(%q) = %v, want the invalid-name refusal", name, err)
		}
	}
}

// TestTheRootStaticsRefuseWithoutAGame pins the one divergence both root types
// share: the reference's statics need nothing and CNA's routes take a game
// handle for thread affinity.
func TestTheRootStaticsRefuseWithoutAGame(t *testing.T) {
	if err := FrameworkDispatcherUpdate(); !errors.Is(err, errNoRunningGame) {
		t.Fatalf("FrameworkDispatcherUpdate outside a game = %v", err)
	}
	// A name that passes every guard still refuses, which is what places the
	// failure at the READ rather than at a guard.
	if _, err := TitleContainerOpenStream("content/asset.xnb"); !errors.Is(err, errNoRunningGame) {
		t.Fatalf("OpenStream outside a game = %v", err)
	}
}

// TestTitleContainerOpenStreamAnswersAReader is the shape claim: the contract
// says Stream, the settled BCL mapping says io.Reader, and a refusal must hand
// back a NIL one rather than a typed nil.
func TestTitleContainerOpenStreamAnswersAReader(t *testing.T) {
	var _ func(string) (io.Reader, error) = TitleContainerOpenStream
	reader, err := TitleContainerOpenStream("missing")
	if err == nil {
		t.Fatal("OpenStream succeeded with no game")
	}
	if reader != nil {
		t.Fatal("a refused OpenStream handed back a non-nil reader; a caller checking the reader would dereference it")
	}
}
