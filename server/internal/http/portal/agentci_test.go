package portal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ── Stubs for agent CI portal tests ──────────────────────────────────────────

// These stubs mirror the release test stubs but are tuned for agent CI paths.
// Agent CI routes use /api/v1/skills/{skillID}/ci/..., so we need FindByID on
// skill repo rather than FindByNamespaceIDAndSlug.

type ciSkillRepo struct {
	skills map[int64]skill.Skill
}

func newCISkillRepo(skills ...skill.Skill) *ciSkillRepo {
	m := make(map[int64]skill.Skill)
	for _, s := range skills {
		m[s.ID] = s
	}
	return &ciSkillRepo{skills: m}
}

func (r *ciSkillRepo) FindByID(_ context.Context, id int64) (*skill.Skill, error) {
	if s, ok := r.skills[id]; ok {
		return &s, nil
	}
	return nil, nil
}
func (r *ciSkillRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, id := range ids {
		if s, ok := r.skills[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r *ciSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error)       { return nil, nil }
func (r *ciSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r *ciSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, _ int64, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r *ciSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, _ int64, _, _ string) (*skill.Skill, error) {
	return nil, nil
}
func (r *ciSkillRepo) FindByOwnerID(_ context.Context, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r *ciSkillRepo) FindBySlug(_ context.Context, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r *ciSkillRepo) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (r *ciSkillRepo) Save(_ context.Context, s skill.Skill) (skill.Skill, error) { return s, nil }
func (r *ciSkillRepo) Delete(_ context.Context, _ int64) error                    { return nil }
func (r *ciSkillRepo) IncrementDownloadCount(_ context.Context, _ int64) error     { return nil }
func (r *ciSkillRepo) IncrementSubscriptionCount(_ context.Context, _ int64) error { return nil }
func (r *ciSkillRepo) DecrementSubscriptionCount(_ context.Context, _ int64) error { return nil }

// ciVersionRepo returns a PUBLISHED version for any requested ID.
type ciVersionRepo struct{}

func (r ciVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	return &skill.SkillVersion{ID: id, SkillID: 100, Version: "1.0.0", Status: "PUBLISHED"}, nil
}
func (r ciVersionRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ciVersionRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ciVersionRepo) FindBySkillIDAndVersion(_ context.Context, _ int64, _ string) (*skill.SkillVersion, error) {
	return nil, nil
}
func (r ciVersionRepo) FindBySkillIDAndStatus(_ context.Context, _ int64, _ string) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ciVersionRepo) Save(_ context.Context, _ skill.SkillVersion) (skill.SkillVersion, error) {
	return skill.SkillVersion{}, nil
}
func (r ciVersionRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r ciVersionRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type ciFileRepo struct{}

func (r ciFileRepo) FindByVersionID(_ context.Context, _ int64) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r ciFileRepo) Save(_ context.Context, _ skill.SkillFile) (skill.SkillFile, error) {
	return skill.SkillFile{}, nil
}
func (r ciFileRepo) SaveAll(_ context.Context, _ []skill.SkillFile) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r ciFileRepo) DeleteByVersionID(_ context.Context, _ int64) error { return nil }

type ciTagRepo struct{}

func (r ciTagRepo) FindBySkillIDAndTagName(_ context.Context, _ int64, _ string) (*skill.SkillTag, error) {
	return nil, nil
}
func (r ciTagRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillTag, error) { return nil, nil }
func (r ciTagRepo) Save(_ context.Context, _ skill.SkillTag) (skill.SkillTag, error) {
	return skill.SkillTag{}, nil
}
func (r ciTagRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r ciTagRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type ciStore struct{}

func (s ciStore) PutObject(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}
func (s ciStore) GetObject(_ context.Context, _ string) (io.ReadCloser, error) { return nil, nil }
func (s ciStore) DeleteObject(_ context.Context, _ string) error               { return nil }
func (s ciStore) DeleteObjects(_ context.Context, _ []string) error            { return nil }
func (s ciStore) Exists(_ context.Context, _ string) (bool, error)             { return false, nil }
func (s ciStore) Metadata(_ context.Context, _ string) (storage.ObjectMetadata, error) {
	return storage.ObjectMetadata{}, nil
}
func (s ciStore) PresignedURL(_ context.Context, _ string, _ time.Duration, _ string) (string, error) {
	return "", nil
}

type ciNsRepo struct{}

func (r ciNsRepo) FindBySlug(_ context.Context, _ string) (*namespace.Namespace, error) {
	return nil, nil
}
func (r ciNsRepo) FindByID(_ context.Context, _ int64) (*namespace.Namespace, error) {
	return nil, nil
}
func (r ciNsRepo) FindByIDs(_ context.Context, _ []int64) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r ciNsRepo) FindByStatus(_ context.Context, _ string) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r ciNsRepo) Save(_ context.Context, _ namespace.Namespace) (namespace.Namespace, error) {
	return namespace.Namespace{}, nil
}
func (r ciNsRepo) Delete(_ context.Context, _ int64) error { return nil }

