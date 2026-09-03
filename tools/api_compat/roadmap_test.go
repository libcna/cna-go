package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Foundation 74 — the ROADMAP staleness guard.
//
// ROADMAP.md is the file a reader opens to find out what is left, and until
// this milestone every number in it was typed by hand. A hand-typed count is
// wrong the moment the next milestone lands, and a wrong count in the file
// whose whole purpose is to be trusted is worse than no count at all.
//
// The guard is deliberately narrow. It checks the ONE fenced block ROADMAP.md
// marks as its scoreboard, and it checks it against the generated reports.
// Historical evidence documents -- every docs/foundation-NN-*.md -- quote the
// scoreboard of the milestone they record, and those quotes are correct FOR
// THAT MILESTONE. Treating them as stale would be a category error, so nothing
// here looks at them.
// ---------------------------------------------------------------------------

const (
	roadmapPath           = "../../ROADMAP.md"
	roadmapScoreboardOpen = "<!-- cna-go:scoreboard -->"
	roadmapScoreboardEnd  = "<!-- /cna-go:scoreboard -->"
)

// roadmapSources says which generated report each guarded key comes from. A
// key that is not here cannot appear in the block, which is what stops the
// scoreboard from growing a number nobody generates.
var roadmapSources = map[string]string{
	"TOTAL_DIAGNOSTICS":          "api",
	"MISSING_TYPE":               "api",
	"MISSING_MEMBER":             "api",
	"COMPLETE_TYPES":             "api",
	"PARTIAL_TYPES":              "api",
	"UNEXPECTED_MEMBER":          "api",
	"ALLOWLIST_ENTRIES":          "api",
	"GLOBAL_ACTIONABLE_LOCAL":    "api",
	"GLOBAL_UNREVIEWED":          "api",
	"BOUND_FUNCTIONS":            "abi",
	"MANIFEST_LAYOUT_AGREEMENTS": "abi",
	"ABI_MISMATCHES":             "abi",
}

// TestRoadmapScoreboardMatchesTheGeneratedReports is the guard itself.
func TestRoadmapScoreboardMatchesTheGeneratedReports(t *testing.T) {
	document, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	quoted, err := parseRoadmapScoreboard(string(document))
	if err != nil {
		t.Fatal(err)
	}
	generated := generatedScoreboard(t)
	for key, want := range quoted {
		source, guarded := roadmapSources[key]
		if !guarded {
			t.Fatalf("ROADMAP.md's scoreboard quotes %q, which no generated report produces", key)
		}
		got, present := generated[key]
		if !present {
			t.Fatalf("ROADMAP.md quotes %q from the %s report, which does not carry it", key, source)
		}
		if got != want {
			t.Fatalf("ROADMAP.md says %s=%d, the generated report says %d; regenerate the reports and copy their values",
				key, want, got)
		}
	}
	for key := range roadmapSources {
		if _, present := quoted[key]; !present {
			t.Fatalf("ROADMAP.md's scoreboard does not quote %q, which the guard covers", key)
		}
	}
}

// TestRoadmapStalenessGuardRejectsAStaleNumber is the guard's own falsifiability
// proof: a planted stale value must be caught. Without it the guard could pass
// by looking at nothing.
func TestRoadmapStalenessGuardRejectsAStaleNumber(t *testing.T) {
	document, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	generated := generatedScoreboard(t)

	// The live block must agree, or the mutation below proves nothing.
	live, err := parseRoadmapScoreboard(string(document))
	if err != nil {
		t.Fatal(err)
	}
	if disagreement := scoreboardDisagreement(live, generated); disagreement != "" {
		t.Fatalf("the unmutated scoreboard already disagrees: %s", disagreement)
	}

	mutations := map[string]func(string) string{
		"stale MISSING_TYPE": func(s string) string {
			return strings.Replace(s, "MISSING_TYPE                    98", "MISSING_TYPE                    97", 1)
		},
		"stale BOUND_FUNCTIONS": func(s string) string {
			return strings.Replace(s, "BOUND_FUNCTIONS                230", "BOUND_FUNCTIONS                229", 1)
		},
		"stale GLOBAL_UNREVIEWED": func(s string) string {
			return strings.Replace(s, "GLOBAL_UNREVIEWED                0", "GLOBAL_UNREVIEWED                4", 1)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			mutated := mutate(string(document))
			if mutated == string(document) {
				t.Fatal("the mutation did not change the document, so it tests nothing")
			}
			quoted, err := parseRoadmapScoreboard(mutated)
			if err != nil {
				t.Fatal(err)
			}
			if scoreboardDisagreement(quoted, generated) == "" {
				t.Fatal("the staleness guard accepted a planted stale number")
			}
		})
	}

	// A key nobody generates must be refused too, so the block cannot grow a
	// number that is unfalsifiable by construction.
	t.Run("ungenerated key", func(t *testing.T) {
		mutated := strings.Replace(string(document),
			"ABI_MISMATCHES                   0",
			"ABI_MISMATCHES                   0\nINVENTED_COUNTER                42", 1)
		quoted, err := parseRoadmapScoreboard(mutated)
		if err != nil {
			t.Fatal(err)
		}
		if _, guarded := roadmapSources["INVENTED_COUNTER"]; guarded {
			t.Fatal("INVENTED_COUNTER is somehow a guarded key")
		}
		if _, quotedIt := quoted["INVENTED_COUNTER"]; !quotedIt {
			t.Fatal("the parser dropped the invented key instead of surfacing it")
		}
	})
}

