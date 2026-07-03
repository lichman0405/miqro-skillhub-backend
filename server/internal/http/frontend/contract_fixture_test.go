package frontend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contractFixtureEnvelope is the minimal shape every fixture must have.
type contractFixtureEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// expectedContractFixtures maps route labels to fixture filenames.
// Every frontend read-model route in the Phase 16 target surface must have a
// representative fixture.
var expectedContractFixtures = map[string]string{
	"search":                        "search.json",
	"skill-detail":                  "skill-detail.json",
	"version-detail":                "version-detail.json",
	"publish-validate":              "publish-validate.json",
	"namespaces":                    "namespaces.json",
	"namespace-detail":              "namespace-detail.json",
	"releases":                      "releases.json",
	"release-detail":                "release-detail.json",
	"community-issue-detail":        "community-issue-detail.json",
	"community-issue-list":          "community-issue-list.json",
	"community-discussion-detail":   "community-discussion-detail.json",
	"community-discussion-list":     "community-discussion-list.json",
	"community-wiki-page":           "community-wiki-page.json",
	"community-wiki-list":           "community-wiki-list.json",
	"community-proposal-detail":     "community-proposal-detail.json",
	"community-proposal-list":       "community-proposal-list.json",
	"reviews":                       "reviews.json",
	"review-detail":                 "review-detail.json",
	"promotions":                    "promotions.json",
	"promotion-detail":              "promotion-detail.json",
	"governance":                    "governance.json",
	"admin":                         "admin.json",
}

func TestContractFixtures_Manifest(t *testing.T) {
	contractsDir := filepath.Join("testdata", "contracts")

	for label, filename := range expectedContractFixtures {
		path := filepath.Join(contractsDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing expected fixture for %q: %s", label, path)
		}
	}
}

func TestContractFixtures_AreValidJSON(t *testing.T) {
	contractsDir := filepath.Join("testdata", "contracts")
	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		t.Fatalf("failed to read contracts directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			path := filepath.Join(contractsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", path, err)
			}

			var env contractFixtureEnvelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("fixture %s is not valid JSON: %v", path, err)
			}

			if !env.Success {
				t.Errorf("fixture %s should have success=true", path)
			}
			if len(env.Data) == 0 {
				t.Errorf("fixture %s should have non-empty data", path)
			}
		})
	}
}

func TestContractFixtures_MatchKnownRouteLabels(t *testing.T) {
	contractsDir := filepath.Join("testdata", "contracts")
	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		t.Fatalf("failed to read contracts directory: %v", err)
	}

	known := make(map[string]bool, len(expectedContractFixtures))
	for _, filename := range expectedContractFixtures {
		known[filename] = true
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if !known[entry.Name()] {
			t.Errorf("fixture %s does not match any expected route label", entry.Name())
		}
	}
}
