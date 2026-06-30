// Package packagekit handles archive extraction, path normalization,
// file policy enforcement, and root SKILL.md metadata parsing.
//
// Source module mapping:
//
//	skillhub-domain domain/skill/validation
//	  SkillPackageValidator — requires SKILL.md at root
//	  SkillPackagePolicy — max 500 files, max 10 MB single file, max 100 MB total
//	  Extension allowlist: docs, config, source, scripts, images, office, PDFs
//	  Lightweight content checks: PNG, JPG, GIF, WebP, ICO, PDF, SVG, UTF-8 text
//	  Path normalization: no drive/scheme prefix, no absolute path, no ".." escape
//	  Warnings require confirmWarnings=true for portal; CLI dry-run treats warnings as invalid
//
//	skillhub-domain domain/skill
//	  SkillMetadataParser — parses YAML frontmatter and body from SKILL.md
//	  SkillPublishService derives slug from metadata.name(), auto-generates version
package packagekit
