package domain

import (
	"errors"
	"regexp"
	"time"
)

var (
	ErrInvalidUsername  = errors.New("invalid username format")
	ErrUsernameTooShort = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong  = errors.New("username must not exceed 30 characters")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrEmptySupabaseID  = errors.New("supabase ID cannot be empty")
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type User struct {
	ID          string
	SupabaseID  string
	Email       string
	Username    string
	DisplayName string
	Bio         string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewUser(supabaseID, email, username string) (*User, error) {
	if err := validateSupabaseID(supabaseID); err != nil {
		return nil, err
	}

	if err := validateEmail(email); err != nil {
		return nil, err
	}

	if err := validateUsername(username); err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		SupabaseID: supabaseID,
		Email:      email,
		Username:   username,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (u *User) UpdateProfile(displayName, bio, avatarURL string) {
	if displayName != "" {
		u.DisplayName = displayName
	}
	if bio != "" {
		u.Bio = bio
	}
	if avatarURL != "" {
		u.AvatarURL = avatarURL
	}
	u.UpdatedAt = time.Now()
}

func validateSupabaseID(id string) error {
	if id == "" {
		return ErrEmptySupabaseID
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	// Basic email validation - Supabase already validates this
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func validateUsername(username string) error {
	if len(username) < 3 {
		return ErrUsernameTooShort
	}
	if len(username) > 30 {
		return ErrUsernameTooLong
	}
	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}