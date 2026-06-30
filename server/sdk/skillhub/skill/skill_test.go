package skill_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockSkillRepo struct {
	skills  map[int64]skill.Skill
	nextID  int64
	downloads map[int64]int64
}

func newMockSkillRepo() *mockSkillRepo {
	return &mockSkillRepo{
		skills:    make(map[int64]skill.Skill),
		nextID:    1,
		downloads: make(map[int64]int64),
	}
}

func (m *mockSkillRepo) Save(_ context.Context, s skill.Skill) (skill.Skill, error) {
	if s.ID == 0 {
		s.ID = m.nextID
		m.nextID++
	}
	m.skills[s.ID] = s
	return s, nil
}
func (m *mockSkillRepo) FindByID(_ context.Context, id int64) (*skill.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}
func (m *mockSkillRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, id := range ids {
		if s, ok := m.skills[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		out = append(out, s)
	}
	return out, nil
}
func (m *mockSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, slug string) ([]skill.Skill, error) {
	return m.FindByNamespaceIDAndSlug(context.Background(), 0, slug)
}
func (m *mockSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, nsID int64, slug, ownerID string) (*skill.Skill, error) {
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug && s.OwnerID == ownerID {
			return &s, nil
		}
	}
	return nil, nil
}
func (m *mockSkillRepo) FindByOwnerID(_ context.Context, ownerID string) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		if s.OwnerID == ownerID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockSkillRepo) FindBySlug(_ context.Context, slug string) ([]skill.Skill, error) {
	return m.FindByNamespaceIDAndSlug(context.Background(), 0, slug)
}
func (m *mockSkillRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error) {
	for _, s := range m.skills {
		if s.NamespaceID == nsID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockSkillRepo) Delete(_ context.Context, id int64) error {
	delete(m.skills, id)
	return nil
}
func (m *mockSkillRepo) IncrementDownloadCount(_ context.Context, id int64) error {
	m.downloads[id]++
	return nil
}
func (m *mockSkillRepo) IncrementSubscriptionCount(_ context.Context, id int64) error { return nil }
func (m *mockSkillRepo) DecrementSubscriptionCount(_ context.Context, id int64) error { return nil }

type mockVersionRepo struct {
	versions map[int64]skill.SkillVersion
	nextID   int64
}

func newMockVersionRepo() *mockVersionRepo {
	return &mockVersionRepo{versions: make(map[int64]skill.SkillVersion), nextID: 1}
}
func (m *mockVersionRepo) Save(_ context.Context, v skill.SkillVersion) (skill.SkillVersion, error) {
	if v.ID == 0 {
		v.ID = m.nextID
		m.nextID++
	}
	m.versions[v.ID] = v
	return v, nil
}
func (m *mockVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *mockVersionRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, id := range ids {
		if v, ok := m.versions[id]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *mockVersionRepo) FindBySkillID(_ context.Context, skillID int64) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *mockVersionRepo) FindBySkillIDAndVersion(_ context.Context, skillID int64, version string) (*skill.SkillVersion, error) {
	for _, v := range m.versions {
		if v.SkillID == skillID && v.Version == version {
			return &v, nil
		}
	}
	return nil, nil
}
func (m *mockVersionRepo) FindBySkillIDAndStatus(_ context.Context, skillID int64, status string) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID && v.Status == status {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *mockVersionRepo) Delete(_ context.Context, id int64) error {
	delete(m.versions, id)
	return nil
}
func (m *mockVersionRepo) DeleteBySkillID(_ context.Context, skillID int64) error {
	for id, v := range m.versions {
		if v.SkillID == skillID {
			delete(m.versions, id)
		}
	}
	return nil
}

type mockFileRepo struct {
	files  map[int64]skill.SkillFile
	nextID int64
}

func newMockFileRepo() *mockFileRepo {
	return &mockFileRepo{files: make(map[int64]skill.SkillFile), nextID: 1}
}
func (m *mockFileRepo) FindByVersionID(_ context.Context, versionID int64) ([]skill.SkillFile, error) {
	var out []skill.SkillFile
	for _, f := range m.files {
		if f.VersionID == versionID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *mockFileRepo) Save(_ context.Context, f skill.SkillFile) (skill.SkillFile, error) {
	f.ID = m.nextID
	m.nextID++
	m.files[f.ID] = f
	return f, nil
}
func (m *mockFileRepo) SaveAll(_ context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	var saved []skill.SkillFile
	for _, f := range files {
		sf, _ := m.Save(context.Background(), f)
		saved = append(saved, sf)
	}
	return saved, nil
}
func (m *mockFileRepo) DeleteByVersionID(_ context.Context, versionID int64) error {
	for id, f := range m.files {
		if f.VersionID == versionID {
			delete(m.files, id)
		}
	}
	return nil
}

type mockTagRepo struct {
	tags   map[int64]skill.SkillTag
	nextID int64
}

func newMockTagRepo() *mockTagRepo {
	return &mockTagRepo{tags: make(map[int64]skill.SkillTag), nextID: 1}
}
func (m *mockTagRepo) FindBySkillIDAndTagName(_ context.Context, skillID int64, tagName string) (*skill.SkillTag, error) {
	for _, t := range m.tags {
		if t.SkillID == skillID && t.TagName == tagName {
			return &t, nil
		}
	}
	return nil, nil
}
func (m *mockTagRepo) FindBySkillID(_ context.Context, skillID int64) ([]skill.SkillTag, error) {
	var out []skill.SkillTag
	for _, t := range m.tags {
		if t.SkillID == skillID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockTagRepo) Save(_ context.Context, tag skill.SkillTag) (skill.SkillTag, error) {
	if tag.ID == 0 {
		tag.ID = m.nextID
		m.nextID++
	}
	m.tags[tag.ID] = tag
	return tag, nil
}
func (m *mockTagRepo) Delete(_ context.Context, id int64) error {
	delete(m.tags, id)
	return nil
}
func (m *mockTagRepo) DeleteBySkillID(_ context.Context, skillID int64) error {
	for id, t := range m.tags {
		if t.SkillID == skillID {
			delete(m.tags, id)
		}
	}
	return nil
}

// ============================================================================
// Mock namespace repo
// ============================================================================

type mockNsRepo struct {
	ns map[int64]namespace.Namespace
}

func newMockNsRepo(nsList ...namespace.Namespace) *mockNsRepo {
	m := &mockNsRepo{ns: make(map[int64]namespace.Namespace)}
	for _, n := range nsList {
		m.ns[n.ID] = n
	}
	return m
}
func (m *mockNsRepo) FindByID(_ context.Context, id int64) (*namespace.Namespace, error) {
	ns, ok := m.ns[id]
	if !ok {
		return nil, nil
	}
	return &ns, nil
}
func (m *mockNsRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error) {
	var out []namespace.Namespace
	for _, id := range ids {
		if ns, ok := m.ns[id]; ok {
			out = append(out, ns)
		}
	}
	return out, nil
}
func (m *mockNsRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	for _, ns := range m.ns {
		if ns.Slug == slug {
			return &ns, nil
		}
	}
	return nil, nil
}
func (m *mockNsRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) {
	var out []namespace.Namespace
	for _, ns := range m.ns {
		if ns.Status == status {
			out = append(out, ns)
		}
	}
	return out, nil
}
func (m *mockNsRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	m.ns[ns.ID] = ns
	return ns, nil
}
func (m *mockNsRepo) Delete(_ context.Context, id int64) error {
	delete(m.ns, id)
	return nil
}

// ============================================================================
// Mock member repo
// ============================================================================

type mockNsMemberRepo struct {
	members map[int64]namespace.NamespaceMember
	nextID  int64
}

func newMockNsMemberRepo() *mockNsMemberRepo {
	return &mockNsMemberRepo{members: make(map[int64]namespace.NamespaceMember), nextID: 1}
}
func (m *mockNsMemberRepo) Save(_ context.Context, mb namespace.NamespaceMember) (namespace.NamespaceMember, error) {
	if mb.ID == 0 {
		mb.ID = m.nextID
		m.nextID++
	}
	m.members[mb.ID] = mb
	return mb, nil
}
func (m *mockNsMemberRepo) FindByNamespaceAndUser(_ context.Context, nsID int64, userID string) (*namespace.NamespaceMember, error) {
	for _, mb := range m.members {
		if mb.NamespaceID == nsID && mb.UserID == userID {
			return &mb, nil
		}
	}
	return nil, nil
}
func (m *mockNsMemberRepo) FindByUserID(_ context.Context, userID string) ([]namespace.NamespaceMember, error) {
	var out []namespace.NamespaceMember
	for _, mb := range m.members {
		if mb.UserID == userID {
			out = append(out, mb)
		}
	}
	return out, nil
}
func (m *mockNsMemberRepo) FindByNamespaceID(_ context.Context, nsID int64) ([]namespace.NamespaceMember, error) {
	var out []namespace.NamespaceMember
	for _, mb := range m.members {
		if mb.NamespaceID == nsID {
			out = append(out, mb)
		}
	}
	return out, nil
}
func (m *mockNsMemberRepo) FindByNamespaceIDAndRoles(_ context.Context, nsID int64, roles []string) ([]namespace.NamespaceMember, error) { return nil, nil }
func (m *mockNsMemberRepo) DeleteByNamespaceAndUser(_ context.Context, nsID int64, userID string) error { return nil }
func (m *mockNsMemberRepo) DeleteByNamespaceID(_ context.Context, nsID int64) error { return nil }

// ============================================================================
// Mock object store (in-memory)
// ============================================================================

type mockStore struct {
	objects map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{objects: make(map[string][]byte)}
}

func (s *mockStore) PutObject(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(data); err != nil {
		return err
	}
	s.objects[key] = buf.Bytes()
	return nil
}
func (s *mockStore) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
func (s *mockStore) DeleteObject(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}
func (s *mockStore) DeleteObjects(_ context.Context, keys []string) error {
	for _, k := range keys {
		delete(s.objects, k)
	}
	return nil
}
func (s *mockStore) Exists(_ context.Context, key string) (bool, error) {
	_, ok := s.objects[key]
	return ok, nil
}
func (s *mockStore) Metadata(_ context.Context, key string) (storage.ObjectMetadata, error) {
	data, ok := s.objects[key]
	if !ok {
		return storage.ObjectMetadata{}, errors.New("not found")
	}
	return storage.ObjectMetadata{ContentLength: int64(len(data)), ContentType: "application/octet-stream"}, nil
}
func (s *mockStore) PresignedURL(_ context.Context, key string, _ time.Duration, _ string) (string, error) {
	return "", errors.New("not supported")
}

// ============================================================================
// Visibility tests
// ============================================================================

func TestVisibility_PublicVisibleToAll(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PUBLIC",
		Status:          "ACTIVE",
		LatestVersionID: ptrInt64(1),
	}
	if !vc.CanAccess(sk, "anyone", nil, nil) {
		t.Error("PUBLIC skill should be visible to anyone")
	}
	if !vc.CanAccess(sk, "", nil, nil) {
		t.Error("PUBLIC skill should be visible to anonymous")
	}
}

func TestVisibility_PrivateOwnerOnly(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PRIVATE",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		LatestVersionID: ptrInt64(1),
	}
	if vc.CanAccess(sk, "stranger", nil, nil) {
		t.Error("PRIVATE skill should NOT be visible to strangers")
	}
	if !vc.CanAccess(sk, "owner-1", nil, nil) {
		t.Error("PRIVATE skill should be visible to owner")
	}
}

func TestVisibility_PrivateAdminCanAccess(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PRIVATE",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		LatestVersionID: ptrInt64(1),
	}
	roles := map[int64]string{1: "ADMIN"}
	if !vc.CanAccess(sk, "admin-user", roles, nil) {
		t.Error("ADMIN should see PRIVATE skill in their namespace")
	}
}

