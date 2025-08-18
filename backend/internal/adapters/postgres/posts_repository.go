package postgres

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/platform/postgres"
	"backend/internal/posts/domain"
	"backend/internal/posts/ports"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostRepository implements the posts.PostRepository interface using PostgreSQL
type PostRepository struct {
	postgres.BaseRepository // Embed the base repository for common functionality
}

// NewPostRepository creates a new PostgreSQL posts repository
func NewPostRepository(db *pgxpool.Pool) *PostRepository {
	return &PostRepository{
		BaseRepository: postgres.NewBaseRepository(db),
	}
}

// WithTx creates a new repository instance that uses the provided transaction
func (r *PostRepository) WithTx(tx pgx.Tx) ports.PostRepository {
	return &PostRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// Create inserts a new post into the database
func (r *PostRepository) Create(ctx context.Context, post *domain.Post) error {
	var publishedAt pgtype.Timestamptz
	if post.PublishedAt != nil {
		publishedAt = pgtype.Timestamptz{
			Time:  *post.PublishedAt,
			Valid: true,
		}
	}

	query, args, err := r.SB.
		Insert("posts").
		Columns(
			"id", "title", "content", "excerpt", "slug", "status",
			"author_id", "published_at", "created_at", "updated_at",
		).
		Values(
			pgtype.UUID{Bytes: uuid.UUID(post.ID), Valid: true},
			post.Title,
			post.Content,
			post.Excerpt,
			post.Slug,
			string(post.Status),
			pgtype.UUID{Bytes: uuid.UUID(post.AuthorID), Valid: true},
			publishedAt,
			pgtype.Timestamptz{Time: post.CreatedAt, Valid: true},
			pgtype.Timestamptz{Time: post.UpdatedAt, Valid: true},
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("PostRepository.Create: build query: %w", err)
	}

	_, err = r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("PostRepository.Create: %w", err)
	}

	return nil
}

// Update updates an existing post in the database
func (r *PostRepository) Update(ctx context.Context, post *domain.Post) error {
	var publishedAt pgtype.Timestamptz
	if post.PublishedAt != nil {
		publishedAt = pgtype.Timestamptz{
			Time:  *post.PublishedAt,
			Valid: true,
		}
	}

	query, args, err := r.SB.
		Update("posts").
		Set("title", post.Title).
		Set("content", post.Content).
		Set("excerpt", post.Excerpt).
		Set("slug", post.Slug).
		Set("status", string(post.Status)).
		Set("published_at", publishedAt).
		Set("updated_at", pgtype.Timestamptz{Time: post.UpdatedAt, Valid: true}).
		Where(sq.Eq{"id": pgtype.UUID{Bytes: uuid.UUID(post.ID), Valid: true}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("PostRepository.Update: build query: %w", err)
	}

	result, err := r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("PostRepository.Update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ports.ErrPostNotFound
	}

	return nil
}

// Delete removes a post from the database
func (r *PostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.SB.
		Delete("posts").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: id, Valid: true}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("PostRepository.Delete: build query: %w", err)
	}

	result, err := r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("PostRepository.Delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ports.ErrPostNotFound
	}

	return nil
}

// FindByID retrieves a post by its ID
func (r *PostRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	query, args, err := r.SB.
		Select(
			"id", "title", "content", "excerpt", "slug", "status",
			"author_id", "published_at", "created_at", "updated_at",
		).
		From("posts").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: id, Valid: true}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("PostRepository.FindByID: build query: %w", err)
	}

	row := r.DB.QueryRow(ctx, query, args...)
	post, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrPostNotFound
		}
		return nil, fmt.Errorf("PostRepository.FindByID: %w", err)
	}

	return post, nil
}

