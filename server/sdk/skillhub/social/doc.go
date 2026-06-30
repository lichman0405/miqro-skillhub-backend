// Package social manages user interactions with skills: stars,
// ratings, and subscriptions.
//
// Source module mapping:
//
//	skillhub-domain domain/social
//	  Stars (star / unstar)
//	  Ratings (rate)
//	  Subscriptions (subscribe / unsubscribe)
//	  Events: StarredEvent, UnstarredEvent, RatedEvent,
//	          SubscribedEvent, UnsubscribedEvent
//
// Implementation starts in Phase 07.
package social