func TestVisibility_NamespaceOnly(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "NAMESPACE_ONLY",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		LatestVersionID: ptrInt64(1),
	}
	if vc.CanAccess(sk, "stranger", nil, nil) {
		t.Error("NAMESPACE_ONLY should NOT be visible to non-members")
	}
	roles := map[int64]string{1: "MEMBER"}
	if !vc.CanAccess(sk, "member-1", roles, nil) {
		t.Error("NAMESPACE_ONLY should be visible to namespace members")
	}
}

func TestVisibility_SuperAdminSeesAll(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PRIVATE",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		LatestVersionID: ptrInt64(1),
	}
	platformRoles := map[string]bool{"SUPER_ADMIN": true}
	if !vc.CanAccess(sk, "admin", nil, platformRoles) {
		t.Error("SUPER_ADMIN should see everything")
	}
}

func TestVisibility_HiddenSkill(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PUBLIC",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		Hidden:          true,
		LatestVersionID: ptrInt64(1),
	}
	if vc.CanAccess(sk, "stranger", nil, nil) {
		t.Error("hidden PUBLIC skill should NOT be visible to strangers")
	}
	if !vc.CanAccess(sk, "owner-1", nil, nil) {
		t.Error("hidden skill should be visible to owner")
	}
}

func TestVisibility_NoLatestVersion(t *testing.T) {
	vc := skill.NewVisibilityChecker()
	sk := skill.Skill{
		NamespaceID:     1,
		Visibility:      "PUBLIC",
		Status:          "ACTIVE",
		OwnerID:         "owner-1",
		LatestVersionID: nil, // no published version
	}
	if vc.CanAccess(sk, "stranger", nil, nil) {
		t.Error("skill with no latest version should NOT be visible to strangers")
	}
	if !vc.CanAccess(sk, "owner-1", nil, nil) {
		t.Error("owner should still see their own unpublished skill")
	}
}