// FindBySlug retrieves a post by its URL slug
func (r *PostRepository) FindBySlug(ctx context.Context, slug string) (*domain.Post, error) {
	query, args, err := r.SB.
		Select(
			"id", "title", "content", "excerpt", "slug", "status",
			"author_id", "published_at", "created_at", "updated_at",
		).
		From("posts").
		Where(sq.Eq{"slug": slug}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("PostRepository.FindBySlug: build query: %w", err)
	}

	row := r.DB.QueryRow(ctx, query, args...)
	post, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrPostNotFound
		}
		return nil, fmt.Errorf("PostRepository.FindBySlug: %w", err)
	}

	return post, nil
}

// ListSummaries retrieves a list of post summaries based on the filter
func (r *PostRepository) ListSummaries(ctx context.Context, filter ports.ListFilter) ([]*ports.PostSummary, error) {
	// Start with a fresh query builder for the main query
	qb := r.SB.Select(
		"p.id", "p.title", "p.excerpt", "p.slug", "p.status",
		"p.author_id", "u.username as author_name",
		"p.published_at", "p.created_at", "p.updated_at",
	).
		From("posts p").
		LeftJoin("users u ON p.author_id = u.id")

	// Apply filters
	qb = r.applyFilters(qb, filter)

	// Add sorting
	orderColumn := getOrderColumn(filter.OrderBy)
	if filter.OrderDesc {
		qb = qb.OrderBy(fmt.Sprintf("%s DESC", orderColumn))
	} else {
		qb = qb.OrderBy(fmt.Sprintf("%s ASC", orderColumn))
	}

	// Add pagination
	if filter.Limit > 0 {
		qb = qb.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(uint64(filter.Offset))
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("PostRepository.ListSummaries: build query: %w", err)
	}

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("PostRepository.ListSummaries: %w", err)
	}
	defer rows.Close()

	var summaries []*ports.PostSummary
	for rows.Next() {
		summary, err := scanPostSummaryFromRows(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PostRepository.ListSummaries: rows error: %w", err)
	}

	return summaries, nil
}

// Count returns the total number of posts matching the filter
func (r *PostRepository) Count(ctx context.Context, filter ports.ListFilter) (int, error) {
	// Start with a fresh query builder for count
	qb := r.SB.Select("COUNT(*)").From("posts p")

	// Apply filters
	qb = r.applyFilters(qb, filter)

	query, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("PostRepository.Count: build query: %w", err)
	}

	var count int
	err = r.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("PostRepository.Count: %w", err)
	}

	return count, nil
}

// SlugExists checks if a slug already exists, optionally excluding a specific post ID
func (r *PostRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	// Build the subquery
	subQuery := r.SB.Select("1").From("posts").Where(sq.Eq{"slug": slug})

	if excludeID != nil {
		subQuery = subQuery.Where(sq.NotEq{"id": pgtype.UUID{Bytes: *excludeID, Valid: true}})
	}

	// Build the EXISTS query - we need to build the full SQL manually
	subQuerySQL, subQueryArgs, err := subQuery.ToSql()
	if err != nil {
		return false, fmt.Errorf("PostRepository.SlugExists: build subquery: %w", err)
	}

	// Construct the EXISTS query manually since squirrel doesn't support it directly
	query := fmt.Sprintf("SELECT EXISTS(%s)", subQuerySQL)

	var exists bool
	err = r.DB.QueryRow(ctx, query, subQueryArgs...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("PostRepository.SlugExists: %w", err)
	}

	return exists, nil
}

// FindSummariesByAuthor retrieves post summaries by a specific author
func (r *PostRepository) FindSummariesByAuthor(ctx context.Context, authorID uuid.UUID, filter ports.ListFilter) ([]*ports.PostSummary, error) {
	// Override the filter to include the author
	filter.AuthorID = &authorID
	return r.ListSummaries(ctx, filter)
}

// GetPostAuthor retrieves just the author ID for a post (for ownership checks)
func (r *PostRepository) GetPostAuthor(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	query, args, err := r.SB.
		Select("author_id").
		From("posts").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: postID, Valid: true}}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("PostRepository.GetPostAuthor: build query: %w", err)
	}

	var authorIDBytes pgtype.UUID
	err = r.DB.QueryRow(ctx, query, args...).Scan(&authorIDBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ports.ErrPostNotFound
		}
		return uuid.Nil, fmt.Errorf("PostRepository.GetPostAuthor: %w", err)
	}

	return uuid.UUID(authorIDBytes.Bytes), nil
}

