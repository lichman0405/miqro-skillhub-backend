package frontend

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
)

// ReleaseListView is a release entry in the release list.
type ReleaseListView struct {
	ID           int64  `json:"id"`
	VersionID    int64  `json:"versionId"`
	Channel      string `json:"channel"`
	Title        string `json:"title"`
	Draft        bool   `json:"draft"`
	Prerelease   bool   `json:"prerelease"`
	Yanked       bool   `json:"yanked"`
	PublishedAt  string `json:"publishedAt,omitempty"`
	PublisherID  string `json:"publisherId"`
}

// ReleaseListReadModel is the page-level release list response.
type ReleaseListReadModel struct {
	Releases         []ReleaseListView  `json:"releases"`
	TotalCount       int64              `json:"totalCount"`
	Page             int                `json:"page"`
	Size             int                `json:"size"`
	AvailableActions ReleaseListActions `json:"availableActions"`
}

// ReleaseListActions lists viewer-specific actions for the release list.
type ReleaseListActions struct {
	CanCreateRelease bool `json:"canCreateRelease"`
}

// ReleaseDetailReadModel is the page-level release detail response.
type ReleaseDetailReadModel struct {
	Release          ReleaseDetailView    `json:"release"`
	Assets           []ReleaseAssetView   `json:"assets,omitempty"`
	AvailableActions ReleaseDetailActions `json:"availableActions"`
}

// ReleaseDetailView is a detailed view of a single release.
type ReleaseDetailView struct {
	ID           int64  `json:"id"`
	SkillID      int64  `json:"skillId"`
	VersionID    int64  `json:"versionId"`
	Channel      string `json:"channel"`
	Title        string `json:"title"`
	Notes        string `json:"notes,omitempty"`
	Draft        bool   `json:"draft"`
	Prerelease   bool   `json:"prerelease"`
	Yanked       bool   `json:"yanked"`
	PublishedAt  string `json:"publishedAt,omitempty"`
	PublisherID  string `json:"publisherId"`
	ReviewerID   string `json:"reviewerId,omitempty"`
	PackageHash  string `json:"packageHash,omitempty"`
	CiCheckRunID string `json:"ciCheckRunId,omitempty"`
}

// ReleaseAssetView is an asset entry in the release detail.
type ReleaseAssetView struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Label         string `json:"label,omitempty"`
	ContentType   string `json:"contentType"`
	Size          int64  `json:"size"`
	DownloadCount int64  `json:"downloadCount"`
}

// ReleaseDetailActions lists viewer-specific actions for release detail.
type ReleaseDetailActions struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
	CanYank   bool `json:"canYank"`
	CanUnYank bool `json:"canUnYank"`
}

// handleReleaseList returns the release list read model.
func handleReleaseList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	_ = r.PathValue("namespace")
	_ = r.PathValue("slug")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	actions := ReleaseListActions{
		CanCreateRelease: p.IsAuthenticated,
	}

	middleware.WriteJSON(w, http.StatusOK, ReleaseListReadModel{
		Releases:         []ReleaseListView{},
		TotalCount:       0,
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

// handleReleaseDetail returns the release detail read model.
func handleReleaseDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	_ = r.PathValue("namespace")
	_ = r.PathValue("slug")

	isOwnerOrAdmin := p.HasPlatformRole("SUPER_ADMIN")
	actions := ReleaseDetailActions{
		CanEdit:   isOwnerOrAdmin,
		CanDelete: isOwnerOrAdmin,
		CanYank:   isOwnerOrAdmin,
		CanUnYank: isOwnerOrAdmin,
	}

	middleware.WriteJSON(w, http.StatusOK, ReleaseDetailReadModel{
		Release:          ReleaseDetailView{},
		Assets:           []ReleaseAssetView{},
		AvailableActions: actions,
	})
}
