package portal

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// AgentCIHandler exposes agent CI query endpoints.
// Write operations (trigger pipeline) go through the tooling API.
type AgentCIHandler struct {
	AgentCISvc *agentci.Service
	SkillSvc   *skill.Service
}

// resolveCISkill resolves skillID to a skill and checks that the requesting
// user is authorized to view CI data, reusing the same visibility rules as
// skill queries (VisibilityChecker.CanAccess).
//
// Rules:
//   - PUBLIC: visible to everyone (anonymous OK)
//   - NAMESPACE_ONLY: visible to namespace members, admins, owner, super admins
//   - PRIVATE: visible to owner, namespace admin, super admin
//   - Hidden or no LatestVersionID: follows CanAccess rules
func (h *AgentCIHandler) resolveCISkill(w http.ResponseWriter, r *http.Request, skillID int64) (*skill.Skill, bool) {
	if h.SkillSvc == nil || h.SkillSvc.Query == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "skill service not available"})
		return nil, false
	}

	sk, err := h.SkillSvc.Query.GetSkillByID(r.Context(), skillID)
	if err != nil {
		middleware.WriteError(w, err)
		return nil, false
	}
	if sk == nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "skill not found"})
		return nil, false
	}

	// Use the same visibility checker as the skill query service.
	p := middleware.GetPrincipal(r)
	checker := h.SkillSvc.Visibility
	if checker == nil {
		checker = skill.NewVisibilityChecker()
	}
	if !checker.CanAccess(*sk, p.UserID, p.NamespaceRoles, p.PlatformRoles) {
		// For NAMESPACE_ONLY and PRIVATE: unauthenticated gets 401,
		// authenticated-but-unauthorized gets 403.
		if !p.IsAuthenticated {
			middleware.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		} else {
			middleware.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "access denied"})
		}
		return nil, false
	}

	return sk, true
}

// RegisterAgentCIRoutes registers agent CI query routes on the given mux.
// Read routes use optional auth (like community read routes).
// Mutating routes (gates evaluation) require authenticated users.
func (h *AgentCIHandler) RegisterAgentCIRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	optAuth := func(next http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			return authMW.Authenticate(next)
		}
		return next
	}
	_ = rl

	// Read routes — optional auth.
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs", optAuth(h.HandleListPipelineRuns))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs/{runID}", optAuth(h.HandleGetPipelineRun))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs/{runID}/checks", optAuth(h.HandleListCheckRuns))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/checks/{checkID}", optAuth(h.HandleGetCheckRun))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/checks/{checkID}/artifacts", optAuth(h.HandleListArtifacts))

	// Gate evaluation — requires authenticated user.
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/gates",
		authMW.Authenticate(middleware.RequireAuth(h.HandleEvaluateGates)))
}

// ── Pipeline run queries ────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleListPipelineRuns(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}

	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	result, err := h.AgentCISvc.ListPipelineRuns(r.Context(), agentci.PipelineRunFilter{
		SkillID: skillID,
		Page:    page,
		Size:    size,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *AgentCIHandler) HandleGetPipelineRun(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}
	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	runID, err := strconv.ParseInt(r.PathValue("runID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run ID"})
		return
	}

	run, err := h.AgentCISvc.GetPipelineRun(r.Context(), runID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if run.SkillID != skillID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "pipeline run not found"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, run)
}

// ── Check run queries ───────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleListCheckRuns(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}
	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	runID, err := strconv.ParseInt(r.PathValue("runID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run ID"})
		return
	}

	// Verify the run belongs to this skill first.
	run, err := h.AgentCISvc.GetPipelineRun(r.Context(), runID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if run.SkillID != skillID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "pipeline run not found"})
		return
	}

	checks, err := h.AgentCISvc.ListCheckRuns(r.Context(), runID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, checks)
}

func (h *AgentCIHandler) HandleGetCheckRun(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}
	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	checkID, err := strconv.ParseInt(r.PathValue("checkID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid check ID"})
		return
	}

	check, err := h.AgentCISvc.GetCheckRun(r.Context(), checkID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if check.SkillID != skillID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "check run not found"})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, check)
}

// ── Artifact queries ────────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleListArtifacts(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}
	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	checkID, err := strconv.ParseInt(r.PathValue("checkID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid check ID"})
		return
	}

	// Verify the check belongs to this skill.
	check, err := h.AgentCISvc.GetCheckRun(r.Context(), checkID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if check.SkillID != skillID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "check run not found"})
		return
	}

	artifacts, err := h.AgentCISvc.ListArtifacts(r.Context(), checkID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, artifacts)
}

// ── Gate evaluation ─────────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleEvaluateGates(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	skillID, err := strconv.ParseInt(r.PathValue("skillID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid skill ID"})
		return
	}

	if _, ok := h.resolveCISkill(w, r, skillID); !ok {
		return
	}

	triggerType := r.URL.Query().Get("trigger")
	if triggerType == "" {
		triggerType = "release_publish"
	}

	versionIDStr := r.URL.Query().Get("versionId")
	releaseIDStr := r.URL.Query().Get("releaseId")

	var req agentci.GateEvalRequest
	req.SkillID = skillID
	req.TriggerType = triggerType
	if versionIDStr != "" {
		vID, _ := strconv.ParseInt(versionIDStr, 10, 64)
		req.VersionID = &vID
	}
	if releaseIDStr != "" {
		rID, _ := strconv.ParseInt(releaseIDStr, 10, 64)
		req.ReleaseID = &rID
	}

	result, err := h.AgentCISvc.EvaluateGates(r.Context(), req)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}
