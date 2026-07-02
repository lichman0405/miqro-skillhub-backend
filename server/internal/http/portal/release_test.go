package portal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/release"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ── Stubs for portal release tests ──────────────────────────────────────────

// stubNsRepo returns a fixed namespace for slug "ns1".
type stubNsRepo struct{}

func (r stubNsRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	if slug == "ns1" {
		return &namespace.Namespace{ID: 10, Slug: "ns1", Status: "ACTIVE"}, nil
	}
	return nil, nil
}
func (r stubNsRepo) FindByID(_ context.Context, _ int64) (*namespace.Namespace, error) { return nil, nil }
func (r stubNsRepo) FindByIDs(_ context.Context, _ []int64) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r stubNsRepo) FindByStatus(_ context.Context, _ string) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r stubNsRepo) Save(_ context.Context, _ namespace.Namespace) (namespace.Namespace, error) {
	return namespace.Namespace{}, nil
}
func (r stubNsRepo) Delete(_ context.Context, _ int64) error { return nil }

// stubSkillRepo returns a fixed skill for namespace 10, slug "myskill".
type stubSkillRepo struct{}

func (r stubSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
	if nsID == 10 && slug == "myskill" {
		latestID := int64(1)
		return []skill.Skill{{
			ID:              100,
			NamespaceID:     10,
			Slug:            "myskill",
			DisplayName:     "My Skill",
			OwnerID:         "u1",
			Visibility:      "PUBLIC",
			Status:          "ACTIVE",
			LatestVersionID: &latestID,
		}}, nil
	}
	return nil, nil
}
func (r stubSkillRepo) FindByID(_ context.Context, _ int64) (*skill.Skill, error)        { return nil, nil }
func (r stubSkillRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.Skill, error)    { return nil, nil }
func (r stubSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error)                  { return nil, nil }
func (r stubSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r stubSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, _ int64, _, _ string) (*skill.Skill, error) {
	return nil, nil
}
func (r stubSkillRepo) FindByOwnerID(_ context.Context, _ string) ([]skill.Skill, error)  { return nil, nil }
func (r stubSkillRepo) FindBySlug(_ context.Context, _ string) ([]skill.Skill, error)      { return nil, nil }
func (r stubSkillRepo) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error)       { return false, nil }
func (r stubSkillRepo) Save(_ context.Context, _ skill.Skill) (skill.Skill, error)          { return skill.Skill{}, nil }
func (r stubSkillRepo) Delete(_ context.Context, _ int64) error                             { return nil }
func (r stubSkillRepo) IncrementDownloadCount(_ context.Context, _ int64) error             { return nil }
func (r stubSkillRepo) IncrementSubscriptionCount(_ context.Context, _ int64) error         { return nil }
func (r stubSkillRepo) DecrementSubscriptionCount(_ context.Context, _ int64) error         { return nil }

// stubVersionRepo returns a published version for ID 1 (used by GetSkillDetail path).
type stubVersionRepo struct{}

func (r stubVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	return &skill.SkillVersion{ID: id, SkillID: 100, Version: "1.0.0", Status: "PUBLISHED"}, nil
}
func (r stubVersionRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubVersionRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubVersionRepo) FindBySkillIDAndVersion(_ context.Context, _ int64, _ string) (*skill.SkillVersion, error) {
	return nil, nil
}
func (r stubVersionRepo) FindBySkillIDAndStatus(_ context.Context, _ int64, _ string) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubVersionRepo) Save(_ context.Context, _ skill.SkillVersion) (skill.SkillVersion, error) {
	return skill.SkillVersion{}, nil
}
func (r stubVersionRepo) Delete(_ context.Context, _ int64) error               { return nil }
func (r stubVersionRepo) DeleteBySkillID(_ context.Context, _ int64) error      { return nil }

// stubFileRepo — no files needed for GetSkillDetail.
type stubFileRepo struct{}

func (r stubFileRepo) FindByVersionID(_ context.Context, _ int64) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r stubFileRepo) Save(_ context.Context, _ skill.SkillFile) (skill.SkillFile, error) {
	return skill.SkillFile{}, nil
}
func (r stubFileRepo) SaveAll(_ context.Context, _ []skill.SkillFile) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r stubFileRepo) DeleteByVersionID(_ context.Context, _ int64) error { return nil }

// stubTagRepo — no tags needed.
type stubTagRepo struct{}

func (r stubTagRepo) FindBySkillIDAndTagName(_ context.Context, _ int64, _ string) (*skill.SkillTag, error) {
	return nil, nil
}
func (r stubTagRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillTag, error) {
	return nil, nil
}
func (r stubTagRepo) Save(_ context.Context, _ skill.SkillTag) (skill.SkillTag, error) {
	return skill.SkillTag{}, nil
}
func (r stubTagRepo) Delete(_ context.Context, _ int64) error            { return nil }
func (r stubTagRepo) DeleteBySkillID(_ context.Context, _ int64) error   { return nil }

// stubStore — no storage operations needed.
type stubStore struct{}