// ============================================================================
// Installability tests
// ============================================================================

func TestInstallability_PublishedDownloadReady(t *testing.T) {
	v := skill.SkillVersion{
		Status:        "PUBLISHED",
		DownloadReady: true,
	}
	if !skill.IsInstallable(v) {
		t.Error("published, download-ready version should be installable")
	}
}

func TestInstallability_NotPublished(t *testing.T) {
	v := skill.SkillVersion{Status: "UPLOADED", DownloadReady: true}
	if skill.IsInstallable(v) {
		t.Error("non-published version should NOT be installable")
	}
}

func TestInstallability_Yanked(t *testing.T) {
	now := time.Now()
	v := skill.SkillVersion{
		Status:        "PUBLISHED",
		DownloadReady: true,
		YankedAt:      &now,
	}
	if skill.IsInstallable(v) {
		t.Error("yanked version should NOT be installable")
	}
}

// ============================================================================
// Tag service tests
// ============================================================================

// setupTagService creates a fully-wired SkillTagService with a mock skill
// owned by "owner-1" in namespace 1, plus mock version and tag repos.
func setupTagService() (*mockSkillRepo, *mockVersionRepo, *mockTagRepo, *skill.SkillTagService) {
	skillRepo := newMockSkillRepo()
	versionRepo := newMockVersionRepo()
	tagRepo := newMockTagRepo()

	// Create a skill owned by "owner-1" in namespace 1.
	skillRepo.Save(context.Background(), skill.Skill{
		ID:          1,
		NamespaceID: 1,
		OwnerID:     "owner-1",
		Slug:        "test-skill",
		Visibility:  "PUBLIC",
		Status:      "ACTIVE",
	})

	svc := skill.NewSkillTagService(tagRepo, versionRepo, skillRepo)
	return skillRepo, versionRepo, tagRepo, svc
}

