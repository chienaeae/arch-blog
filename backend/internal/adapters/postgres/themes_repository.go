package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/philly/arch-blog/backend/internal/platform/postgres"
	"github.com/philly/arch-blog/backend/internal/themes/domain"
	"github.com/philly/arch-blog/backend/internal/themes/ports"
)

// ThemeRepository implements the themes.ThemeRepository interface using PostgreSQL
type ThemeRepository struct {
	postgres.BaseRepository // Embed the base repository for common functionality
}

// NewThemeRepository creates a new PostgreSQL themes repository
func NewThemeRepository(db *pgxpool.Pool) *ThemeRepository {
	return &ThemeRepository{
		BaseRepository: postgres.NewBaseRepository(db),
	}
}

// WithTx creates a new repository instance that uses the provided transaction
func (r *ThemeRepository) WithTx(tx pgx.Tx) ports.ThemeRepository {
	return &ThemeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// Create inserts a new theme into the database
func (r *ThemeRepository) Create(ctx context.Context, theme *domain.Theme) error {
	query, args, err := r.SB.
		Insert("themes").
		Columns(
			"id", "name", "description", "slug",
			"curator_id", "is_active", "created_at", "updated_at",
		).
		Values(
			pgtype.UUID{Bytes: uuid.UUID(theme.ID), Valid: true},
			theme.Name,
			theme.Description,
			theme.Slug,
			pgtype.UUID{Bytes: uuid.UUID(theme.CuratorID), Valid: true},
			theme.IsActive,
			pgtype.Timestamptz{Time: theme.CreatedAt, Valid: true},
			pgtype.Timestamptz{Time: theme.UpdatedAt, Valid: true},
		).
		ToSql()

	if err != nil {
		return fmt.Errorf("ThemeRepository.Create: build query: %w", err)
	}

	_, err = r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ThemeRepository.Create: %w", err)
	}

	return nil
}

// Save persists the entire aggregate atomically
// Note: This method assumes it's being called within a transaction context.
// The service layer is responsible for transaction management.
func (r *ThemeRepository) Save(ctx context.Context, theme *domain.Theme) error {
	// Step 1: Update the theme entity itself
	query, args, err := r.SB.
		Update("themes").
		Set("name", theme.Name).
		Set("description", theme.Description).
		Set("slug", theme.Slug).
		Set("is_active", theme.IsActive).
		Set("updated_at", pgtype.Timestamptz{Time: theme.UpdatedAt, Valid: true}).
		Where(sq.Eq{"id": pgtype.UUID{Bytes: uuid.UUID(theme.ID), Valid: true}}).
		ToSql()

	if err != nil {
		return fmt.Errorf("ThemeRepository.Save: build update query: %w", err)
	}

	result, err := r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ThemeRepository.Save: update theme: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ports.ErrThemeNotFound
	}

	// Step 2: Sync the articles collection (diff and sync algorithm)
	if err := r.syncArticles(ctx, theme.ID, theme.Articles); err != nil {
		return fmt.Errorf("ThemeRepository.Save: sync articles: %w", err)
	}

	return nil
}

// Delete removes a theme from the database
func (r *ThemeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.SB.
		Delete("themes").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: id, Valid: true}}).
		ToSql()

	if err != nil {
		return fmt.Errorf("ThemeRepository.Delete: build query: %w", err)
	}

	result, err := r.DB.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("ThemeRepository.Delete: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ports.ErrThemeNotFound
	}

	return nil
}

// FindByID retrieves a theme by its ID (without articles)
func (r *ThemeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Theme, error) {
	query, args, err := r.SB.
		Select(
			"id", "name", "description", "slug",
			"curator_id", "is_active", "created_at", "updated_at",
		).
		From("themes").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: id, Valid: true}}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.FindByID: build query: %w", err)
	}

	row := r.DB.QueryRow(ctx, query, args...)
	theme, err := scanTheme(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrThemeNotFound
		}
		return nil, fmt.Errorf("ThemeRepository.FindByID: %w", err)
	}

	return theme, nil
}

