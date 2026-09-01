package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stageIdentityPackage copies the framework package sources into a temporary
// directory, applying one mutation, and returns the directory.
//
// The identity gate is a claim about METHOD BODIES, so a synthetic
// declaration-level fixture cannot falsify it. Real sources are staged instead,
// which is the same shape the native ABI mutation controls take: a mutation
// that still passes is a hole in the evidence rather than a passing test.
func stageIdentityPackage(t *testing.T, file, old, replacement string) string {
	t.Helper()
	source := filepath.Join("..", "..", "Microsoft", "Xna", "Framework")
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	staged := t.TempDir()
	mutated := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(source, entry.Name()))
		if readErr != nil {
			t.Fatal(readErr)
		}
		text := string(content)
		if entry.Name() == file {
			if !strings.Contains(text, old) {
				t.Fatalf("mutation target %q is no longer in %s; the evidence it carries is stale", old, file)
			}
			text = strings.Replace(text, old, replacement, 1)
			mutated = true
		}
		if writeErr := os.WriteFile(filepath.Join(staged, entry.Name()), []byte(text), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if file != "" && !mutated {
		t.Fatalf("mutation names %s, which the package does not contain", file)
	}
	return staged
}

func identityMeasurement(t *testing.T, staged string) (report, []xnaCompositionIdentityMeasurement) {
	t.Helper()
	expected, actual := loadPinnedSurfaces(t)
	if staged != "" {
		actual.PackageDirs[modulePath+"/Microsoft/Xna/Framework"] = staged
	}
	result := report{Summary: map[string]int{}}
	measurements := measureXNACompositionIdentity(&result, expected, actual)
	return result, measurements
}

// TestXNACompositionIdentityIsMeasuredOnTheRealSources is the control. It has
// to pass unmutated, or every mutation below would "fail" for the wrong reason.
func TestXNACompositionIdentityIsMeasuredOnTheRealSources(t *testing.T) {
	result, measurements := identityMeasurement(t, "")
	if len(measurements) != 1 || measurements[0].Verdict != "PASS" {
		t.Fatalf("identity measurement = %+v", measurements)
	}
	if got := result.Summary["XNA_COMPOSED_IDENTITY_SITES"]; got != 5 {
		t.Fatalf("%d identity sites, want the five GameComponent has", got)
	}
	if got := result.Summary["XNA_COMPOSED_IDENTITY_USES"]; got != 6 {
		t.Fatalf("%d identity uses, want six: Dispose(bool) has two and the other four have one each", got)
	}
	if got := result.Summary["XNA_COMPOSED_IDENTITY_BINDINGS"]; got != 1 {
		t.Fatalf("%d identity bindings, want the one projected derived type", got)
	}
	for _, category := range []string{"BASE_MAPPING_MISMATCH"} {
		if result.Summary[category] != 0 {
			t.Fatalf("%s = %d on unmutated sources", category, result.Summary[category])
		}
	}
}

// TestXNACompositionIdentityMutationsAreRejected plants one defect at a time.
// Each is a real way to lose CLR object identity under composition, and each
// compiles: the bare receiver IS a valid GameComponent everywhere it appears.
func TestXNACompositionIdentityMutationsAreRejected(t *testing.T) {
	for name, mutation := range map[string]struct{ file, old, replacement string }{
		// The two Dispose(bool) sites, separately. Together they are the reason
		// the registry counts USES rather than asking whether the member
		// reaches self at all: either one alone leaves the other correct.
		"disposed_event_announces_the_base_half": {
			"game_component.go",
			"c.disposed.Raise(c.self(), EventArgsEmpty())",
			"c.disposed.Raise(c, EventArgsEmpty())",
		},
		"components_remove_names_the_base_half": {
			"game_component.go",
			"c.game.Components().Remove(c.self())",
			"c.game.Components().Remove(c)",
		},
		"enabled_changed_announces_the_base_half": {
			"game_component.go",
			"return c.enabledChanged.Raise(c.self(), args)",
			"return c.enabledChanged.Raise(c, args)",
		},
		"update_order_changed_announces_the_base_half": {
			"game_component.go",
			"return c.updateOrderChanged.Raise(c.self(), args)",
			"return c.updateOrderChanged.Raise(c, args)",
		},
		"set_enabled_passes_the_base_half_as_sender": {
			"game_component.go",
			"return c.OnEnabledChanged(c.self(), EventArgsEmpty())",
			"return c.OnEnabledChanged(c, EventArgsEmpty())",
		},
		"set_update_order_passes_the_base_half_as_sender": {
			"game_component.go",
			"return c.OnUpdateOrderChanged(c.self(), EventArgsEmpty())",
			"return c.OnUpdateOrderChanged(c, EventArgsEmpty())",
		},
		// The derived object never installs itself, so every site above reaches
		// self and self answers with the base half anyway.
		"derived_constructor_does_not_install_the_clr_this": {
			"drawable_game_component.go",
			"component.component.bindDerived(component)",
			"_ = component",
		},
		// The accessor itself is neutered. It still compiles and still returns
		// an IGameComponent; it just never returns the derived object.
		"self_ignores_the_installed_derived_object": {
			"game_component.go",
			"\tif c.derived != nil {\n\t\treturn c.derived\n\t}\n\treturn c",
			"\treturn c",
		},
	} {
		t.Run(name, func(t *testing.T) {
			staged := stageIdentityPackage(t, mutation.file, mutation.old, mutation.replacement)
			result, measurements := identityMeasurement(t, staged)
			if result.Summary["BASE_MAPPING_MISMATCH"] == 0 {
				t.Fatalf("mutation %q produced no BASE_MAPPING_MISMATCH; the identity rule it breaks is not enforced", name)
			}
			if len(measurements) != 1 || measurements[0].Verdict != "FAIL" {
				t.Fatalf("mutation %q left the measurement %+v", name, measurements)
			}
		})
	}
}

// TestSelfIgnoringMutationIsNotCaughtStructurally records the one defect the
// structural gate CANNOT see, so the split between the two kinds of evidence is
// a stated fact rather than an assumption.
//
// A self() that ignores the installed derived object still compiles, is still
// called the recorded number of times at every site, and still comes from a
// constructor that binds. Only behaviour distinguishes it, which is what
// TestDrawableInheritedEventsAnnounceTheDerivedObject and
// TestDrawableDisposeRemovesTheDerivedObjectFromComponents are for.
func TestSelfIgnoringMutationIsNotCaughtStructurally(t *testing.T) {
	staged := stageIdentityPackage(t, "game_component.go",
		"\tif c.derived != nil {\n\t\treturn c.derived\n\t}\n\treturn c",
		"\tif c.derived != nil && false {\n\t\treturn c.derived\n\t}\n\treturn c")
	result, measurements := identityMeasurement(t, staged)
	if result.Summary["BASE_MAPPING_MISMATCH"] != 0 {
		t.Fatalf("the structural gate rejected a body it cannot actually distinguish: %+v", measurements)
	}
	if got := result.Summary["XNA_COMPOSED_IDENTITY_USES"]; got != 6 {
		t.Fatalf("identity uses = %d", got)
	}
}
