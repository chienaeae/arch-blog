package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/philly/arch-blog/backend/internal/platform/eventbus"
)

// Event topics for posts
const (
	PostCreatedTopic   eventbus.Topic = "posts.created"
	PostUpdatedTopic   eventbus.Topic = "posts.updated"
	PostPublishedTopic eventbus.Topic = "posts.published"
	PostArchivedTopic  eventbus.Topic = "posts.archived"
	PostDeletedTopic   eventbus.Topic = "posts.deleted"
)

// PostCreatedEvent is published when a new post is created
type PostCreatedEvent struct {
	PostID     uuid.UUID
	ActorID    uuid.UUID // Author who created the post
	Title      string
	Slug       string
	OccurredAt time.Time
}

// PostUpdatedEvent is published when a post is updated
type PostUpdatedEvent struct {
	PostID     uuid.UUID
	ActorID    uuid.UUID // User who updated the post
	Title      string
	Slug       string
	OccurredAt time.Time
}

// PostPublishedEvent is published when a post is published
type PostPublishedEvent struct {
	PostID      uuid.UUID
	ActorID     uuid.UUID // User who published the post
	PublishedAt time.Time
	OccurredAt  time.Time
}

// PostArchivedEvent is published when a post is archived
type PostArchivedEvent struct {
	PostID     uuid.UUID
	ActorID    uuid.UUID // User who archived the post
	OccurredAt time.Time
}

// PostDeletedEvent is published when a post is deleted
type PostDeletedEvent struct {
	PostID     uuid.UUID
	ActorID    uuid.UUID // User who deleted the post
	OccurredAt time.Time
}