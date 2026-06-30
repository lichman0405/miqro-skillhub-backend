// Package promotion manages the workflow for copying an approved
// skill into the global namespace.
//
// Source module mapping:
//
//	skillhub-domain domain/review/PromotionService
//	  PromotionRequest entity — request to copy to global namespace
//	  PromotionService — submission and approval
//	  Approval creates new target Skill in global namespace with source_skill_id
//	  Copies file records and reuses storage keys
//	  Sets latest version pointer on new skill
//	  Emits PromotionApprovedEvent, PublicationEvent
//	  Rejects duplicate pending or already-approved promotion
//
// Implemented in Phase 06.
package promotion