// FindBySlug retrieves a theme by its URL slug (without articles)
func (r *ThemeRepository) FindBySlug(ctx context.Context, slug string) (*domain.Theme, error) {
	query, args, err := r.SB.
		Select(
			"id", "name", "description", "slug",
			"curator_id", "is_active", "created_at", "updated_at",
		).
		From("themes").
		Where(sq.Eq{"slug": slug}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.FindBySlug: build query: %w", err)
	}

	row := r.DB.QueryRow(ctx, query, args...)
	theme, err := scanTheme(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrThemeNotFound
		}
		return nil, fmt.Errorf("ThemeRepository.FindBySlug: %w", err)
	}

	return theme, nil
}

// LoadThemeWithArticles loads the full theme aggregate including articles
func (r *ThemeRepository) LoadThemeWithArticles(ctx context.Context, id uuid.UUID) (*domain.Theme, error) {
	// First load the theme
	theme, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Then load its articles
	query, args, err := r.SB.
		Select(
			"ta.post_id", "ta.position", "ta.added_by", "ta.added_at", "ta.updated_at",
		).
		From("theme_articles ta").
		Where(sq.Eq{"ta.theme_id": pgtype.UUID{Bytes: id, Valid: true}}).
		OrderBy("ta.position ASC").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.LoadThemeWithArticles: build articles query: %w", err)
	}

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.LoadThemeWithArticles: query articles: %w", err)
	}
	defer rows.Close()

	var articles []*domain.ThemeArticle
	for rows.Next() {
		var article domain.ThemeArticle
		var postIDBytes, addedByBytes pgtype.UUID

		err := rows.Scan(
			&postIDBytes,
			&article.Position,
			&addedByBytes,
			&article.AddedAt,
			&article.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ThemeRepository.LoadThemeWithArticles: scan article: %w", err)
		}

		article.ThemeID = theme.ID
		article.PostID = uuid.UUID(postIDBytes.Bytes)
		article.AddedBy = uuid.UUID(addedByBytes.Bytes)
		articles = append(articles, &article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ThemeRepository.LoadThemeWithArticles: rows error: %w", err)
	}

	theme.Articles = articles
	return theme, nil
}

// ListThemes retrieves a list of theme summaries based on the filter
func (r *ThemeRepository) ListThemes(ctx context.Context, filter ports.ListFilter) ([]*ports.ThemeSummary, error) {
	// Start with a fresh query builder for the main query
	qb := r.SB.Select(
		"t.id", "t.name", "t.description", "t.slug",
		"t.curator_id", "u.username as curator_name",
		"t.is_active", "t.created_at", "t.updated_at",
		"COUNT(DISTINCT ta.post_id) as article_count",
	).
		From("themes t").
		LeftJoin("users u ON t.curator_id = u.id").
		LeftJoin("theme_articles ta ON t.id = ta.theme_id").
		GroupBy("t.id", "t.name", "t.description", "t.slug", "t.curator_id", "u.username", "t.is_active", "t.created_at", "t.updated_at")

	// Apply filters
	qb = r.applyThemeFilters(qb, filter)

	// Add sorting - default to created_at DESC
	qb = qb.OrderBy("t.created_at DESC")

	// Add pagination
	if filter.Limit > 0 {
		qb = qb.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		qb = qb.Offset(uint64(filter.Offset))
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.ListThemes: build query: %w", err)
	}

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThemeRepository.ListThemes: %w", err)
	}
	defer rows.Close()

	var summaries []*ports.ThemeSummary
	for rows.Next() {
		summary, err := scanThemeSummaryFromRows(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ThemeRepository.ListThemes: rows error: %w", err)
	}

	return summaries, nil
}

// CountThemes returns the total number of themes matching the filter
func (r *ThemeRepository) CountThemes(ctx context.Context, filter ports.ListFilter) (int, error) {
	// Start with a fresh query builder for count
	qb := r.SB.Select("COUNT(*)").From("themes t")

	// Apply filters
	qb = r.applyThemeFilters(qb, filter)

	query, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("ThemeRepository.CountThemes: build query: %w", err)
	}

	var count int
	err = r.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ThemeRepository.CountThemes: %w", err)
	}

	return count, nil
}