// ownerRoles returns userNsRoles granting OWNER in namespace 1.
func ownerRoles() map[int64]string { return map[int64]string{1: "OWNER"} }

// adminRoles returns userNsRoles granting ADMIN in namespace 1.
func adminRoles() map[int64]string { return map[int64]string{1: "ADMIN"} }

// memberRoles returns userNsRoles granting MEMBER in namespace 1.
func memberRoles() map[int64]string { return map[int64]string{1: "MEMBER"} }

func TestTagService_CreateAndList_Success_Owner(t *testing.T) {
	_, versionRepo, tagRepo, svc := setupTagService()
	version, _ := versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	tag, err := svc.CreateTag(context.Background(), 1, "stable", version.Version, "owner-1", ownerRoles())
	if err != nil {
		t.Fatalf("CreateTag (owner) failed: %v", err)
	}
	if tag.TagName != "stable" {
		t.Errorf("expected tag 'stable', got '%s'", tag.TagName)
	}

	tags, err := svc.ListTags(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}

	// Ensure the tag was stored.
	stored, _ := tagRepo.FindBySkillIDAndTagName(context.Background(), 1, "stable")
	if stored == nil || stored.VersionID != version.ID {
		t.Error("tag not found or wrong version in repo")
	}
}

func TestTagService_CreateTag_Success_NamespaceAdmin(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "2.0.0", Status: "PUBLISHED",
	})

	// Namespace ADMIN (not the skill owner) should also be able to create a tag.
	tag, err := svc.CreateTag(context.Background(), 1, "admin-tag", "2.0.0", "admin-user", adminRoles())
	if err != nil {
		t.Fatalf("CreateTag (admin) failed: %v", err)
	}
	if tag.TagName != "admin-tag" {
		t.Errorf("expected tag 'admin-tag', got '%s'", tag.TagName)
	}
}

