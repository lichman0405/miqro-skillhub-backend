package release

import (
	"context"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/skill"
)

// stubRepo is an in-memory ReleaseRepository for testing.
type stubRepo struct {
	releases map[int64]Release
	nextID   int64
}

func newStubRepo() *stubRepo {
	return &stubRepo{releases: make(map[int64]Release), nextID: 1}
}

func (r *stubRepo) Create(ctx context.Context, rel Release) (Release, error) {
	rel.ID = r.nextID
	r.nextID++
	r.releases[rel.ID] = rel
	return rel, nil
}

func (r *stubRepo) Update(ctx context.Context, rel Release) (Release, error) {
	r.releases[rel.ID] = rel
	return rel, nil
}

func (r *stubRepo) FindByID(ctx context.Context, id int64) (*Release, error) {
	rel, ok := r.releases[id]
	if !ok {
		return nil, nil
	}
	return &rel, nil
}

func (r *stubRepo) FindBySkillID(ctx context.Context, skillID int64) ([]Release, error) {
	var out []Release
	for _, rel := range r.releases {
		if rel.SkillID == skillID {
			out = append(out, rel)
		}
	}
	return out, nil
}

func (r *stubRepo) FindByVersionIDAndChannel(ctx context.Context, versionID int64, channel string) (*Release, error) {
	for _, rel := range r.releases {
		if rel.VersionID == versionID && rel.Channel == channel {
			return &rel, nil
		}
	}
	return nil, nil
}

func (r *stubRepo) FindLatestStable(ctx context.Context, skillID int64, channel string) (*Release, error) {
	var latest *Release
	var latestTime time.Time
	for _, rel := range r.releases {
		if rel.SkillID == skillID && rel.Channel == channel && !rel.Draft && !rel.Yanked {
			if latest == nil || (rel.PublishedAt != nil && rel.PublishedAt.After(latestTime)) {
				latest = &rel
				if rel.PublishedAt != nil {
					latestTime = *rel.PublishedAt
				}
			}
		}
	}
	return latest, nil
}

func (r *stubRepo) Delete(ctx context.Context, id int64) error {
	delete(r.releases, id)
	return nil
}

func (r *stubRepo) ListBySkillIDPaginated(ctx context.Context, skillID int64, offset int, limit int) ([]Release, error) {
	var all []Release
	for _, rel := range r.releases {
		if rel.SkillID == skillID {
			all = append(all, rel)
		}
	}
	// Sort by published_at desc — simple: use slice
	// For tests with small sets, just return what we have.
	if offset > len(all) {
		return make([]Release, 0), nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (r *stubRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	for _, rel := range r.releases {
		if rel.SkillID == skillID {
			count++
		}
	}
	return count, nil
}

// stubVersionRepo is an in-memory skill.SkillVersionRepository for testing.
type stubVersionRepo struct {
	versions map[int64]skill.SkillVersion
}

func newStubVersionRepo(versions ...skill.SkillVersion) *stubVersionRepo {
	m := make(map[int64]skill.SkillVersion)
	for _, v := range versions {
		m[v.ID] = v
	}
	return &stubVersionRepo{versions: m}
}

func pubVersion(id, skillID int64) skill.SkillVersion {
	return skill.SkillVersion{
		ID:      id,
		SkillID: skillID,
		Version: "1.0.0",
		Status:  "PUBLISHED",
	}
}

func (r *stubVersionRepo) FindByID(ctx context.Context, id int64) (*skill.SkillVersion, error) {
	if v, ok := r.versions[id]; ok {
		return &v, nil
	}
	return nil, nil
}
func (r *stubVersionRepo) FindByIDs(ctx context.Context, ids []int64) ([]skill.SkillVersion, error) { return nil, nil }
func (r *stubVersionRepo) FindBySkillID(ctx context.Context, skillID int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r *stubVersionRepo) FindBySkillIDAndVersion(ctx context.Context, skillID int64, version string) (*skill.SkillVersion, error) {
	return nil, nil
}
func (r *stubVersionRepo) FindBySkillIDAndStatus(ctx context.Context, skillID int64, status string) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r *stubVersionRepo) Save(ctx context.Context, v skill.SkillVersion) (skill.SkillVersion, error) {
	return v, nil
}
func (r *stubVersionRepo) Delete(ctx context.Context, id int64) error                  { return nil }
func (r *stubVersionRepo) DeleteBySkillID(ctx context.Context, skillID int64) error    { return nil }

// newTestService creates a Service with a stub repo and pre-seeded published
// versions for skill 1 (version IDs 10, 20, 30, 40, 50, 60).
func newTestService() *Service {
	var versions []skill.SkillVersion
	for _, vid := range []int64{10, 20, 30, 40, 50, 60} {
		versions = append(versions, pubVersion(vid, 1))
	}
	return NewService(newStubRepo(), nil, newStubVersionRepo(versions...))
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestCreateRelease_Success(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID:     1,
		VersionID:   10,
		Title:       "v1.0.0 Release",
		Notes:       "First stable release.",
		PublisherID: "u1",
	})
	if err != nil {
		t.Fatalf("CreateRelease: %v", err)
	}
	if r.ID != 1 {
		t.Errorf("expected ID 1, got %d", r.ID)
	}
	if r.Channel != "stable" {
		t.Errorf("expected channel 'stable', got %q", r.Channel)
	}
	if r.Draft {
		t.Error("expected non-draft release")
	}
	if r.Yanked {
		t.Error("expected non-yanked release")
	}
}

