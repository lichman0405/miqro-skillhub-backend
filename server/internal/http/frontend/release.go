package frontend

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	sdkerrors "miqro-skillhub/server/sdk/skillhub/errors"
	"miqro-skillhub/server/sdk/skillhub/release"
)

// ReleaseListView is a release entry in the release list.
type ReleaseListView struct {
	ID          int64  `json:"id"`
	VersionID   int64  `json:"versionId"`
	Channel     string `json:"channel"`
	Title       string `json:"title"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	Yanked      bool   `json:"yanked"`
	PublishedAt string `json:"publishedAt,omitempty"`
	PublisherID string `json:"publisherId"`
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
	Assets           []ReleaseAssetView   `json:"assets"`
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
// Uses portal handlers to fetch real release data when available.
func handleReleaseList(w http.ResponseWriter, r *http.Request, skillH *portal.SkillHandler, releaseH *portal.ReleaseHandler) {
	p := middleware.GetPrincipal(r)

	page := parsePositiveInt(r.URL.Query().Get("page"), 0)
	size := parsePositiveInt(r.URL.Query().Get("size"), 20)
	if size > 100 {
		size = 100
	}

	actions := ReleaseListActions{
		CanCreateRelease: p.IsAuthenticated,
	}

	releases := make([]ReleaseListView, 0)
	totalCount := int64(0)

	// Resolve skill and list releases when services are available.
	// Skip if skillH is nil to avoid an error response from resolveFrontendSkill.
	if skillH != nil && releaseH != nil && releaseH.ReleaseSvc != nil {
		if sk, ok := resolveFrontendSkill(w, r, skillH); ok {
			result, err := releaseH.ReleaseSvc.ListReleases(r.Context(), release.ListReleasesInput{
				SkillID: sk.ID,
				Page:    page,
				Size:    size,
			})
			if err != nil {
				middleware.WriteError(w, err)
				return
			}
			if result != nil {
				totalCount = result.TotalCount
				for _, rel := range result.Releases {
					pub := ""
					if rel.PublishedAt != nil {
						pub = rel.PublishedAt.Format("2006-01-02T15:04:05Z")
					}
					releases = append(releases, ReleaseListView{
						ID: rel.ID, VersionID: rel.VersionID, Channel: rel.Channel,
						Title: rel.Title, Draft: rel.Draft, Prerelease: rel.Prerelease,
						Yanked: rel.Yanked, PublishedAt: pub, PublisherID: rel.PublisherID,
					})
				}
			}
		}
	}

	middleware.WriteJSON(w, http.StatusOK, ReleaseListReadModel{
		Releases:         releases,
		TotalCount:       totalCount,
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

// handleReleaseDetail returns the release detail read model.
// Uses portal handlers to fetch real release data when available.
func handleReleaseDetail(w http.ResponseWriter, r *http.Request, skillH *portal.SkillHandler, releaseH *portal.ReleaseHandler) {
	p := middleware.GetPrincipal(r)

	releaseID, err := strconv.ParseInt(pathValueOrSegment(r.URL.Path, r.PathValue("releaseID"), 1), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release ID"})
		return
	}

	isSuperAdmin := p.HasPlatformRole("SUPER_ADMIN")
	actions := ReleaseDetailActions{
		CanEdit:   isSuperAdmin,
		CanDelete: isSuperAdmin,
		CanYank:   isSuperAdmin,
		CanUnYank: isSuperAdmin,
	}

	relView := ReleaseDetailView{}
	assetViews := make([]ReleaseAssetView, 0)

	if skillH != nil && releaseH != nil && releaseH.ReleaseSvc != nil {
		sk, ok := resolveFrontendSkill(w, r, skillH)
		if ok {
			rel, err := releaseH.ReleaseSvc.GetRelease(r.Context(), releaseID)
			if err != nil {
				middleware.WriteError(w, err)
				return
			}
			if rel == nil || rel.SkillID != sk.ID {
				middleware.WriteError(w, sdkerrors.NotFound("release.not_found"))
				return
			}
			isPublisher := p.UserID == rel.PublisherID
			canEdit := isPublisher || isSuperAdmin
			actions = ReleaseDetailActions{
				CanEdit: canEdit, CanDelete: canEdit,
				CanYank: canEdit, CanUnYank: canEdit,
			}

			pub := ""
			if rel.PublishedAt != nil {
				pub = rel.PublishedAt.Format("2006-01-02T15:04:05Z")
			}
			reviewer := ""
			if rel.ReviewerID != nil {
				reviewer = *rel.ReviewerID
			}
			hash := ""
			if rel.PackageHash != nil {
				hash = *rel.PackageHash
			}
			ci := ""
			if rel.CiCheckRunID != nil {
				ci = *rel.CiCheckRunID
			}
			relView = ReleaseDetailView{
				ID: rel.ID, SkillID: rel.SkillID, VersionID: rel.VersionID,
				Channel: rel.Channel, Title: rel.Title, Notes: rel.Notes,
				Draft: rel.Draft, Prerelease: rel.Prerelease, Yanked: rel.Yanked,
				PublishedAt: pub, PublisherID: rel.PublisherID,
				ReviewerID: reviewer, PackageHash: hash, CiCheckRunID: ci,
			}

			assets, err := releaseH.ReleaseSvc.ListAssets(r.Context(), releaseID)
			if err != nil {
				middleware.WriteError(w, err)
				return
			}
			if assets != nil {
				for _, a := range assets {
					assetViews = append(assetViews, ReleaseAssetView{
						ID: a.ID, Name: a.Name, Label: ptrToStr(a.Label),
						ContentType: a.ContentType, Size: a.Size,
						DownloadCount: a.DownloadCount,
					})
				}
			}
		}
	}

	middleware.WriteJSON(w, http.StatusOK, ReleaseDetailReadModel{
		Release:          relView,
		Assets:           assetViews,
		AvailableActions: actions,
	})
}
