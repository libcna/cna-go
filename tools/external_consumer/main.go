// Command external_consumer materialises the event conformance canary as its
// own Go module and runs it against an extracted CNA-Go source artifact.
//
// The canary must prove what a real downstream user can do, so it must not be
// able to reach the development checkout. This command therefore:
//
//   - copies the canary sources out of testdata into a fresh module directory;
//   - writes a go.mod whose only requirement is github.com/openeggbert/cna-go,
//     replaced by the -source directory;
//   - runs `go test` with GOWORK=off and GOFLAGS=-mod=mod so no workspace file
//     and no sibling checkout can satisfy the import.
//
// Point -source at an extracted deterministic artifact for a real isolated run.
// Pointing it at the development tree is a convenience for iterating and is
// reported as such.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const modulePath = "github.com/openeggbert/cna-go"

type report struct {
	SchemaVersion int      `json:"schemaVersion"`
	Source        string   `json:"source"`
	Isolation     string   `json:"isolation"`
	Module        string   `json:"module"`
	Packages      []string `json:"packages"`
	Tests         int      `json:"TESTS"`
	Failures      int      `json:"FAILURES"`
	Status        string   `json:"status"`
}

func main() {
	source := flag.String("source", "", "CNA-Go source root the canary must build against (required)")
	work := flag.String("work", "build-consumer/eventcanary", "directory to materialise the isolated module in")
	fixture := flag.String("fixture", "tools/external_consumer/testdata/eventcanary", "canary sources")
	output := flag.String("output", "", "optional JSON report path")
	flag.Parse()

	if *source == "" {
		fail("-source is required; point it at an extracted CNA-Go source artifact")
	}
	result, err := run(*source, *work, *fixture)
	if err != nil {
		if result.Status == "" {
			result.Status = "FAIL"
		}
		fmt.Fprintln(os.Stderr, "external-consumer:", err)
	}
	if *output != "" {
		data, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			fail(marshalErr.Error())
		}
		data = append(data, '\n')
		if writeErr := os.WriteFile(*output, data, 0o644); writeErr != nil {
			fail(writeErr.Error())
		}
	}
	fmt.Printf("EXTERNAL_CONSUMER_SOURCE=%s\n", result.Source)
	fmt.Printf("EXTERNAL_CONSUMER_ISOLATION=%s\n", result.Isolation)
	fmt.Printf("EXTERNAL_CONSUMER_TESTS=%d\n", result.Tests)
	fmt.Printf("EXTERNAL_CONSUMER_FAILURES=%d\n", result.Failures)
	fmt.Printf("EXTERNAL_CONSUMER_STATUS=%s\n", result.Status)
	if result.Status != "PASS" {
		os.Exit(1)
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "external-consumer:", message)
	os.Exit(1)
}

func run(source, work, fixture string) (report, error) {
	absoluteSource, err := filepath.Abs(source)
	if err != nil {
		return report{}, err
	}
	result := report{
		SchemaVersion: 1,
		Source:        absoluteSource,
		Module:        "cna-go-event-canary",
		Isolation:     "GOWORK=off; go.mod replace only; no sibling checkout",
		Status:        "FAIL",
	}
	if _, err := os.Stat(filepath.Join(absoluteSource, "go.mod")); err != nil {
		return result, fmt.Errorf("source %q has no go.mod", absoluteSource)
	}

	if err := os.RemoveAll(work); err != nil {
		return result, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return result, err
	}
	entries, err := os.ReadDir(fixture)
	if err != nil {
		return result, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if err != nil {
			return result, err
		}
		if err := os.WriteFile(filepath.Join(work, entry.Name()), data, 0o644); err != nil {
			return result, err
		}
		result.Packages = append(result.Packages, entry.Name())
	}
	if len(result.Packages) == 0 {
		return result, fmt.Errorf("no canary sources in %q", fixture)
	}

	goMod := fmt.Sprintf("module %s\n\ngo 1.22\n\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		result.Module, modulePath, modulePath, absoluteSource)
	if err := os.WriteFile(filepath.Join(work, "go.mod"), []byte(goMod), 0o644); err != nil {
		return result, err
	}

	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	command := exec.Command(goBinary, "test", "-count=1", "-v", "./...")
	command.Dir = work
	// GOWORK=off is the isolation claim: no workspace file may resolve the
	// import, so only the replace directive above can.
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	output, testErr := command.CombinedOutput()
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "--- PASS:"):
			result.Tests++
		case strings.HasPrefix(trimmed, "--- FAIL:"):
			result.Tests++
			result.Failures++
		}
	}
	if testErr != nil {
		fmt.Fprintln(os.Stderr, string(output))
		return result, fmt.Errorf("isolated canary failed: %w", testErr)
	}
	if result.Tests == 0 {
		return result, fmt.Errorf("isolated canary ran no tests")
	}
	result.Status = "PASS"
	return result, nil
}
