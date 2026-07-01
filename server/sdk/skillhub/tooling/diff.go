package tooling

import (
	"strings"

	"miqro-skillhub/server/sdk/skillhub/skill"
)

const (
	maxDiffFileBytes = 1 * 1024 * 1024 // 1 MB
	maxDiffLines     = 5000
)

// binaryFileExtensions mirrors source SkillQueryService.BINARY_FILE_EXTENSIONS.
var binaryFileExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2", ".ttf", ".eot",
	".zip", ".tar", ".gz", ".jar", ".war", ".class", ".so", ".dll", ".exe", ".pdf",
}

// CompareVersions produces a version diff between two sets of skill files.
// It mirrors source SkillQueryService.compareVersions.
func CompareVersions(fromVersion, toVersion string, fromFiles, toFiles []skill.SkillFile) VersionDiff {
	fromMap := make(map[string]skill.SkillFile, len(fromFiles))
	for _, f := range fromFiles {
		fromMap[f.FilePath] = f
	}
	toMap := make(map[string]skill.SkillFile, len(toFiles))
	for _, f := range toFiles {
		toMap[f.FilePath] = f
	}

	// Collect all unique paths.
	paths := make(map[string]bool)
	for _, f := range fromFiles {
		paths[f.FilePath] = true
	}
	for _, f := range toFiles {
		paths[f.FilePath] = true
	}

	var files []DiffFile
	var addedFiles, modifiedFiles, removedFiles, addedLines, removedLines int

	// Sort paths for deterministic output.
	sortedPaths := sortedKeys(paths)

	for _, path := range sortedPaths {
		fromFile, fromExists := fromMap[path]
		toFile, toExists := toMap[path]

		// Same SHA-256 → skip.
		if fromExists && toExists && fromFile.SHA256 == toFile.SHA256 {
			continue
		}

		file := buildDiffFile(path, fromExists, toExists, fromFile, toFile)
		if file == nil {
			continue
		}
		files = append(files, *file)

		switch file.ChangeType {
		case "ADDED":
			addedFiles++
		case "REMOVED":
			removedFiles++
		case "MODIFIED":
			modifiedFiles++
		}

		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				switch line.Type {
				case "ADD":
					addedLines++
				case "DELETE":
					removedLines++
				}
			}
		}
	}

	return VersionDiff{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Summary: DiffSummary{
			TotalFiles:    len(files),
			AddedFiles:    addedFiles,
			ModifiedFiles: modifiedFiles,
			RemovedFiles:  removedFiles,
			AddedLines:    addedLines,
			RemovedLines:  removedLines,
		},
		Files: files,
	}
}

func buildDiffFile(path string, fromExists, toExists bool, fromFile, toFile skill.SkillFile) *DiffFile {
	var changeType string
	var oldSize, newSize *int64

	if !fromExists {
		changeType = "ADDED"
		s := toFile.FileSize
		newSize = &s
	} else if !toExists {
		changeType = "REMOVED"
		s := fromFile.FileSize
		oldSize = &s
	} else {
		changeType = "MODIFIED"
		fs := fromFile.FileSize
		ts := toFile.FileSize
		oldSize = &fs
		newSize = &ts
	}

	if isBinaryPath(path) {
		return &DiffFile{
			Path:       path,
			ChangeType: changeType,
			OldSize:    oldSize,
			NewSize:    newSize,
			Binary:     true,
			Truncated:  false,
		}
	}

	// For text diff, we'd need file content. Since we only have file metadata
	// (no content access in this layer), we produce a metadata-only diff.
	// Content-level diffs are produced at the HTTP/service layer when content is available.
	return &DiffFile{
		Path:       path,
		ChangeType: changeType,
		OldSize:    oldSize,
		NewSize:    newSize,
		Binary:     false,
		Truncated:  false,
	}
}

// CompareTextFiles produces a line-level text diff between two content strings.
// It returns the list of DiffHunks using a simple LCS-based algorithm.
func CompareTextFiles(oldContent, newContent string) []DiffHunk {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	// Truncate if too large.
	if len(oldLines) > maxDiffLines || len(newLines) > maxDiffLines {
		return nil // caller handles truncated flag
	}

	diffs := computeLCSDiff(oldLines, newLines)
	return groupIntoHunks(diffs, oldLines, newLines)
}

