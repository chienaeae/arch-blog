package events

import (
	"time"

	"backend/internal/platform/eventbus"
	"github.com/google/uuid"
)

// Theme event topics
const (
	ThemeCreatedTopic           eventbus.Topic = "themes.created"
	ThemeUpdatedTopic           eventbus.Topic = "themes.updated"
	ThemeActivatedTopic         eventbus.Topic = "themes.activated"
	ThemeDeactivatedTopic       eventbus.Topic = "themes.deactivated"
	ThemeDeletedTopic           eventbus.Topic = "themes.deleted"
	ThemeArticleAddedTopic      eventbus.Topic = "themes.article.added"
	ThemeArticleRemovedTopic    eventbus.Topic = "themes.article.removed"
	ThemeArticlesReorderedTopic eventbus.Topic = "themes.articles.reordered"
)

// ThemeCreatedEvent is published when a new theme is created
type ThemeCreatedEvent struct {
	ThemeID    uuid.UUID
	ActorID    uuid.UUID // Curator who created the theme
	Name       string
	Slug       string
	OccurredAt time.Time
}

// ThemeUpdatedEvent is published when a theme is updated
type ThemeUpdatedEvent struct {
	ThemeID    uuid.UUID
	ActorID    uuid.UUID // User who updated the theme
	Name       string
	Slug       string
	OccurredAt time.Time
}

// ThemeActivatedEvent is published when a theme is activated
type ThemeActivatedEvent struct {
	ThemeID    uuid.UUID
	ActorID    uuid.UUID // User who activated the theme
	OccurredAt time.Time
}

// ThemeDeactivatedEvent is published when a theme is deactivated
type ThemeDeactivatedEvent struct {
	ThemeID    uuid.UUID
	ActorID    uuid.UUID // User who deactivated the theme
	OccurredAt time.Time
}

// ThemeDeletedEvent is published when a theme is deleted
type ThemeDeletedEvent struct {
	ThemeID    uuid.UUID
	ActorID    uuid.UUID // User who deleted the theme
	OccurredAt time.Time
}

// ThemeArticleAddedEvent is published when an article is added to a theme
type ThemeArticleAddedEvent struct {
	ThemeID    uuid.UUID
	PostID     uuid.UUID
	Position   int
	ActorID    uuid.UUID // User who added the article
	OccurredAt time.Time
}

// ThemeArticleRemovedEvent is published when an article is removed from a theme
type ThemeArticleRemovedEvent struct {
	ThemeID    uuid.UUID
	PostID     uuid.UUID
	ActorID    uuid.UUID // User who removed the article
	OccurredAt time.Time
}

// ThemeArticlesReorderedEvent is published when articles in a theme are reordered
type ThemeArticlesReorderedEvent struct {
	ThemeID        uuid.UUID
	OrderedPostIDs []uuid.UUID
	ActorID        uuid.UUID // User who reordered the articles
	OccurredAt     time.Time
}
