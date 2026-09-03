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

// pinnedBCL is the .NET Framework 4.0 mscorlib the retained XNA assemblies bind
// against, admitted by the sha256 every "read from the pinned mscorlib" claim in
// this repository already names.
func pinnedBCL(t *testing.T) string {
	t.Helper()
	if explicit := os.Getenv("PINNED_BCL"); explicit != "" {
		return explicit
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, "deps", "bcl-4.0-pinned")
	if _, err := os.Stat(filepath.Join(root, "mscorlib.dll")); err != nil {
		t.Skip("the pinned .NET Framework 4.0 mscorlib is not available; set PINNED_BCL")
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
	result, err := run(retainedAssemblies(t), pinnedBCL(t), repositoryRoot(t))
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
	result, err := run(assemblies, pinnedBCL(t), repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("an invented message passed; the tool is not checking anything")
	}
	joined := strings.Join(result.Findings, "\n")
	if !strings.Contains(joined, "BackBufferDimMustBePositive") || !strings.Contains(joined, "greater than zero") {
		t.Fatalf("findings did not report the key's real value: %v", result.Findings)
	}
}

// TestARealMessageUnderAnInventedKeyIsRejected is the falsifiability proof for
// what Foundation 50 added, and it plants the defect the substring search could
// not see: a sentence that IS Microsoft's, filed under a key that is not.
//
// This is not hypothetical. CNA-Go carried it from Foundation 44 to
// Foundation 50 under DopplerScaleMustBeGreaterThanOrEqualToZero, and every
// substring search passed, because the sentence really is in the assembly. The
// key is InvalidEmitterDopplerScale, and the key is what names the throw site.
func TestARealMessageUnderAnInventedKeyIsRejected(t *testing.T) {
	assemblies := retainedAssemblies(t)
	saved := registry
	t.Cleanup(func() { registry = saved })
	registry = []claimedString{{
		Key:      "DopplerScaleMustBeGreaterThanOrEqualToZero",
		Assembly: "Microsoft.Xna.Framework.dll",
		Value:    "The doppler scale of an audio emitter must be greater than or equal to zero.",
	}}
	result, err := run(assemblies, pinnedBCL(t), repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("a real message under an invented key passed; the reader is not keyed by name")
	}
	if !strings.Contains(strings.Join(result.Findings, "\n"), "is not a resource key") {
		t.Fatalf("findings did not name the missing key: %v", result.Findings)
	}
}

// TestTheResourceReaderFindsTheKeysTheThrowSitesCall reads four keys whose
// values this milestone depends on and requires each to be exactly what the
// assembly holds. It is the control for resources.go itself: a reader that
// returned an empty map would make every registry check vacuous, and the
// registry check alone cannot tell "verified" from "found nothing to compare".
func TestTheResourceReaderFindsTheKeysTheThrowSitesCall(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join(retainedAssemblies(t), "Microsoft.Xna.Framework.dll"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := resourceStrings(blob)
	if err != nil {
		t.Fatal(err)
	}
	if len(set) < 100 {
		t.Fatalf("the reader found %d strings in Microsoft.Xna.Framework.dll, which is too few to be the real set", len(set))
	}
	for key, want := range map[string]string{
		"NullNotAllowed":              "This method does not accept null for this parameter.",
		"BeginMustBeCalledBeforeDraw": "Begin must be called successfully before a Draw can be called.",
		"BeginMustBeCalledBeforeEnd":  "Begin must be called successfully before End can be called.",
		"EndMustBeCalledBeforeBegin":  "Begin cannot be called again until End has been successfully called.",
	} {
		if got := set[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	// The two keys that read alike are DIFFERENT strings, and a reader that
	// confused them would make one of the two throw sites report the other's
	// message. Both name Begin and End; only one of them is about calling Begin
	// twice.
	if set["BeginMustBeCalledBeforeEnd"] == set["EndMustBeCalledBeforeBegin"] {
		t.Fatal("the two begin/end messages came back identical")
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
	result, err := run(assemblies, pinnedBCL(t), repositoryRoot(t))
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