func TestCreateRelease_DuplicateChannelRejected(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Channel: "stable", Title: "First", PublisherID: "u1",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Channel: "stable", Title: "Second", PublisherID: "u1",
	})
	if err == nil {
		t.Fatal("expected error for duplicate version+channel")
	}
}

func TestCreateRelease_TitleRequired(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, PublisherID: "u1",
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestCreateRelease_Draft(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Draft Release", Draft: true, PublisherID: "u1",
	})
	if err != nil {
		t.Fatalf("CreateRelease draft: %v", err)
	}
	if !r.Draft {
		t.Error("expected draft=true")
	}
	if r.PublishedAt != nil {
		t.Error("draft should have nil PublishedAt")
	}
}

func TestCreateRelease_Prerelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Beta", Prerelease: true, PublisherID: "u1",
	})
	if err != nil {
		t.Fatalf("CreateRelease prerelease: %v", err)
	}
	if !r.Prerelease {
		t.Error("expected prerelease=true")
	}
}

func TestCreateRelease_Provenance(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	hash := "sha256:abc123"
	ciID := "check-run-42"
	reviewer := "rv1"

	r, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID:      1,
		VersionID:    10,
		Title:        "Provenance Release",
		PublisherID:  "u1",
		ReviewerID:   &reviewer,
		PackageHash:  &hash,
		CiCheckRunID: &ciID,
	})
	if err != nil {
		t.Fatalf("CreateRelease provenance: %v", err)
	}
	if r.PackageHash == nil || *r.PackageHash != hash {
		t.Errorf("expected PackageHash %q, got %v", hash, r.PackageHash)
	}
	if r.CiCheckRunID == nil || *r.CiCheckRunID != ciID {
		t.Errorf("expected CiCheckRunID %q, got %v", ciID, r.CiCheckRunID)
	}
	if r.ReviewerID == nil || *r.ReviewerID != reviewer {
		t.Errorf("expected ReviewerID %q, got %v", reviewer, r.ReviewerID)
	}
}

func TestGetRelease_NotFound(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.GetRelease(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent release")
	}
}

