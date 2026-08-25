// Command capabilities validates the machine-readable runtime inventory and
// renders its Markdown counterpart deterministically.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type inventory struct {
	SchemaVersion     int          `json:"schema_version"`
	Profile           string       `json:"profile"`
	QualifiedPlatform string       `json:"qualified_platform"`
	Capabilities      []capability `json:"capabilities"`
}

type capability struct {
	ID         string `json:"id"`
	Capability string `json:"capability"`
	Status     string `json:"status"`
	Evidence   string `json:"evidence"`
	Notes      string `json:"notes"`
}

var admittedStatuses = map[string]bool{
	"VERIFIED_MANAGED": true, "VERIFIED_NATIVE": true,
	"UPSTREAM_CNA_BLOCKED": true, "BACKEND_BLOCKED": true,
	"HARDWARE_PENDING": true, "PLATFORM_PENDING": true,
	"ASSET_PENDING": true, "LANGUAGE_MAPPING_LIMITATION": true,
	"UNIMPLEMENTED_CNA_GO": true,
}

var requiredIDs = []string{
	"xna-package-mapping", "pure-value-foundation", "game-lifecycle",
	"callback-containment", "owner-os-thread", "game-recreation",
	"graphics-device", "viewport", "clear", "texture-stream", "spritebatch",
	"keyboard", "visible-rendering", "content-xnb", "effects-3d", "audio",
	"media", "storage", "linux-amd64", "windows", "macos", "android",
	"web-wasm", "buffer-usage", "surface-format", "depth-format",
	"graphics-profile", "button-state",
}

func main() {
	source := flag.String("source", "docs/runtime-capabilities.json", "inventory JSON")
	output := flag.String("output", "docs/generated/runtime-capabilities.md", "generated Markdown")
	check := flag.Bool("check", false, "fail if generated output is stale")
	flag.Parse()
	if err := run(*source, *output, *check); err != nil {
		fmt.Fprintln(os.Stderr, "capabilities:", err)
		os.Exit(1)
	}
}

func run(source, output string, check bool) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	var value inventory
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if err := validate(value); err != nil {
		return err
	}
	rendered := render(value)
	if check {
		current, err := os.ReadFile(output)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, rendered) {
			return errors.New("generated runtime-capabilities.md is stale")
		}
		fmt.Printf("RUNTIME_CAPABILITIES=%d STATUS=PASS\n", len(value.Capabilities))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(output, rendered, 0o644); err != nil {
		return err
	}
	fmt.Printf("RUNTIME_CAPABILITIES=%d GENERATED=%s\n", len(value.Capabilities), output)
	return nil
}

func validate(value inventory) error {
	if value.SchemaVersion != 1 || value.Profile == "" || value.QualifiedPlatform == "" {
		return errors.New("schema_version=1, profile, and qualified_platform are required")
	}
	seen := make(map[string]bool)
	for index, row := range value.Capabilities {
		if row.ID == "" || row.Capability == "" || row.Evidence == "" || row.Notes == "" {
			return fmt.Errorf("capability row %d has an empty required field", index)
		}
		if seen[row.ID] {
			return fmt.Errorf("duplicate capability id %q", row.ID)
		}
		seen[row.ID] = true
		if !admittedStatuses[row.Status] {
			return fmt.Errorf("capability %q has invalid status %q", row.ID, row.Status)
		}
	}
	for _, id := range requiredIDs {
		if !seen[id] {
			return fmt.Errorf("required capability %q is absent", id)
		}
	}
	return nil
}

func render(value inventory) []byte {
	rows := append([]capability(nil), value.Capabilities...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Status != rows[j].Status {
			return rows[i].Status < rows[j].Status
		}
		return rows[i].ID < rows[j].ID
	})
	var output bytes.Buffer
	fmt.Fprintln(&output, "# Generated runtime capability inventory")
	fmt.Fprintln(&output)
	fmt.Fprintln(&output, "Generated from `docs/runtime-capabilities.json`; do not edit by hand.")
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "- Profile: %s\n", value.Profile)
	fmt.Fprintf(&output, "- Qualified platform: %s\n", value.QualifiedPlatform)
	fmt.Fprintf(&output, "- Capability rows: %d\n\n", len(rows))
	fmt.Fprintln(&output, "| Status | Capability | Evidence | Notes |")
	fmt.Fprintln(&output, "|---|---|---|---|")
	for _, row := range rows {
		fmt.Fprintf(&output, "| `%s` | %s | %s | %s |\n", row.Status, row.Capability, row.Evidence, row.Notes)
	}
	return output.Bytes()
}
