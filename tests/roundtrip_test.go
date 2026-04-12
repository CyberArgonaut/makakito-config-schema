package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/CyberArgonaut/makakito-config-schema/generated/go/config"
)

// TestRoundtrip verifies that every valid example can survive a full
// unmarshal → marshal → unmarshal cycle without data loss.
//
// Why map[string]any comparison instead of string equality:
// json.Marshal does not guarantee the same key order as the source file.
// Comparing via two map[string]any unmarshals avoids false failures due to
// key-ordering differences while still catching dropped or mutated fields.
//
// The critical case exercised here is microservices-full.json, which sets
// report.includeMetricsSnapshot to false. If that field were typed as plain
// bool (not *bool) in the Go struct, json.Marshal with omitempty would drop it
// silently and this test would catch the regression.
func TestRoundtrip(t *testing.T) {
	t.Helper()

	paths, err := filepath.Glob("../examples/*.json")
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}

	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasPrefix(name, "invalid-") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			// Step 1: parse via the typed API (validates + unmarshals).
			cfg, err := config.Parse(original)
			if err != nil {
				t.Fatalf("config.Parse(%s): %v", name, err)
			}

			// Step 2: remarshal to JSON.
			remarshalled, err := json.Marshal(cfg)
			if err != nil {
				t.Fatalf("json.Marshal(%s): %v", name, err)
			}

			// Step 3: unmarshal both into map[string]any for structural comparison.
			var originalMap, remarshalledMap map[string]any
			if err := json.Unmarshal(original, &originalMap); err != nil {
				t.Fatalf("unmarshal original %s: %v", name, err)
			}
			if err := json.Unmarshal(remarshalled, &remarshalledMap); err != nil {
				t.Fatalf("unmarshal remarshalled %s: %v", name, err)
			}

			if !reflect.DeepEqual(originalMap, remarshalledMap) {
				origPretty, _ := json.MarshalIndent(originalMap, "", "  ")
				remPretty, _ := json.MarshalIndent(remarshalledMap, "", "  ")
				t.Errorf("%s roundtrip mismatch.\noriginal:\n%s\n\nremarshalled:\n%s",
					name, origPretty, remPretty)
			}
		})
	}
}
