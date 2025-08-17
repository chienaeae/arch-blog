package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Slug validation errors
var (
	ErrInvalidSlugFormat = errors.New("slug must contain only lowercase letters, numbers, and hyphens")
	ErrSlugEmpty         = errors.New("slug cannot be empty")
	ErrSlugTooLong       = errors.New("slug is too long")
)

// Compile regex patterns once at package level for performance
var (
	slugValidationRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
	slugReplaceRegex    = regexp.MustCompile(`[^a-z0-9-]+`)
	slugCollapseRegex   = regexp.MustCompile(`-+`)
)

// ValidateSlugFormat checks if a slug has valid format
func ValidateSlugFormat(slug string, maxLength int) error {
	if slug == "" {
		return ErrSlugEmpty
	}

	if len(slug) > maxLength {
		return ErrSlugTooLong
	}

	if !slugValidationRegex.MatchString(slug) {
		return ErrInvalidSlugFormat
	}

	return nil
}

// GenerateSlug creates a URL-friendly slug from a text string
func GenerateSlug(text string, maxLength int) string {
	// Convert to lowercase
	slug := strings.ToLower(text)

	// Replace spaces and special characters with hyphens
	slug = slugReplaceRegex.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Collapse multiple hyphens
	slug = slugCollapseRegex.ReplaceAllString(slug, "-")

	// Truncate if too long
	if len(slug) > maxLength {
		slug = slug[:maxLength]
		// Ensure we don't end with a hyphen after truncation
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// MakeSlugUnique appends a numeric suffix to make a slug unique
// It simply appends the suffix - truncation should be handled by the caller if needed
func MakeSlugUnique(baseSlug string, suffix int) string {
	if suffix <= 0 {
		return baseSlug
	}

	return fmt.Sprintf("%s-%d", baseSlug, suffix)
}

// MakeSlugUniqueWithMaxLength appends a suffix and ensures the result doesn't exceed maxLength
// This is a convenience function that handles both suffix and truncation
func MakeSlugUniqueWithMaxLength(baseSlug string, suffix int, maxLength int) string {
	if suffix <= 0 {
		if len(baseSlug) > maxLength {
			return baseSlug[:maxLength]
		}
		return baseSlug
	}

	suffixStr := "-" + strconv.Itoa(suffix)

	// If the base slug plus suffix would exceed max length, truncate the base
	if len(baseSlug)+len(suffixStr) > maxLength {
		maxBaseLength := maxLength - len(suffixStr)
		if maxBaseLength > 0 {
			baseSlug = baseSlug[:maxBaseLength]
			// Ensure we don't end with a hyphen after truncation
			baseSlug = strings.TrimRight(baseSlug, "-")
		}
	}

	return baseSlug + suffixStr
}
