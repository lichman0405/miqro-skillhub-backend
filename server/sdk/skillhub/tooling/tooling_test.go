package tooling

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

func TestBuildManifest_Deterministic(t *testing.T) {
	entries1 := []packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte("---\nname: test\n---\nbody"), Size: 30, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Test"), Size: 7, ContentType: "text/markdown"},
		{Path: "scripts/run.sh", Content: []byte("#!/bin/sh\necho hi"), Size: 18, ContentType: "text/x-sh"},
	}
	// Same entries but different slice order.
	entries2 := []packagekit.PackageEntry{
		{Path: "scripts/run.sh", Content: []byte("#!/bin/sh\necho hi"), Size: 18, ContentType: "text/x-sh"},
		{Path: "SKILL.md", Content: []byte("---\nname: test\n---\nbody"), Size: 30, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Test"), Size: 7, ContentType: "text/markdown"},
	}

	m1 := BuildManifest(entries1)
	m2 := BuildManifest(entries2)

	if m1.Hash != m2.Hash {
		t.Errorf("manifest hash should be deterministic regardless of input order:\n  m1: %s\n  m2: %s", m1.Hash, m2.Hash)
	}
	if m1.FileCount != 3 {
		t.Errorf("expected 3 files, got %d", m1.FileCount)
	}
	if m1.TotalSize != 55 {
		t.Errorf("expected total size 55, got %d", m1.TotalSize)
	}
	if m1.Entries[0].Path != "README.md" {
		t.Errorf("entries should be sorted by path, got first: %s", m1.Entries[0].Path)
	}
}

func TestBuildManifest_IncludesSHA256(t *testing.T) {
	entries := []packagekit.PackageEntry{
		{Path: "a.txt", Content: []byte("hello"), Size: 5, ContentType: "text/plain"},
		{Path: "b.txt", Content: []byte("world"), Size: 5, ContentType: "text/plain"},
	}

	m := BuildManifest(entries)

	if len(m.Entries) != 2 {
		t.Fatal("expected 2 entries")
	}
	// SHA-256 of "hello" is 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	if m.Entries[0].SHA256 != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("unexpected SHA-256 for a.txt: %s", m.Entries[0].SHA256)
	}
	// SHA-256 of "world" is 486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7
	if m.Entries[1].SHA256 != "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7" {
		t.Errorf("unexpected SHA-256 for b.txt: %s", m.Entries[1].SHA256)
	}
}

func TestComputeManifestHash_ChangesWithContent(t *testing.T) {
	entries1 := []packagekit.PackageEntry{
		{Path: "a.txt", Content: []byte("hello"), Size: 5, ContentType: "text/plain"},
	}
	entries2 := []packagekit.PackageEntry{
		{Path: "a.txt", Content: []byte("world"), Size: 5, ContentType: "text/plain"},
	}

	m1 := BuildManifest(entries1)
	m2 := BuildManifest(entries2)

	if m1.Hash == m2.Hash {
		t.Error("different content should produce different manifest hashes")
	}
}

func TestComputeVersionFingerprint(t *testing.T) {
	files := []skill.SkillFile{
		{FilePath: "README.md", SHA256: "aaa", FileSize: 10},
		{FilePath: "SKILL.md", SHA256: "bbb", FileSize: 20},
		{FilePath: "a.txt", SHA256: "ccc", FileSize: 30},
	}

	fp := ComputeVersionFingerprint(files)
	if fp == "" {
		t.Fatal("fingerprint should not be empty")
	}
	if fp[:7] != "sha256:" {
		t.Errorf("fingerprint should start with 'sha256:', got %s", fp[:7])
	}

	// Same files in different order should produce same fingerprint.
	files2 := []skill.SkillFile{
		{FilePath: "a.txt", SHA256: "ccc", FileSize: 30},
		{FilePath: "SKILL.md", SHA256: "bbb", FileSize: 20},
		{FilePath: "README.md", SHA256: "aaa", FileSize: 10},
	}
	fp2 := ComputeVersionFingerprint(files2)
	if fp != fp2 {
		t.Errorf("fingerprint should be order-independent:\n  fp1: %s\n  fp2: %s", fp, fp2)
	}

	// Different content should produce different fingerprint.
	files3 := []skill.SkillFile{
		{FilePath: "README.md", SHA256: "ddd", FileSize: 10},
	}
	fp3 := ComputeVersionFingerprint(files3)
	if fp == fp3 {
		t.Error("different content should produce different fingerprints")
	}
}

