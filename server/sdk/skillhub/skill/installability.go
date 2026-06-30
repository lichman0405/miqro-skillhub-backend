package skill

// ---------------------------------------------------------------------------
// SkillInstallability — determines if a version is installable/downloadable
// ---------------------------------------------------------------------------

// SkillInstallability defines whether a skill version can be installed
// through public download paths.
// Mirrors source com.iflytek.skillhub.domain.skill.SkillInstallability.
type SkillInstallability struct{}

// IsInstallableVersion returns true when the version is published,
// download-ready, and not yanked.
func (si *SkillInstallability) IsInstallableVersion(version SkillVersion) bool {
	return version.Status == "PUBLISHED" &&
		version.DownloadReady &&
		version.YankedAt == nil
}

// IsInstallable is a package-level convenience function.
func IsInstallable(version SkillVersion) bool {
	return version.Status == "PUBLISHED" &&
		version.DownloadReady &&
		version.YankedAt == nil
}