func (s stubStore) PutObject(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}
func (s stubStore) GetObject(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (s stubStore) DeleteObject(_ context.Context, _ string) error    { return nil }
func (s stubStore) DeleteObjects(_ context.Context, _ []string) error { return nil }
func (s stubStore) Exists(_ context.Context, _ string) (bool, error)  { return false, nil }
func (s stubStore) Metadata(_ context.Context, _ string) (storage.ObjectMetadata, error) {
	return storage.ObjectMetadata{}, nil
}
func (s stubStore) PresignedURL(_ context.Context, _ string, _ time.Duration, _ string) (string, error) {
	return "", nil
}

// ── Release-specific stubs ──────────────────────────────────────────────────

// stubReleaseRepo is a configurable in-memory release repository.
type stubReleaseRepo struct {
	releases map[int64]release.Release
	nextID   int64
}

func newStubReleaseRepo(releases ...release.Release) *stubReleaseRepo {
	m := make(map[int64]release.Release)
	for i, r := range releases {
		id := int64(i + 1)
		r.ID = id
		m[id] = r
	}
	return &stubReleaseRepo{releases: m, nextID: int64(len(releases) + 1)}
}

func (r *stubReleaseRepo) Create(_ context.Context, rel release.Release) (release.Release, error) {
	rel.ID = r.nextID
	r.nextID++
	r.releases[rel.ID] = rel
	return rel, nil
}
func (r *stubReleaseRepo) Update(_ context.Context, rel release.Release) (release.Release, error) {
	r.releases[rel.ID] = rel
	return rel, nil
}
func (r *stubReleaseRepo) FindByID(_ context.Context, id int64) (*release.Release, error) {
	if rel, ok := r.releases[id]; ok {
		return &rel, nil
	}
	return nil, nil
}
func (r *stubReleaseRepo) FindBySkillID(_ context.Context, _ int64) ([]release.Release, error) {
	return nil, nil
}
func (r *stubReleaseRepo) FindByVersionIDAndChannel(_ context.Context, _ int64, _ string) (*release.Release, error) {
	return nil, nil
}
func (r *stubReleaseRepo) FindLatestStable(_ context.Context, _ int64, _ string) (*release.Release, error) {
	return nil, nil
}
func (r *stubReleaseRepo) Delete(_ context.Context, id int64) error {
	delete(r.releases, id)
	return nil
}
func (r *stubReleaseRepo) ListBySkillIDPaginated(_ context.Context, _ int64, _ int, _ int) ([]release.Release, error) {
	return nil, nil
}
func (r *stubReleaseRepo) CountBySkillID(_ context.Context, _ int64) (int64, error) { return 0, nil }

// ── Helper to build a ReleaseHandler with all stubs ─────────────────────────

// newTestReleaseHandler creates a ReleaseHandler backed by stubs.
// skillID is the SkillID resolved from the path (skill with namespace "ns1", slug "myskill").
// releaseSkillID is the SkillID on the pre-created release.
func newTestReleaseHandler(releaseSkillID int64) *ReleaseHandler {
	// Build SkillQueryService with stubs.
	querySvc := skill.NewSkillQueryService(
		stubNsRepo{},
		stubSkillRepo{},
		stubVersionRepo{},
		stubFileRepo{},
		stubTagRepo{},
		stubStore{},
		nil, // visibility checker (nil = default)
	)

	skillSvc := &skill.Service{
		Query: querySvc,
	}

	// Build release service with a stub release repo pre-seeded with one release.
	r := release.Release{
		SkillID:     releaseSkillID,
		VersionID:   1,
		Channel:     "stable",
		Title:       "Test Release",
		PublisherID: "u1",
	}
	releaseRepo := newStubReleaseRepo(r)

	// Version repo that returns published version 1 for skill 100 (matching the
	// resolved skill — needed by the SDK create path but not used by GET/PATCH/DELETE).
	verRepo := stubVersionRepo{}
	releaseSvc := release.NewService(releaseRepo, nil, verRepo)

	return &ReleaseHandler{
		ReleaseSvc: releaseSvc,
		SkillSvc:   skillSvc,
	}
}

// ── Portal release path-scope tests ─────────────────────────────────────────

func TestRelease_Get_WrongPathSkill(t *testing.T) {
	// Release has SkillID=999 but the path resolves to SkillID=100.
	h := newTestReleaseHandler(999)

	req := httptest.NewRequest("GET", "/api/v1/skills/ns1/myskill/releases/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("releaseID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.handleGetRelease(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for release belonging to different skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRelease_Update_WrongPathSkill(t *testing.T) {
	// Release has SkillID=999 but the path resolves to SkillID=100.
	h := newTestReleaseHandler(999)

	body := `{"title": "Updated"}`
	req := httptest.NewRequest("PATCH", "/api/v1/skills/ns1/myskill/releases/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("releaseID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.handleUpdateRelease(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for update on release belonging to different skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRelease_Delete_WrongPathSkill(t *testing.T) {
	// Release has SkillID=999 but the path resolves to SkillID=100.
	h := newTestReleaseHandler(999)

	req := httptest.NewRequest("DELETE", "/api/v1/skills/ns1/myskill/releases/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("releaseID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.handleDeleteRelease(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for delete on release belonging to different skill, got %d: %s", w.Code, w.Body.String())
	}
}

// Ensure unused imports don't cause compile errors.
var _ = json.Marshal
var _ = io.EOF