// SlugExists checks if a slug already exists, optionally excluding a specific theme ID
func (r *ThemeRepository) SlugExists(ctx context.Context, slug string, excludeID *uuid.UUID) (bool, error) {
	// Build the subquery
	subQuery := r.SB.Select("1").From("themes").Where(sq.Eq{"slug": slug})

	if excludeID != nil {
		subQuery = subQuery.Where(sq.NotEq{"id": pgtype.UUID{Bytes: *excludeID, Valid: true}})
	}

	// Build the EXISTS query - we need to build the full SQL manually
	subQuerySQL, subQueryArgs, err := subQuery.ToSql()
	if err != nil {
		return false, fmt.Errorf("ThemeRepository.SlugExists: build subquery: %w", err)
	}

	// Construct the EXISTS query manually since squirrel doesn't support it directly
	query := fmt.Sprintf("SELECT EXISTS(%s)", subQuerySQL)

	var exists bool
	err = r.DB.QueryRow(ctx, query, subQueryArgs...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ThemeRepository.SlugExists: %w", err)
	}

	return exists, nil
}

// GetThemeCurator retrieves just the curator ID for a theme (for ownership checks)
func (r *ThemeRepository) GetThemeCurator(ctx context.Context, themeID uuid.UUID) (uuid.UUID, error) {
	query, args, err := r.SB.
		Select("curator_id").
		From("themes").
		Where(sq.Eq{"id": pgtype.UUID{Bytes: themeID, Valid: true}}).
		ToSql()

	if err != nil {
		return uuid.Nil, fmt.Errorf("ThemeRepository.GetThemeCurator: build query: %w", err)
	}

	var curatorIDBytes pgtype.UUID
	err = r.DB.QueryRow(ctx, query, args...).Scan(&curatorIDBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ports.ErrThemeNotFound
		}
		return uuid.Nil, fmt.Errorf("ThemeRepository.GetThemeCurator: %w", err)
	}

	return uuid.UUID(curatorIDBytes.Bytes), nil
}

// ListThemesByCurator retrieves theme summaries by a specific curator
func (r *ThemeRepository) ListThemesByCurator(ctx context.Context, curatorID uuid.UUID) ([]*ports.ThemeSummary, error) {
	filter := ports.ListFilter{
		CuratorID: &curatorID,
	}
	return r.ListThemes(ctx, filter)
}

// Helper functions

// syncArticles performs the diff and sync operation for theme articles
func (r *ThemeRepository) syncArticles(ctx context.Context, themeID uuid.UUID, desiredArticles []*domain.ThemeArticle) error {
	// Step 1: Get current state from database
	query, args, err := r.SB.
		Select("post_id", "position", "added_by", "added_at", "updated_at").
		From("theme_articles").
		Where(sq.Eq{"theme_id": pgtype.UUID{Bytes: themeID, Valid: true}}).
		ToSql()

	if err != nil {
		return fmt.Errorf("syncArticles: build select query: %w", err)
	}

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("syncArticles: query current articles: %w", err)
	}
	defer rows.Close()

	// Build map of current articles
	type articleData struct {
		position  int
		addedBy   uuid.UUID
		addedAt   time.Time
		updatedAt time.Time
	}
	currentArticles := make(map[uuid.UUID]articleData)

	for rows.Next() {
		var postIDBytes, addedByBytes pgtype.UUID
		var data articleData

		err := rows.Scan(&postIDBytes, &data.position, &addedByBytes, &data.addedAt, &data.updatedAt)
		if err != nil {
			return fmt.Errorf("syncArticles: scan current article: %w", err)
		}

		postID := uuid.UUID(postIDBytes.Bytes)
		data.addedBy = uuid.UUID(addedByBytes.Bytes)
		currentArticles[postID] = data
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("syncArticles: rows error: %w", err)
	}

	// Step 2: Build map of desired articles
	desiredMap := make(map[uuid.UUID]*domain.ThemeArticle)
	for _, article := range desiredArticles {
		desiredMap[article.PostID] = article
	}

	// Step 3: Calculate differences and prepare batch operations
	batch := &pgx.Batch{}

	// Delete articles that are no longer in the theme
	for postID := range currentArticles {
		if _, exists := desiredMap[postID]; !exists {
			delQuery, delArgs, err := r.SB.
				Delete("theme_articles").
				Where(sq.Eq{
					"theme_id": pgtype.UUID{Bytes: themeID, Valid: true},
					"post_id":  pgtype.UUID{Bytes: postID, Valid: true},
				}).
				ToSql()
			if err != nil {
				return fmt.Errorf("syncArticles: build delete query: %w", err)
			}
			batch.Queue(delQuery, delArgs...)
		}
	}

	// Insert new articles and update existing ones
	for postID, article := range desiredMap {
		current, exists := currentArticles[postID]
		
		if !exists {
			// Insert new article
			insQuery, insArgs, err := r.SB.
				Insert("theme_articles").
				Columns("theme_id", "post_id", "position", "added_by", "added_at", "updated_at").
				Values(
					pgtype.UUID{Bytes: themeID, Valid: true},
					pgtype.UUID{Bytes: article.PostID, Valid: true},
					article.Position,
					pgtype.UUID{Bytes: article.AddedBy, Valid: true},
					pgtype.Timestamptz{Time: article.AddedAt, Valid: true},
					pgtype.Timestamptz{Time: article.UpdatedAt, Valid: true},
				).
				ToSql()
			if err != nil {
				return fmt.Errorf("syncArticles: build insert query: %w", err)
			}
			batch.Queue(insQuery, insArgs...)
		} else if current.position != article.Position {
			// Update position if it changed
			updQuery, updArgs, err := r.SB.
				Update("theme_articles").
				Set("position", article.Position).
				Set("updated_at", pgtype.Timestamptz{Time: article.UpdatedAt, Valid: true}).
				Where(sq.Eq{
					"theme_id": pgtype.UUID{Bytes: themeID, Valid: true},
					"post_id":  pgtype.UUID{Bytes: article.PostID, Valid: true},
				}).
				ToSql()
			if err != nil {
				return fmt.Errorf("syncArticles: build update query: %w", err)
			}
			batch.Queue(updQuery, updArgs...)
		}
	}

	// Step 4: Execute the batch if there are any operations
	if batch.Len() > 0 {
		results := r.DB.SendBatch(ctx, batch)
		defer func() { _ = results.Close() }()

		for i := 0; i < batch.Len(); i++ {
			_, err := results.Exec()
			if err != nil {
				return fmt.Errorf("syncArticles: execute batch operation %d: %w", i, err)
			}
		}
	}

	return nil
}

