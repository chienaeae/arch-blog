package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/philly/arch-blog/backend/internal/users/domain"
	"github.com/philly/arch-blog/backend/internal/users/ports"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) ports.UserRepository {
	return &UserRepository{
		pool: pool,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, supabase_id, email, username, display_name, bio, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	id := uuid.New()
	user.ID = id.String()

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.SupabaseID,
		user.Email,
		user.Username,
		nullString(user.DisplayName),
		nullString(user.Bio),
		nullString(user.AvatarURL),
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, supabase_id, email, username, display_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var displayName, bio, avatarURL *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.SupabaseID,
		&user.Email,
		&user.Username,
		&displayName,
		&bio,
		&avatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	user.DisplayName = stringValue(displayName)
	user.Bio = stringValue(bio)
	user.AvatarURL = stringValue(avatarURL)

	return &user, nil
}

func (r *UserRepository) FindBySupabaseID(ctx context.Context, supabaseID string) (*domain.User, error) {
	query := `
		SELECT id, supabase_id, email, username, display_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE supabase_id = $1
	`

	var user domain.User
	var displayName, bio, avatarURL *string

	err := r.pool.QueryRow(ctx, query, supabaseID).Scan(
		&user.ID,
		&user.SupabaseID,
		&user.Email,
		&user.Username,
		&displayName,
		&bio,
		&avatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by Supabase ID: %w", err)
	}

	user.DisplayName = stringValue(displayName)
	user.Bio = stringValue(bio)
	user.AvatarURL = stringValue(avatarURL)

	return &user, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, supabase_id, email, username, display_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user domain.User
	var displayName, bio, avatarURL *string

	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.SupabaseID,
		&user.Email,
		&user.Username,
		&displayName,
		&bio,
		&avatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	user.DisplayName = stringValue(displayName)
	user.Bio = stringValue(bio)
	user.AvatarURL = stringValue(avatarURL)

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, supabase_id, email, username, display_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var displayName, bio, avatarURL *string

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.SupabaseID,
		&user.Email,
		&user.Username,
		&displayName,
		&bio,
		&avatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	user.DisplayName = stringValue(displayName)
	user.Bio = stringValue(bio)
	user.AvatarURL = stringValue(avatarURL)

	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET display_name = $2, bio = $3, avatar_url = $4, updated_at = $5
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		nullString(user.DisplayName),
		nullString(user.Bio),
		nullString(user.AvatarURL),
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// Helper functions for handling null values
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