func TestCompareVersions_SameContent(t *testing.T) {
	files := []skill.SkillFile{
		{ID: 1, VersionID: 1, FilePath: "a.txt", SHA256: "abc", FileSize: 10},
		{ID: 2, VersionID: 1, FilePath: "b.txt", SHA256: "def", FileSize: 20},
	}

	result := CompareVersions("1.0", "2.0", files, files)

	if len(result.Files) != 0 {
		t.Errorf("same content should produce no diff files, got %d", len(result.Files))
	}
	if result.Summary.TotalFiles != 0 {
		t.Errorf("expected 0 total diff files, got %d", result.Summary.TotalFiles)
	}
}

func TestCompareVersions_AddedRemovedModified(t *testing.T) {
	fromFiles := []skill.SkillFile{
		{ID: 1, VersionID: 1, FilePath: "a.txt", SHA256: "abc", FileSize: 10},
		{ID: 2, VersionID: 1, FilePath: "b.txt", SHA256: "def", FileSize: 20},
		{ID: 3, VersionID: 1, FilePath: "old.txt", SHA256: "old", FileSize: 5},
	}
	toFiles := []skill.SkillFile{
		{ID: 4, VersionID: 2, FilePath: "a.txt", SHA256: "xyz", FileSize: 12}, // modified
		{ID: 5, VersionID: 2, FilePath: "b.txt", SHA256: "def", FileSize: 20},  // unchanged
		{ID: 6, VersionID: 2, FilePath: "new.txt", SHA256: "new", FileSize: 8}, // added
	}

	result := CompareVersions("1.0", "2.0", fromFiles, toFiles)

	if result.Summary.AddedFiles != 1 {
		t.Errorf("expected 1 added file, got %d", result.Summary.AddedFiles)
	}
	if result.Summary.RemovedFiles != 1 {
		t.Errorf("expected 1 removed file, got %d", result.Summary.RemovedFiles)
	}
	if result.Summary.ModifiedFiles != 1 {
		t.Errorf("expected 1 modified file, got %d", result.Summary.ModifiedFiles)
	}
	if result.Summary.TotalFiles != 3 {
		t.Errorf("expected 3 total diff files, got %d", result.Summary.TotalFiles)
	}

	// Verify each file entry.
	for _, f := range result.Files {
		switch f.Path {
		case "a.txt":
			if f.ChangeType != "MODIFIED" {
				t.Errorf("a.txt should be MODIFIED, got %s", f.ChangeType)
			}
		case "new.txt":
			if f.ChangeType != "ADDED" {
				t.Errorf("new.txt should be ADDED, got %s", f.ChangeType)
			}
		case "old.txt":
			if f.ChangeType != "REMOVED" {
				t.Errorf("old.txt should be REMOVED, got %s", f.ChangeType)
			}
		}
	}
}

func TestCompareVersions_BinaryFile(t *testing.T) {
	fromFiles := []skill.SkillFile{
		{FilePath: "logo.png", SHA256: "abc", FileSize: 1024},
	}
	toFiles := []skill.SkillFile{
		{FilePath: "logo.png", SHA256: "def", FileSize: 2048},
	}

	result := CompareVersions("1.0", "2.0", fromFiles, toFiles)

	if len(result.Files) != 1 {
		t.Fatal("expected 1 diff file")
	}
	if !result.Files[0].Binary {
		t.Error("png file should be marked as binary")
	}
	if result.Files[0].ChangeType != "MODIFIED" {
		t.Errorf("expected MODIFIED, got %s", result.Files[0].ChangeType)
	}
}

