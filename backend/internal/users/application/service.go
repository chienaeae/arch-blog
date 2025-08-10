package application

import (
	"context"
	"net/http"

	"github.com/philly/arch-blog/backend/internal/platform/apperror"
	"github.com/philly/arch-blog/backend/internal/users/domain"
	"github.com/philly/arch-blog/backend/internal/users/ports"
)

var (
	ErrUserNotFound = apperror.New(
		apperror.CodeNotFound,
		apperror.BusinessCodeUserNotFound,
		"user not found",
		http.StatusNotFound,
	)
	ErrEmailAlreadyExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeEmailExists,
		"email already registered",
		http.StatusConflict,
	)
	ErrUsernameAlreadyExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeUsernameExists,
		"username already taken",
		http.StatusConflict,
	)
	ErrSupabaseIDAlreadyExists = apperror.New(
		apperror.CodeConflict,
		apperror.BusinessCodeSupabaseIDExists,
		"user with this Supabase ID already exists",
		http.StatusConflict,
	)
	ErrMissingSupabaseID = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeMissingRequiredField,
		"supabase ID is required",
		http.StatusBadRequest,
	).WithDetails(map[string]string{"field": "supabase_id"})
	ErrMissingEmail = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeMissingRequiredField,
		"email is required",
		http.StatusBadRequest,
	).WithDetails(map[string]string{"field": "email"})
	ErrMissingUsername = apperror.New(
		apperror.CodeValidationFailed,
		apperror.BusinessCodeMissingRequiredField,
		"username is required",
		http.StatusBadRequest,
	).WithDetails(map[string]string{"field": "username"})
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
		return nil, ErrMissingSupabaseID
	}
	if params.Email == "" {
		return nil, ErrMissingEmail
	}
	if params.Username == "" {
		return nil, ErrMissingUsername
	}
	// Check if user already exists with this Supabase ID
	existingUser, err := s.repo.FindBySupabaseID(ctx, params.SupabaseID)
	if err == nil && existingUser != nil {
		return nil, ErrSupabaseIDAlreadyExists
	}

	// Check if username is already taken
	exists, err := s.repo.ExistsByUsername(ctx, params.Username)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to check username availability", http.StatusInternalServerError)
	}
	if exists {
		return nil, ErrUsernameAlreadyExists
	}

	// Check if email is already registered
	exists, err = s.repo.ExistsByEmail(ctx, params.Email)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to check email availability", http.StatusInternalServerError)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Create new user domain object
	user, err := domain.NewUser(params.SupabaseID, params.Email, params.Username)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeValidationFailed, apperror.BusinessCodeInvalidFormat,
			"failed to create user", http.StatusBadRequest)
	}

	// Set optional fields
	user.UpdateProfile(params.DisplayName, params.Bio, params.AvatarURL)

	// Persist to repository
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to save user", http.StatusInternalServerError)
	}

	return user, nil
}

func (s *UserService) GetUserBySupabaseID(ctx context.Context, supabaseID string) (*domain.User, error) {
	user, err := s.repo.FindBySupabaseID(ctx, supabaseID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to find user", http.StatusInternalServerError)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to find user", http.StatusInternalServerError)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) UpdateUserProfile(ctx context.Context, params UpdateUserParams) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, params.UserID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to find user", http.StatusInternalServerError)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.UpdateProfile(params.DisplayName, params.Bio, params.AvatarURL)

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(err, apperror.CodeInternalError, apperror.BusinessCodeGeneral,
			"failed to update user", http.StatusInternalServerError)
	}

	return user, nil
}