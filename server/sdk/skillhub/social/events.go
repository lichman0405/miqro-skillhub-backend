package social

import "miqro-skillhub/server/sdk/skillhub/eventbus"

// SkillStarredEvent is emitted when a user stars a skill.
type SkillStarredEvent struct {
	SkillID int64
	UserID  string
}

func (e SkillStarredEvent) EventName() string { return "social.starred" }

// SkillUnstarredEvent is emitted when a user unstars a skill.
type SkillUnstarredEvent struct {
	SkillID int64
	UserID  string
}

func (e SkillUnstarredEvent) EventName() string { return "social.unstarred" }

// SkillRatedEvent is emitted when a user rates a skill.
type SkillRatedEvent struct {
	SkillID int64
	UserID  string
	Score   int16
}

func (e SkillRatedEvent) EventName() string { return "social.rated" }

// SkillSubscribedEvent is emitted when a user subscribes to a skill.
type SkillSubscribedEvent struct {
	SkillID int64
	UserID  string
}

func (e SkillSubscribedEvent) EventName() string { return "social.subscribed" }

// SkillUnsubscribedEvent is emitted when a user unsubscribes from a skill.
type SkillUnsubscribedEvent struct {
	SkillID int64
	UserID  string
}

func (e SkillUnsubscribedEvent) EventName() string { return "social.unsubscribed" }

var _ eventbus.Event = SkillStarredEvent{}
var _ eventbus.Event = SkillUnstarredEvent{}
var _ eventbus.Event = SkillRatedEvent{}
var _ eventbus.Event = SkillSubscribedEvent{}
var _ eventbus.Event = SkillUnsubscribedEvent{}