// applyThemeFilters applies common WHERE clauses to a query builder
func (r *ThemeRepository) applyThemeFilters(qb sq.SelectBuilder, filter ports.ListFilter) sq.SelectBuilder {
	if filter.CuratorID != nil {
		qb = qb.Where(sq.Eq{"t.curator_id": pgtype.UUID{Bytes: *filter.CuratorID, Valid: true}})
	}

	if filter.IsActive != nil {
		qb = qb.Where(sq.Eq{"t.is_active": *filter.IsActive})
	}

	return qb
}

// Helper functions

// scanTheme scans a single theme from pgx.Row
func scanTheme(row pgx.Row) (*domain.Theme, error) {
	var theme domain.Theme
	var idBytes, curatorIDBytes pgtype.UUID

	err := row.Scan(
		&idBytes,
		&theme.Name,
		&theme.Description,
		&theme.Slug,
		&curatorIDBytes,
		&theme.IsActive,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanTheme: %w", err)
	}

	// Convert pgtype values
	theme.ID = uuid.UUID(idBytes.Bytes)
	theme.CuratorID = uuid.UUID(curatorIDBytes.Bytes)

	// Initialize empty Articles slice
	theme.Articles = make([]*domain.ThemeArticle, 0)

	return &theme, nil
}


// scanThemeSummaryFromRows scans a theme summary from pgx.Rows
func scanThemeSummaryFromRows(rows pgx.Rows) (*ports.ThemeSummary, error) {
	var summary ports.ThemeSummary
	var idBytes, curatorIDBytes pgtype.UUID
	var curatorName pgtype.Text

	err := rows.Scan(
		&idBytes,
		&summary.Name,
		&summary.Description,
		&summary.Slug,
		&curatorIDBytes,
		&curatorName,
		&summary.IsActive,
		&summary.CreatedAt,
		&summary.UpdatedAt,
		&summary.ArticleCount,
	)
	if err != nil {
		return nil, fmt.Errorf("scanThemeSummaryFromRows: %w", err)
	}

	// Convert pgtype values
	summary.ID = uuid.UUID(idBytes.Bytes)
	summary.CuratorID = uuid.UUID(curatorIDBytes.Bytes)

	if curatorName.Valid {
		summary.CuratorName = curatorName.String
	}

	return &summary, nil
}

// Compile-time check to ensure ThemeRepository implements ports.ThemeRepository
var _ ports.ThemeRepository = (*ThemeRepository)(nil)