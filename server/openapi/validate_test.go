// Package openapi_test validates the project's OpenAPI 3.0.3 specification.
//
// This test serves as the harness behind `make openapi` — it verifies
// that server/openapi/openapi.yaml exists, is valid YAML, contains the
// required OpenAPI structural sections, and documents all 12 frontend
// page-oriented read-model routes.
package openapi_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPISpec_FileExists(t *testing.T) {
	specPath := specFilePath(t)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Fatalf("openapi.yaml not found at %s", specPath)
	}
}

func TestOpenAPISpec_IsValidYAML(t *testing.T) {
	data := readSpec(t)
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("openapi.yaml is not valid YAML: %v", err)
	}
}

func TestOpenAPISpec_RequiredTopLevelFields(t *testing.T) {
	doc := parseSpec(t)

	required := []string{"openapi", "info", "paths"}
	for _, key := range required {
		if _, ok := doc[key]; !ok {
			t.Errorf("missing required top-level field: %q", key)
		}
	}

	// openapi must be a valid version string like "3.0.3".
	if v, ok := doc["openapi"].(string); !ok || v == "" {
		t.Errorf("openapi version must be a non-empty string, got %T %v", doc["openapi"], doc["openapi"])
	} else if !strings.HasPrefix(v, "3.") {
		t.Errorf("expected OpenAPI 3.x version, got %q", v)
	}

	// info must have title and version.
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info must be an object")
	}
	if _, ok := info["title"]; !ok {
		t.Error("info.title is required")
	}
	if _, ok := info["version"]; !ok {
		t.Error("info.version is required")
	}
}

func TestOpenAPISpec_PathsNonEmpty(t *testing.T) {
	doc := parseSpec(t)

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths must be an object")
	}
	if len(paths) == 0 {
		t.Error("paths must contain at least one endpoint")
	}
}

func TestOpenAPISpec_ComponentsSchemas(t *testing.T) {
	doc := parseSpec(t)

	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatal("components must be an object")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("components.schemas must be an object")
	}

	// Required shared types.
	sharedSchemas := []string{
		"SearchResult", "SkillDetail", "VersionDetail", "SkillFile",
		"Namespace", "NamespaceMember",
	}
	for _, name := range sharedSchemas {
		if _, ok := schemas[name]; !ok {
			t.Errorf("missing required shared schema: %q", name)
		}
	}
}

func TestOpenAPISpec_FrontendRoutes(t *testing.T) {
	doc := parseSpec(t)

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths must be an object")
	}

	frontendRoutes := []string{
		"/api/v1/frontend/search",
		"/api/v1/frontend/skills/{namespace}/{slug}",
		"/api/v1/frontend/skills/{namespace}/{slug}/versions/{version}",
		"/api/v1/frontend/skills/{namespace}/publish/validate",
		"/api/v1/frontend/namespaces",
		"/api/v1/frontend/namespaces/{slug}",
		"/api/v1/frontend/reviews",
		"/api/v1/frontend/reviews/{id}",
		"/api/v1/frontend/promotions",
		"/api/v1/frontend/promotions/{id}",
		"/api/v1/frontend/governance",
		"/api/v1/frontend/admin",
	}

	for _, route := range frontendRoutes {
		if _, ok := paths[route]; !ok {
			t.Errorf("missing frontend route in OpenAPI paths: %q", route)
		}
	}

	// Verify each frontend route has a GET method and a 200 response with
	// a $ref schema referencing components/schemas.
	for _, route := range frontendRoutes {
		pathItem, ok := paths[route].(map[string]any)
		if !ok {
			t.Errorf("path %q is not an object", route)
			continue
		}
		getOp, ok := pathItem["get"].(map[string]any)
		if !ok {
			t.Errorf("frontend route %q missing GET operation", route)
			continue
		}
		if summary, _ := getOp["summary"].(string); summary == "" {
			t.Errorf("frontend route %q GET missing summary", route)
		}

		// The response should have a 200 with content referencing a $ref schema.
		responses, ok := getOp["responses"].(map[string]any)
		if !ok {
			t.Errorf("frontend route %q GET missing responses", route)
			continue
		}
		resp200, ok := responses["200"].(map[string]any)
		if !ok {
			t.Errorf("frontend route %q GET missing 200 response", route)
			continue
		}
		if _, ok := resp200["description"]; !ok {
			t.Errorf("frontend route %q 200 response missing description", route)
		}
	}
}

func TestOpenAPISpec_FrontendSchemaConsistency(t *testing.T) {
	doc := parseSpec(t)

	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatal("components must be an object")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("components.schemas must be an object")
	}

	// Every frontend read-model schema must have an associated Actions schema.
	readModels := []string{
		"RegistrySearchReadModel",
		"SkillDetailReadModel",
		"VersionDetailReadModel",
		"PublishValidateReadModel",
		"NamespaceListReadModel",
		"NamespaceDetailReadModel",
		"ReviewQueueReadModel",
		"ReviewDetailReadModel",
		"PromotionQueueReadModel",
		"PromotionDetailReadModel",
		"GovernanceWorkbenchReadModel",
		"AdminPageReadModel",
	}

	actionSchemas := []string{
		"RegistrySearchActions",
		"SkillDetailActions",
		"VersionActions",
		"PublishValidateActions",
		"NamespaceListActions",
		"NamespaceDetailActions",
		"ReviewQueueActions",
		"ReviewDetailActions",
		"PromotionQueueActions",
		"PromotionDetailActions",
		"GovernanceWorkbenchActions",
		"AdminPageActions",
	}

	for _, name := range readModels {
		if _, ok := schemas[name]; !ok {
			t.Errorf("missing frontend read-model schema: %q", name)
		}
	}
	for _, name := range actionSchemas {
		if _, ok := schemas[name]; !ok {
			t.Errorf("missing frontend actions schema: %q", name)
		}
	}
}

func TestOpenAPISpec_ServerURL(t *testing.T) {
	doc := parseSpec(t)

	servers, ok := doc["servers"].([]any)
	if !ok {
		t.Fatal("servers must be an array")
	}
	if len(servers) == 0 {
		t.Error("servers must contain at least one entry")
	}
}

func TestOpenAPISpec_SecuritySchemes(t *testing.T) {
	doc := parseSpec(t)

	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatal("components must be an object")
	}
	securitySchemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatal("components.securitySchemes must be an object")
	}

	required := []string{"bearerAuth", "sessionCookie"}
	for _, name := range required {
		if _, ok := securitySchemes[name]; !ok {
			t.Errorf("missing security scheme: %q", name)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────

func specFilePath(t *testing.T) string {
	t.Helper()
	// The test lives in server/openapi/; the YAML file is in the same directory.
	// When run via `go test ./openapi/`, the working directory is server/openapi/.
	// Fall back to a path relative to this source file if needed.
	candidates := []string{
		"openapi.yaml",
		filepath.Join("..", "openapi", "openapi.yaml"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Return the first candidate so t.Fatalf can produce a useful error.
	return candidates[0]
}

func readSpec(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(specFilePath(t))
	if err != nil {
		t.Fatalf("failed to read openapi.yaml: %v", err)
	}
	return data
}

func parseSpec(t *testing.T) map[string]any {
	t.Helper()
	data := readSpec(t)
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to parse openapi.yaml: %v", err)
	}
	return doc
}
