package domain

import "github.com/google/uuid"

// PostInfo represents the minimal post information needed by the themes domain
// This is an anti-corruption layer to avoid direct dependency on the posts domain
type PostInfo interface {
	GetID() uuid.UUID
	IsPublished() bool
	GetAuthorID() uuid.UUID
}