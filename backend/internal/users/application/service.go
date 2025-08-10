package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/philly/arch-blog/backend/internal/users/domain"
	"github.com/philly/arch-blog/backend/internal/users/ports"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrValidationFailed  = errors.New("validation failed")
)

// CreateUserParams contains all parameters needed to create a new user
type CreateUserParams struct {
	SupabaseID  string
	Email       string
	Username    string
	DisplayName string
	Bio         string
	AvatarURL   string
}

// UpdateUserParams contains parameters for updating user profile
type UpdateUserParams struct {
	UserID      string
	DisplayName string
	Bio         string
	AvatarURL   string
}

type UserService struct {
	repo ports.UserRepository
}

func NewUserService(repo ports.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, params CreateUserParams) (*domain.User, error) {
	// Validate required fields
	if params.SupabaseID == "" {
		return nil, fmt.Errorf("%w: supabase ID is required", ErrValidationFailed)
	}
	if params.Email == "" {
		return nil, fmt.Errorf("%w: email is required", ErrValidationFailed)
	}
	if params.Username == "" {
		return nil, fmt.Errorf("%w: username is required", ErrValidationFailed)
	}
	// Check if user already exists with this Supabase ID
	existingUser, err := s.repo.FindBySupabaseID(ctx, params.SupabaseID)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with Supabase ID already exists: %w", ErrUserAlreadyExists)
	}

	// Check if username is already taken
	exists, err := s.repo.ExistsByUsername(ctx, params.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username availability: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("username already taken: %w", ErrUserAlreadyExists)
	}

	// Check if email is already registered
	exists, err = s.repo.ExistsByEmail(ctx, params.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email availability: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already registered: %w", ErrUserAlreadyExists)
	}

	// Create new user domain object
	user, err := domain.NewUser(params.SupabaseID, params.Email, params.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Set optional fields
	user.UpdateProfile(params.DisplayName, params.Bio, params.AvatarURL)

	// Persist to repository
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

func (s *UserService) GetUserBySupabaseID(ctx context.Context, supabaseID string) (*domain.User, error) {
	user, err := s.repo.FindBySupabaseID(ctx, supabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) UpdateUserProfile(ctx context.Context, params UpdateUserParams) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, params.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.UpdateProfile(params.DisplayName, params.Bio, params.AvatarURL)

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}