func TestListReleases(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Create 3 releases for skill 1.
	for i := 1; i <= 3; i++ {
		_, err := svc.CreateRelease(ctx, CreateReleaseInput{
			SkillID:     1,
			VersionID:   int64(i * 10),
			Title:       "Release " + string(rune('0'+i)),
			PublisherID: "u1",
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	result, err := svc.ListReleases(ctx, ListReleasesInput{SkillID: 1, Size: 10})
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(result.Releases) != 3 {
		t.Errorf("expected 3 releases, got %d", len(result.Releases))
	}
	if result.TotalCount != 3 {
		t.Errorf("expected totalCount=3, got %d", result.TotalCount)
	}
}

func TestListReleases_Pagination(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		_, _ = svc.CreateRelease(ctx, CreateReleaseInput{
			SkillID: 1, VersionID: int64(i * 10), Title: "R"+string(rune('0'+i)), PublisherID: "u1",
		})
	}

	result, err := svc.ListReleases(ctx, ListReleasesInput{SkillID: 1, Page: 0, Size: 2})
	if err != nil {
		t.Fatalf("ListReleases page0: %v", err)
	}
	if len(result.Releases) != 2 {
		t.Errorf("page 0: expected 2, got %d", len(result.Releases))
	}
	if result.TotalCount != 5 {
		t.Errorf("totalCount should be 5, got %d", result.TotalCount)
	}
}

func TestUpdateRelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "v1", PublisherID: "u1",
	})

	newTitle := "v1 Updated"
	notes := "Updated notes"
	updated, err := svc.UpdateRelease(ctx, UpdateReleaseInput{
		ID:    r.ID,
		Title: &newTitle,
		Notes: &notes,
	})
	if err != nil {
		t.Fatalf("UpdateRelease: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, updated.Title)
	}
	if updated.Notes != notes {
		t.Errorf("expected notes %q, got %q", notes, updated.Notes)
	}
}

func TestUpdateRelease_NotFound(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	title := "x"
	_, err := svc.UpdateRelease(ctx, UpdateReleaseInput{ID: 999, Title: &title})
	if err == nil {
		t.Fatal("expected error for non-existent release")
	}
}

func TestPublishRelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Draft Release", Draft: true, PublisherID: "u1",
	})
	if r.PublishedAt != nil {
		t.Error("draft should have nil PublishedAt")
	}

	pub, err := svc.PublishRelease(ctx, r.ID)
	if err != nil {
		t.Fatalf("PublishRelease: %v", err)
	}
	if pub.Draft {
		t.Error("expected draft=false after publish")
	}
	if pub.PublishedAt == nil {
		t.Error("published should have non-nil PublishedAt")
	}
}

func TestYankRelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "v1", PublisherID: "u1",
	})
	if r.Yanked {
		t.Error("new release should not be yanked")
	}

	yanked, err := svc.YankRelease(ctx, r.ID)
	if err != nil {
		t.Fatalf("YankRelease: %v", err)
	}
	if !yanked.Yanked {
		t.Error("expected yanked=true")
	}

	// Yanked releases should not appear in latest stable query.
	latest, err := svc.GetLatestRelease(ctx, 1, "stable")
	if err == nil {
		t.Errorf("expected no stable release after yank, got %+v", latest)
	}
}

func TestUnyankRelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "v1", PublisherID: "u1",
	})
	_, _ = svc.YankRelease(ctx, r.ID)

	unyanked, err := svc.UnyankRelease(ctx, r.ID)
	if err != nil {
		t.Fatalf("UnyankRelease: %v", err)
	}
	if unyanked.Yanked {
		t.Error("expected yanked=false after unyank")
	}

	// Should now appear in latest stable.
	latest, err := svc.GetLatestRelease(ctx, 1, "stable")
	if err != nil {
		t.Fatalf("GetLatestRelease after unyank: %v", err)
	}
	if latest == nil {
		t.Fatal("expected stable release after unyank")
	}
	if latest.ID != r.ID {
		t.Errorf("expected release %d, got %d", r.ID, latest.ID)
	}
}

func TestYankReflectedInLatestStable(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Create two releases, yank the first.
	r1, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "v1.0", PublisherID: "u1",
	})
	r2, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 20, Title: "v2.0", PublisherID: "u1",
	})

	_, _ = svc.YankRelease(ctx, r1.ID)

	// Latest stable should be r2.
	latest, err := svc.GetLatestRelease(ctx, 1, "stable")
	if err != nil {
		t.Fatalf("GetLatestRelease: %v", err)
	}
	if latest.ID != r2.ID {
		t.Errorf("expected latest release %d (v2.0), got %d", r2.ID, latest.ID)
	}
}