// internalDiff represents one line-level diff operation.
type internalDiff struct {
	op       string // "equal", "add", "delete"
	oldStart int    // index in oldLines
	newStart int    // index in newLines
	count    int
}

// computeLCSDiff computes a diff using the Longest Common Subsequence algorithm.
func computeLCSDiff(oldLines, newLines []string) []internalDiff {
	m, n := len(oldLines), len(newLines)

	// Build LCS table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Backtrack to produce diff.
	var result []internalDiff
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			result = append(result, internalDiff{op: "equal", oldStart: i - 1, newStart: j - 1, count: 1})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, internalDiff{op: "add", oldStart: i, newStart: j - 1, count: 1})
			j--
		} else {
			result = append(result, internalDiff{op: "delete", oldStart: i - 1, newStart: j, count: 1})
			i--
		}
	}

	// Reverse to get forward order.
	for a, b := 0, len(result)-1; a < b; a, b = a+1, b-1 {
		result[a], result[b] = result[b], result[a]
	}

	return result
}

// groupIntoHunks groups raw diff operations into contiguous hunks.
func groupIntoHunks(diffs []internalDiff, oldLines, newLines []string) []DiffHunk {
	if len(diffs) == 0 {
		return nil
	}

	// Find change regions (non-equal consecutive diffs).
	type region struct {
		start int
		end   int
	}
	var regions []region
	i := 0
	for i < len(diffs) {
		if diffs[i].op == "equal" {
			i++
			continue
		}
		start := i
		for i < len(diffs) && diffs[i].op != "equal" {
			i++
		}
		regions = append(regions, region{start, i})
	}

	// Context lines around each change (3 lines context).
	const contextLines = 3

	var hunks []DiffHunk
	for _, r := range regions {
		hunkStart := r.start - contextLines
		if hunkStart < 0 {
			hunkStart = 0
		}
		hunkEnd := r.end + contextLines
		if hunkEnd > len(diffs) {
			hunkEnd = len(diffs)
		}

		// Merge overlapping hunks with previous.
		var hunkLines []DiffLine
		oldStart := -1
		newStart := -1
		oldCount := 0
		newCount := 0

		for k := hunkStart; k < hunkEnd; k++ {
			d := diffs[k]
			switch d.op {
			case "equal":
				if oldStart < 0 {
					oldStart = d.oldStart
				}
				if newStart < 0 {
					newStart = d.newStart
				}
				lineNum := d.oldStart + 1
				line := oldLines[d.oldStart]
				hunkLines = append(hunkLines, DiffLine{
					Type:          "CONTEXT",
					Content:        line,
					OldLineNumber: &lineNum,
					NewLineNumber: &lineNum, // same for context
				})
				oldCount++
				newCount++
			case "delete":
				if oldStart < 0 {
					oldStart = d.oldStart
				}
				lineNum := d.oldStart + 1
				hunkLines = append(hunkLines, DiffLine{
					Type:          "DELETE",
					Content:        oldLines[d.oldStart],
					OldLineNumber: &lineNum,
				})
				oldCount++
			case "add":
				if newStart < 0 {
					newStart = d.newStart
				}
				lineNum := d.newStart + 1
				hunkLines = append(hunkLines, DiffLine{
					Type:          "ADD",
					Content:        newLines[d.newStart],
					NewLineNumber: &lineNum,
				})
				newCount++
			}
		}

		if oldStart < 0 {
			oldStart = 0
		}
		if newStart < 0 {
			newStart = 0
		}
		hunks = append(hunks, DiffHunk{
			OldStart: oldStart + 1,
			OldLines: oldCount,
			NewStart: newStart + 1,
			NewLines: newCount,
			Lines:    hunkLines,
		})
	}

	return hunks
}

// splitLines splits content into lines, keeping trailing newline behavior.
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	// Handle trailing newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func isBinaryPath(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range binaryFileExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
