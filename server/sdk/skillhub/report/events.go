package report

import "miqro-skillhub/server/sdk/skillhub/eventbus"

// ReportSubmittedEvent is emitted when a report is submitted.
type ReportSubmittedEvent struct {
	ReportID   int64
	SkillID    int64
	ReporterID string
}

func (e ReportSubmittedEvent) EventName() string { return "report.submitted" }

// ReportResolvedEvent is emitted when a report is resolved or dismissed.
type ReportResolvedEvent struct {
	ReportID   int64
	SkillID    int64
	ActorID    string
	ReporterID string
	Outcome    string // "resolved" or "dismissed"
}

func (e ReportResolvedEvent) EventName() string { return "report.resolved" }

var _ eventbus.Event = ReportSubmittedEvent{}
var _ eventbus.Event = ReportResolvedEvent{}
