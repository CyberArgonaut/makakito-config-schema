package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CyberArgonaut/makakito-config-schema/generated/go/config"
)

// TestValidExamples asserts that every examples/*.json file whose name does NOT
// start with "invalid-" passes schema validation with zero violations.
func TestValidExamples(t *testing.T) {
	t.Helper()

	paths, err := filepath.Glob("../examples/*.json")
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no example files found under examples/")
	}

	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasPrefix(name, "invalid-") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			violations, err := config.Validate(data)
			if err != nil {
				t.Fatalf("validation engine error: %v", err)
			}
			if len(violations) > 0 {
				t.Errorf("%s should be valid but got %d violation(s):\n%s",
					name, len(violations), strings.Join(violations, "\n"))
			}
		})
	}
}

// TestInvalidExamples asserts that every examples/invalid-*.json file fails
// schema validation with at least one violation.
func TestInvalidExamples(t *testing.T) {
	t.Helper()

	paths, err := filepath.Glob("../examples/invalid-*.json")
	if err != nil {
		t.Fatalf("glob invalid examples: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no invalid-*.json files found under examples/")
	}

	for _, path := range paths {
		name := filepath.Base(path)

		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			violations, err := config.Validate(data)
			if err != nil {
				t.Fatalf("validation engine error: %v", err)
			}
			if len(violations) == 0 {
				t.Errorf("%s should be invalid but passed validation with zero violations", name)
			}
		})
	}
}
