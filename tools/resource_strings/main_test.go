package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func retainedAssemblies(t *testing.T) string {
	t.Helper()
	if explicit := os.Getenv("XNA_ASSEMBLIES"); explicit != "" {
		return explicit
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory; set XNA_ASSEMBLIES")
	}
	root := filepath.Join(home, "deps", "xna40-windows-assemblies")
	if _, err := os.Stat(filepath.Join(root, "Microsoft.Xna.Framework.Game.dll")); err != nil {
		t.Skip("the retained XNA 4.0 assemblies are not available; set XNA_ASSEMBLIES")
	}
	return root
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	t.Fatal("could not locate the module root")
	return ""
}

// TestEveryClaimedMessageIsInARetainedAssembly is the control.
func TestEveryClaimedMessageIsInARetainedAssembly(t *testing.T) {
	result, err := run(retainedAssemblies(t), repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings: %v", result.Findings)
	}
	if result.Verified != result.Claimed || result.Claimed == 0 {
		t.Fatalf("verified %d of %d claimed messages", result.Verified, result.Claimed)
	}
	if result.Scanned == 0 {
		t.Fatal("the source scan found no message constants, so it is measuring nothing")
	}
}

// TestAnInventedMessageIsRejected is the falsifiability proof, and it plants
// exactly the defect that made this tool necessary: a sentence inferred from a
// resource KEY rather than read from the assembly. It is plausible, it is
// grammatical, and it is not what Microsoft wrote.
func TestAnInventedMessageIsRejected(t *testing.T) {
	assemblies := retainedAssemblies(t)
	saved := registry
	t.Cleanup(func() { registry = saved })
	registry = append(append([]claimedString(nil), saved...), claimedString{
		Key:      "BackBufferDimMustBePositive",
		Assembly: "Microsoft.Xna.Framework.Game.dll",
		Value:    "The back buffer dimension must be positive.",
	})
	result, err := run(assemblies, repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("an invented message passed; the tool is not checking anything")
	}
	if !strings.Contains(strings.Join(result.Findings, "\n"), "is not in Microsoft.Xna.Framework.Game.dll") {
		t.Fatalf("findings did not name the missing text: %v", result.Findings)
	}
}

// TestAMessageInTheWrongAssemblyIsRejected proves the check is per assembly
// rather than "somewhere in the pinned set": a Game message is not a
// Framework message, and a binding that looked in the wrong one would be
// verifying a coincidence.
func TestAMessageInTheWrongAssemblyIsRejected(t *testing.T) {
	assemblies := retainedAssemblies(t)
	saved := registry
	t.Cleanup(func() { registry = saved })
	registry = []claimedString{{
		Key:      "MissingGraphicsDeviceService",
		Assembly: "Microsoft.Xna.Framework.Graphics.dll",
		Value:    "Drawable components require a graphics device service in the game service container.",
	}}
	result, err := run(assemblies, repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range result.Findings {
		if strings.Contains(finding, "Microsoft.Xna.Framework.Graphics.dll") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a Game.dll message was accepted as a Graphics.dll one: %v", result.Findings)
	}
}

// TestThePlaceholderSubstitutionIsRestored pins the one deliberate difference
// between what CNA-Go spells and what the assembly holds: the CLR formats with
// {0}/{1} and Go with %s, so the comparison restores the CLR spelling. Without
// that, the one format message in the profile would look like a defect.
func TestThePlaceholderSubstitutionIsRestored(t *testing.T) {
	entry := claimedString{
		Value:        "Service provider object of type %s must be assignable to service type %s.",
		Placeholders: true,
	}
	want := "Service provider object of type {0} must be assignable to service type {1}."
	if got := clrSpelling(entry); got != want {
		t.Fatalf("clrSpelling = %q, want %q", got, want)
	}
	// A message without the flag is compared exactly as written.
	plain := claimedString{Value: "Game cannot be null."}
	if got := clrSpelling(plain); got != plain.Value {
		t.Fatalf("clrSpelling rewrote a plain message: %q", got)
	}
}