// ── Agent CI stub run/check repos ───────────────────────────────────────────

type ciRunRepo struct {
	runs   map[int64]agentci.PipelineRun
	nextID int64
}

func newCIRunRepo(runs ...agentci.PipelineRun) *ciRunRepo {
	m := make(map[int64]agentci.PipelineRun)
	for i, r := range runs {
		id := int64(i + 1)
		r.ID = id
		m[id] = r
	}
	return &ciRunRepo{runs: m, nextID: int64(len(runs) + 1)}
}

func (r *ciRunRepo) Create(_ context.Context, pr agentci.PipelineRun) (agentci.PipelineRun, error) {
	pr.ID = r.nextID
	r.nextID++
	r.runs[pr.ID] = pr
	return pr, nil
}
func (r *ciRunRepo) FindByID(_ context.Context, id int64) (*agentci.PipelineRun, error) {
	if pr, ok := r.runs[id]; ok {
		return &pr, nil
	}
	return nil, nil
}
func (r *ciRunRepo) FindBySkillID(_ context.Context, skillID int64, offset, limit int) ([]agentci.PipelineRun, error) {
	return nil, nil
}
func (r *ciRunRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) { return 0, nil }
func (r *ciRunRepo) FindByVersionID(_ context.Context, versionID int64) ([]agentci.PipelineRun, error) {
	return nil, nil
}
func (r *ciRunRepo) FindByReleaseID(_ context.Context, releaseID int64) ([]agentci.PipelineRun, error) {
	return nil, nil
}
func (r *ciRunRepo) FindPending(_ context.Context, limit int) ([]agentci.PipelineRun, error) {
	return nil, nil
}
func (r *ciRunRepo) ClaimPending(_ context.Context, id int64) (*agentci.PipelineRun, error) {
	return nil, nil
}
func (r *ciRunRepo) Update(_ context.Context, pr agentci.PipelineRun) (agentci.PipelineRun, error) {
	r.runs[pr.ID] = pr
	return pr, nil
}

type ciCheckRepo struct {
	checks map[int64]agentci.CheckRun
	nextID int64
}

func newCICheckRepo(checks ...agentci.CheckRun) *ciCheckRepo {
	m := make(map[int64]agentci.CheckRun)
	for i, c := range checks {
		id := int64(i + 1)
		c.ID = id
		m[id] = c
	}
	return &ciCheckRepo{checks: m, nextID: int64(len(checks) + 1)}
}

func (r *ciCheckRepo) Create(_ context.Context, cr agentci.CheckRun) (agentci.CheckRun, error) {
	cr.ID = r.nextID
	r.nextID++
	r.checks[cr.ID] = cr
	return cr, nil
}
func (r *ciCheckRepo) FindByID(_ context.Context, id int64) (*agentci.CheckRun, error) {
	if cr, ok := r.checks[id]; ok {
		return &cr, nil
	}
	return nil, nil
}
func (r *ciCheckRepo) FindByPipelineRunID(_ context.Context, runID int64) ([]agentci.CheckRun, error) {
	return nil, nil
}
func (r *ciCheckRepo) FindBySkillID(_ context.Context, skillID int64, offset, limit int) ([]agentci.CheckRun, error) {
	return nil, nil
}
func (r *ciCheckRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) { return 0, nil }
func (r *ciCheckRepo) FindByVersionID(_ context.Context, versionID int64) ([]agentci.CheckRun, error) {
	return nil, nil
}
func (r *ciCheckRepo) FindByReleaseID(_ context.Context, releaseID int64) ([]agentci.CheckRun, error) {
	return nil, nil
}
func (r *ciCheckRepo) Update(_ context.Context, cr agentci.CheckRun) (agentci.CheckRun, error) {
	r.checks[cr.ID] = cr
	return cr, nil
}

// ── Helper to build an AgentCIHandler with stubs ────────────────────────────

// newTestAgentCIHandler creates an AgentCIHandler with the given skill in the repo
// and the given pipeline run pre-seeded.
func newTestAgentCIHandler(skills ...skill.Skill) *AgentCIHandler {
	querySvc := skill.NewSkillQueryService(
		ciNsRepo{},
		newCISkillRepo(skills...),
		ciVersionRepo{},
		ciFileRepo{},
		ciTagRepo{},
		ciStore{},
		nil,
	)

	skillSvc := &skill.Service{Query: querySvc, Visibility: skill.NewVisibilityChecker()}

	ciSvc := agentci.NewService(
		nil, // pipelineDefRepo
		newCIRunRepo(),
		newCICheckRepo(),
		nil, nil, nil, nil, nil,
	)

	return &AgentCIHandler{
		AgentCISvc: ciSvc,
		SkillSvc:   skillSvc,
	}
}

