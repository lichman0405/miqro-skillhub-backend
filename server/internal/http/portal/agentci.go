package portal

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

func optAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}
}

// RegisterAgentCIRoutes registers agent CI query routes on the given mux.
func (h *AgentCIHandler) RegisterAgentCIRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	_ = authMW
	_ = rl
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs", optAuth(h.HandleListPipelineRuns))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs/{runID}", optAuth(h.HandleGetPipelineRun))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/runs/{runID}/checks", optAuth(h.HandleListCheckRuns))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/checks/{checkID}", optAuth(h.HandleGetCheckRun))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/checks/{checkID}/artifacts", optAuth(h.HandleListArtifacts))
	mux.HandleFunc("GET /api/v1/skills/{skillID}/ci/gates", optAuth(h.HandleEvaluateGates))
}

// AgentCIHandler exposes agent CI query endpoints.
// Write operations (trigger pipeline) go through the tooling API.
type AgentCIHandler struct {
	AgentCISvc *agentci.Service
	SkillSvc   *skill.Service
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

	middleware.WriteJSON(w, http.StatusOK, run)
}

// ── Check run queries ───────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleListCheckRuns(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	runID, err := strconv.ParseInt(r.PathValue("runID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid run ID"})
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

	middleware.WriteJSON(w, http.StatusOK, check)
}

// ── Artifact queries ────────────────────────────────────────────────────────

func (h *AgentCIHandler) HandleListArtifacts(w http.ResponseWriter, r *http.Request) {
	if h.AgentCISvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "agent CI service not available"})
		return
	}

	checkID, err := strconv.ParseInt(r.PathValue("checkID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid check ID"})
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