func TestGetLatestRelease_WrongChannel(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, _ = svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Channel: "stable", Title: "Stable", PublisherID: "u1",
	})

	_, err := svc.GetLatestRelease(ctx, 1, "beta")
	if err == nil {
		t.Fatal("expected error for missing beta channel release")
	}
}

func TestDeleteRelease(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "To Delete", PublisherID: "u1",
	})
	if err := svc.DeleteRelease(ctx, r.ID); err != nil {
		t.Fatalf("DeleteRelease: %v", err)
	}

	_, err := svc.GetRelease(ctx, r.ID)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

// TestReleaseArtifactImmutability verifies that after creation, core identity
// fields (SkillID, VersionID, Channel, PublisherID) are immutable — only
// metadata (title, notes, draft, prerelease, yanked) can change.
func TestReleaseArtifactImmutability(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	r, _ := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Immutable", PublisherID: "u1",
	})

	// get snapshot of immutable fields
	origSkillID := r.SkillID
	origVersionID := r.VersionID
	origChannel := r.Channel
	origPublisherID := r.PublisherID

	// Update metadata — this should work.
	newTitle := "Updated"
	updated, err := svc.UpdateRelease(ctx, UpdateReleaseInput{ID: r.ID, Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateRelease: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("title should update")
	}

	// Verify immutable fields unchanged.
	refetched, _ := svc.GetRelease(ctx, r.ID)
	if refetched.SkillID != origSkillID {
		t.Errorf("SkillID should be immutable")
	}
	if refetched.VersionID != origVersionID {
		t.Errorf("VersionID should be immutable")
	}
	if refetched.Channel != origChannel {
		t.Errorf("Channel should be immutable")
	}
	if refetched.PublisherID != origPublisherID {
		t.Errorf("PublisherID should be immutable")
	}
}

// ── Version validation tests ────────────────────────────────────────────────

func TestCreateRelease_VersionFromDifferentSkill_Rejected(t *testing.T) {
	// version 10 belongs to skill 2, but we claim skill 1.
	v := pubVersion(10, 2)
	svc := NewService(newStubRepo(), nil, newStubVersionRepo(v))
	ctx := context.Background()

	_, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Mismatch", PublisherID: "u1",
	})
	if err == nil {
		t.Fatal("expected error for version from different skill")
	}
}

func TestCreateRelease_NonPublishedVersion_Rejected(t *testing.T) {
	v := skill.SkillVersion{
		ID: 10, SkillID: 1, Version: "1.0.0", Status: "DRAFT",
	}
	svc := NewService(newStubRepo(), nil, newStubVersionRepo(v))
	ctx := context.Background()

	_, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Draft Version", PublisherID: "u1",
	})
	if err == nil {
		t.Fatal("expected error for non-published version")
	}
}

func TestCreateRelease_YankedVersion_Rejected(t *testing.T) {
	yankedAt := time.Now()
	v := skill.SkillVersion{
		ID: 10, SkillID: 1, Version: "1.0.0", Status: "PUBLISHED", YankedAt: &yankedAt,
	}
	svc := NewService(newStubRepo(), nil, newStubVersionRepo(v))
	ctx := context.Background()

	_, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Yanked Version", PublisherID: "u1",
	})
	if err == nil {
		t.Fatal("expected error for yanked version")
	}
}

func TestCreateRelease_PublishedVersionOwnedBySameSkill_Accepted(t *testing.T) {
	v := pubVersion(10, 1)
	svc := NewService(newStubRepo(), nil, newStubVersionRepo(v))
	ctx := context.Background()

	r, err := svc.CreateRelease(ctx, CreateReleaseInput{
		SkillID: 1, VersionID: 10, Title: "Good Release", PublisherID: "u1",
	})
	if err != nil {
		t.Fatalf("expected success for published version owned by same skill: %v", err)
	}
	if r.SkillID != 1 || r.VersionID != 10 {
		t.Errorf("unexpected release identity: skill=%d version=%d", r.SkillID, r.VersionID)
	}
}