func TestTagService_CreateTag_Unauthorized_Stranger(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	// A stranger (no role in namespace) cannot create a tag.
	_, err := svc.CreateTag(context.Background(), 1, "bad-tag", "1.0.0", "stranger", nil)
	if err == nil {
		t.Fatal("expected access.denied for stranger")
	}
	if !stringsContain(err.Error(), "access.denied") {
		t.Errorf("expected 'access.denied', got: %v", err)
	}
}

func TestTagService_CreateTag_Unauthorized_Member(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	// A plain MEMBER cannot create a tag.
	_, err := svc.CreateTag(context.Background(), 1, "bad-tag", "1.0.0", "member-user", memberRoles())
	if err == nil {
		t.Fatal("expected access.denied for plain MEMBER")
	}
}

func TestTagService_CreateTag_SkillNotFound(t *testing.T) {
	skillRepo := newMockSkillRepo()
	versionRepo := newMockVersionRepo()
	tagRepo := newMockTagRepo()
	svc := skill.NewSkillTagService(tagRepo, versionRepo, skillRepo)

	_, err := svc.CreateTag(context.Background(), 999, "tag", "1.0.0", "owner-1", ownerRoles())
	if err == nil {
		t.Fatal("expected error for non-existent skill")
	}
	if !stringsContain(err.Error(), "notFound") {
		t.Errorf("expected 'notFound', got: %v", err)
	}
}

func TestTagService_CreateTag_OnlyPublishedVersions(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "UPLOADED",
	})

	_, err := svc.CreateTag(context.Background(), 1, "v1", "1.0.0", "owner-1", ownerRoles())
	if err == nil {
		t.Fatal("expected error tagging non-published version")
	}
}