// ── Tests: pipeline run cross-skill scoping ─────────────────────────────────

func TestAgentCI_GetPipelineRun_CrossSkillScoping(t *testing.T) {
	// Pipeline run belongs to skill 999 but path requests skill 100.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PUBLIC", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	// Seed a pipeline run belonging to skill 999.
	h.AgentCISvc = agentci.NewService(
		nil,
		newCIRunRepo(agentci.PipelineRun{SkillID: 999, Status: agentci.RunStatusCompleted}),
		newCICheckRepo(),
		nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs/1", nil)
	req.SetPathValue("skillID", "100")
	req.SetPathValue("runID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleGetPipelineRun(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for pipeline run belonging to different skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_GetPipelineRun_MatchingSkill(t *testing.T) {
	// Pipeline run belongs to skill 100 — same as path.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PUBLIC", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	h.AgentCISvc = agentci.NewService(
		nil,
		newCIRunRepo(agentci.PipelineRun{SkillID: 100, Status: agentci.RunStatusCompleted}),
		newCICheckRepo(),
		nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs/1", nil)
	req.SetPathValue("skillID", "100")
	req.SetPathValue("runID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleGetPipelineRun(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for matching skill pipeline run, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_GetCheckRun_CrossSkillScoping(t *testing.T) {
	// Check run belongs to skill 999 but path requests skill 100.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PUBLIC", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	h.AgentCISvc = agentci.NewService(
		nil,
		nil,
		newCICheckRepo(agentci.CheckRun{SkillID: 999, PipelineRunID: 1, Status: agentci.CheckStatusPassed}),
		nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/checks/1", nil)
	req.SetPathValue("skillID", "100")
	req.SetPathValue("checkID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleGetCheckRun(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for check run belonging to different skill, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Tests: private skill visibility enforcement ─────────────────────────────

func TestAgentCI_PrivateSkill_Unauthenticated(t *testing.T) {
	// Private skill: unauthenticated user gets 401.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PRIVATE", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs", nil)
	req.SetPathValue("skillID", "100")
	// Anonymous principal.
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	h.HandleListPipelineRuns(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated user on private CI skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_PrivateSkill_StrangerForbidden(t *testing.T) {
	// Private skill: authenticated but unauthorized user gets 403.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PRIVATE", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs", nil)
	req.SetPathValue("skillID", "100")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u3", // stranger
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleListPipelineRuns(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unauthorized user on private CI skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_PrivateSkill_OwnerCanAccess(t *testing.T) {
	// Private skill: owner can access CI data.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PRIVATE", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	h.AgentCISvc = agentci.NewService(
		nil,
		newCIRunRepo(), // empty — should return empty list, not error
		newCICheckRepo(),
		nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs", nil)
	req.SetPathValue("skillID", "100")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1", // owner
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleListPipelineRuns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner on private CI skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_PublicSkill_AnonymousCanAccess(t *testing.T) {
	// Public skill: anonymous user can access CI list.
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PUBLIC", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	h.AgentCISvc = agentci.NewService(
		nil,
		newCIRunRepo(), // empty
		newCICheckRepo(),
		nil, nil, nil, nil, nil,
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/runs", nil)
	req.SetPathValue("skillID", "100")
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	h.HandleListPipelineRuns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for anonymous on public CI skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_GateEvaluation_SkillVisibilityEnforced(t *testing.T) {
	// Gate evaluation applies skill visibility check.  Anonymous caller on a
	// private skill gets 401 from resolveCISkill.  (RequireAuth is route-level
	// middleware tested separately; this test verifies the handler itself.)
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PRIVATE", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/gates?trigger=release_publish", nil)
	req.SetPathValue("skillID", "100")
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	h.HandleEvaluateGates(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for anonymous on private skill gate evaluation, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentCI_GateEvaluation_AuthenticatedAllowedOnPublicSkill(t *testing.T) {
	// Authenticated caller on public skill can evaluate gates.
	// (RequireAuth is route-level middleware; handler allows authenticated
	// callers with visibility access.)
	h := newTestAgentCIHandler(
		skill.Skill{ID: 100, NamespaceID: 10, Slug: "myskill", OwnerID: "u1", Visibility: "PUBLIC", Status: "ACTIVE", LatestVersionID: ptrInt64(1)},
	)

	req := httptest.NewRequest("GET", "/api/v1/skills/100/ci/gates?trigger=release_publish", nil)
	req.SetPathValue("skillID", "100")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.HandleEvaluateGates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated user on public skill gate evaluation, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func ptrInt64(v int64) *int64 { return &v }

var _ = json.Marshal