func scoreboardDisagreement(quoted, generated map[string]int) string {
	for key, want := range quoted {
		if _, guarded := roadmapSources[key]; !guarded {
			return fmt.Sprintf("%s is not a generated key", key)
		}
		if got, present := generated[key]; !present || got != want {
			return fmt.Sprintf("%s quoted %d, generated %d (present=%v)", key, want, got, present)
		}
	}
	return ""
}

// parseRoadmapScoreboard reads the marked block's `KEY VALUE` lines.
func parseRoadmapScoreboard(document string) (map[string]int, error) {
	start := strings.Index(document, roadmapScoreboardOpen)
	if start < 0 {
		return nil, fmt.Errorf("ROADMAP.md carries no %s marker", roadmapScoreboardOpen)
	}
	end := strings.Index(document, roadmapScoreboardEnd)
	if end < start {
		return nil, fmt.Errorf("ROADMAP.md carries no %s marker after the opening one", roadmapScoreboardEnd)
	}
	values := make(map[string]int)
	for _, line := range strings.Split(document[start+len(roadmapScoreboardOpen):end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("scoreboard line %q is not `KEY VALUE`", line)
		}
		value, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("scoreboard line %q has a non-numeric value", line)
		}
		if _, duplicate := values[fields[0]]; duplicate {
			return nil, fmt.Errorf("scoreboard quotes %q twice", fields[0])
		}
		values[fields[0]] = value
	}
	return values, nil
}

// generatedScoreboard merges the two generated reports the guard reads.
func generatedScoreboard(t *testing.T) map[string]int {
	t.Helper()
	merged := make(map[string]int)

	apiBytes, err := os.ReadFile("../../docs/generated/api-compat-report.json")
	if err != nil {
		t.Fatal(err)
	}
	var api struct {
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal(apiBytes, &api); err != nil {
		t.Fatal(err)
	}
	for key, value := range api.Summary {
		if roadmapSources[key] == "api" {
			merged[key] = value
		}
	}

	abiBytes, err := os.ReadFile("../../docs/generated/native-abi-report.json")
	if err != nil {
		t.Fatal(err)
	}
	var abi map[string]json.RawMessage
	if err := json.Unmarshal(abiBytes, &abi); err != nil {
		t.Fatal(err)
	}
	for key, source := range roadmapSources {
		if source != "abi" {
			continue
		}
		raw, present := abi[key]
		if !present {
			t.Fatalf("the native ABI report carries no %q", key)
		}
		// ABI_MISMATCHES is an ARRAY in the native report -- the findings
		// themselves, not a tally -- so what the scoreboard quotes is its
		// length. Everything else is a plain number. Reading both shapes here
		// keeps the guard honest about what the report actually contains
		// rather than requiring the report to change for the guard's benefit.
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			var findings []json.RawMessage
			if listErr := json.Unmarshal(raw, &findings); listErr != nil {
				t.Fatalf("native ABI report %q is neither a number nor a list: %v / %v", key, err, listErr)
			}
			value = len(findings)
		}
		merged[key] = value
	}
	return merged
}