// Helper methods

// applyFilters applies common WHERE clauses to a query builder
func (r *PostRepository) applyFilters(qb sq.SelectBuilder, filter ports.ListFilter) sq.SelectBuilder {
	// Add status filter
	if filter.Status != nil {
		qb = qb.Where(sq.Eq{"p.status": string(*filter.Status)})
	}

	// Add author filter
	if filter.AuthorID != nil {
		qb = qb.Where(sq.Eq{"p.author_id": pgtype.UUID{Bytes: *filter.AuthorID, Valid: true}})
	}

	// Add search query if provided
	if filter.SearchQuery != "" {
		searchPattern := "%" + filter.SearchQuery + "%"
		qb = qb.Where(sq.Or{
			sq.Like{"p.title": searchPattern},
			sq.Like{"p.excerpt": searchPattern},
		})
	}

	return qb
}

// getOrderColumn validates and returns the actual column name for ordering
func getOrderColumn(field ports.OrderField) string {
	switch field {
	case ports.OrderByCreatedAt:
		return "p.created_at"
	case ports.OrderByUpdatedAt:
		return "p.updated_at"
	case ports.OrderByPublishedAt:
		return "p.published_at"
	case ports.OrderByTitle:
		return "p.title"
	default:
		return "p.created_at"
	}
}

// scanPost scans a single post from pgx.Row
func scanPost(row pgx.Row) (*domain.Post, error) {
	var post domain.Post
	var publishedAt pgtype.Timestamptz
	var idBytes, authorIDBytes pgtype.UUID
	var statusStr string

	err := row.Scan(
		&idBytes,
		&post.Title,
		&post.Content,
		&post.Excerpt,
		&post.Slug,
		&statusStr,
		&authorIDBytes,
		&publishedAt,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanPost: %w", err)
	}

	// Convert pgtype values
	post.ID = uuid.UUID(idBytes.Bytes)
	post.AuthorID = uuid.UUID(authorIDBytes.Bytes)

	// Parse status
	post.Status = domain.PostStatus(statusStr)
	if !post.Status.IsValid() {
		return nil, fmt.Errorf("scanPost: invalid status %s", statusStr)
	}

	// Handle nullable published_at
	if publishedAt.Valid {
		post.PublishedAt = &publishedAt.Time
	}

	return &post, nil
}

// scanPostSummaryFromRows scans a post summary from pgx.Rows
func scanPostSummaryFromRows(rows pgx.Rows) (*ports.PostSummary, error) {
	var summary ports.PostSummary
	var publishedAt pgtype.Timestamptz
	var idBytes, authorIDBytes pgtype.UUID
	var statusStr string
	var authorName pgtype.Text

	err := rows.Scan(
		&idBytes,
		&summary.Title,
		&summary.Excerpt,
		&summary.Slug,
		&statusStr,
		&authorIDBytes,
		&authorName,
		&publishedAt,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanPostSummaryFromRows: %w", err)
	}

	// Convert pgtype values
	summary.ID = uuid.UUID(idBytes.Bytes)
	summary.AuthorID = uuid.UUID(authorIDBytes.Bytes)

	if authorName.Valid {
		summary.AuthorName = authorName.String
	}

	// Parse status
	summary.Status = domain.PostStatus(statusStr)
	if !summary.Status.IsValid() {
		return nil, fmt.Errorf("scanPostSummaryFromRows: invalid status %s", statusStr)
	}

	// Handle nullable published_at
	if publishedAt.Valid {
		summary.PublishedAt = &publishedAt.Time
	}

	return &summary, nil
}

// Compile-time check to ensure PostRepository implements ports.PostRepository
var _ ports.PostRepository = (*PostRepository)(nil)