func TestTagService_DeleteTag_Success_Owner(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	// Create a tag first.
	svc.CreateTag(context.Background(), 1, "v1", "1.0.0", "owner-1", ownerRoles())

	// Owner can delete it.
	if err := svc.DeleteTag(context.Background(), 1, "v1", "owner-1", ownerRoles()); err != nil {
		t.Fatalf("DeleteTag (owner) failed: %v", err)
	}

	tags, _ := svc.ListTags(context.Background(), 1)
	if len(tags) != 0 {
		t.Fatal("expected 0 tags after delete")
	}
}

func TestTagService_DeleteTag_Success_NamespaceAdmin(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	// Owner creates a tag.
	svc.CreateTag(context.Background(), 1, "admin-delete", "1.0.0", "owner-1", ownerRoles())

	// Namespace ADMIN (not the owner) can delete it.
	if err := svc.DeleteTag(context.Background(), 1, "admin-delete", "admin-user", adminRoles()); err != nil {
		t.Fatalf("DeleteTag (admin) failed: %v", err)
	}
}

func TestTagService_DeleteTag_Unauthorized_Stranger(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	svc.CreateTag(context.Background(), 1, "v1", "1.0.0", "owner-1", ownerRoles())

	// A stranger cannot delete it.
	err := svc.DeleteTag(context.Background(), 1, "v1", "stranger", nil)
	if err == nil {
		t.Fatal("expected access.denied for stranger")
	}
	if !stringsContain(err.Error(), "access.denied") {
		t.Errorf("expected 'access.denied', got: %v", err)
	}
}

func TestTagService_DeleteTag_Unauthorized_Member(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})

	svc.CreateTag(context.Background(), 1, "v1", "1.0.0", "owner-1", ownerRoles())

	// A plain MEMBER cannot delete it.
	err := svc.DeleteTag(context.Background(), 1, "v1", "member-user", memberRoles())
	if err == nil {
		t.Fatal("expected access.denied for plain MEMBER")
	}
}

func TestTagService_DeleteTag_NotFound(t *testing.T) {
	_, _, _, svc := setupTagService()

	err := svc.DeleteTag(context.Background(), 1, "nonexistent", "owner-1", ownerRoles())
	if err == nil {
		t.Fatal("expected error for non-existent tag")
	}
	if !stringsContain(err.Error(), "notFound") {
		t.Errorf("expected 'notFound', got: %v", err)
	}
}

func TestTagService_DeleteTag_SkillNotFound(t *testing.T) {
	skillRepo := newMockSkillRepo()
	versionRepo := newMockVersionRepo()
	tagRepo := newMockTagRepo()
	svc := skill.NewSkillTagService(tagRepo, versionRepo, skillRepo)

	err := svc.DeleteTag(context.Background(), 999, "tag", "owner-1", ownerRoles())
	if err == nil {
		t.Fatal("expected error for non-existent skill")
	}
}

func TestTagService_UpsertExistingTag(t *testing.T) {
	_, versionRepo, _, svc := setupTagService()
	v1, _ := versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "1.0.0", Status: "PUBLISHED",
	})
	v2, _ := versionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 1, Version: "2.0.0", Status: "PUBLISHED",
	})

	// Create first tag.
	tag, _ := svc.CreateTag(context.Background(), 1, "movable", v1.Version, "owner-1", ownerRoles())
	if tag.VersionID != v1.ID {
		t.Fatalf("expected v1, got %d", tag.VersionID)
	}

	// Move the tag to v2.
	tag2, err := svc.CreateTag(context.Background(), 1, "movable", v2.Version, "owner-1", ownerRoles())
	if err != nil {
		t.Fatalf("upsert tag failed: %v", err)
	}
	if tag2.VersionID != v2.ID {
		t.Errorf("expected v2 after move, got %d", tag2.VersionID)
	}
}

// ============================================================================
// Helpers
// ============================================================================

func ptrInt64(v int64) *int64 { return &v }

func stringsContain(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOfSub(s, sub) >= 0)
}

func indexOfSub(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var _ = packagekit.SkillMetadata{}
var _ = namespace.Namespace{}