func TestCompareTextFiles_BasicDiff(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline2-changed\nline3\nline4\n"

	hunks := CompareTextFiles(old, new)

	// Should find changes around line 2 and the new line 4.
	if len(hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}

	hasDelete := false
	hasAdd := false
	for _, h := range hunks {
		for _, l := range h.Lines {
			if l.Type == "DELETE" && l.Content == "line2" {
				hasDelete = true
			}
			if l.Type == "ADD" && l.Content == "line2-changed" {
				hasAdd = true
			}
		}
	}
	if !hasDelete {
		t.Error("expected DELETE of 'line2'")
	}
	if !hasAdd {
		t.Error("expected ADD of 'line2-changed'")
	}
}

func TestCompareTextFiles_TruncationLimit(t *testing.T) {
	// Generate content exceeding maxDiffLines (5000).
	oldLines := make([]byte, 0, maxDiffLines+100)
	newLines := make([]byte, 0, maxDiffLines+100)
	for i := 0; i <= maxDiffLines; i++ {
		if i > 0 {
			oldLines = append(oldLines, '\n')
			newLines = append(newLines, '\n')
		}
		oldLines = append(oldLines, []byte("line")...)
		newLines = append(newLines, []byte("line")...)
	}

	hunks := CompareTextFiles(string(oldLines), string(newLines))
	if hunks != nil {
		t.Error("should return nil hunks when exceeding maxDiffLines")
	}
}

func TestService_ComputePackageHash(t *testing.T) {
	svc := NewService(nil)

	entries := []packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte("---\nname: test\n---\n"), Size: 20, ContentType: "text/markdown"},
	}

	resp := svc.ComputePackageHash(entries)
	if resp.Manifest.FileCount != 1 {
		t.Errorf("expected 1 file, got %d", resp.Manifest.FileCount)
	}
	if resp.Manifest.Hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestService_WorkspaceFromEntries(t *testing.T) {
	svc := NewService(nil)

	content := "---\nname: My Skill\ndescription: A test skill\nversion: 1.0.0\n---\n\n# Body\n"
	entries := []packagekit.PackageEntry{
		{Path: "SKILL.md", Content: []byte(content), Size: int64(len(content)), ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Readme"), Size: 9, ContentType: "text/markdown"},
	}

	ws, err := svc.WorkspaceFromEntries(entries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.Name != "My Skill" {
		t.Errorf("expected name 'My Skill', got %q", ws.Name)
	}
	if ws.Slug != "my-skill" {
		t.Errorf("expected slug 'my-skill', got %q", ws.Slug)
	}
	if ws.Description != "A test skill" {
		t.Errorf("expected description, got %q", ws.Description)
	}
	if ws.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", ws.Version)
	}
	if ws.FileCount != 2 {
		t.Errorf("expected 2 files, got %d", ws.FileCount)
	}
}

func TestService_WorkspaceFromEntries_MissingSkillMD(t *testing.T) {
	svc := NewService(nil)
	entries := []packagekit.PackageEntry{
		{Path: "README.md", Content: []byte("# Readme"), Size: 9, ContentType: "text/markdown"},
	}

	_, err := svc.WorkspaceFromEntries(entries)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md")
	}
}

func TestService_TriggerEvaluate_Placeholder(t *testing.T) {
	svc := NewService(nil)
	resp := svc.TriggerEvaluate(context.TODO(), EvaluateRequest{SkillID: 1, VersionID: 2, TriggerType: "publish"})
	if resp.Accepted {
		t.Error("placeholder evaluate should not accept requests")
	}
}

func TestService_PrepareProposal_Placeholder(t *testing.T) {
	svc := NewService(nil)
	resp := svc.PrepareProposal(context.TODO(), ProposalRequest{Title: "test"})
	if resp.Accepted {
		t.Error("placeholder proposal should not accept requests")
	}
}
