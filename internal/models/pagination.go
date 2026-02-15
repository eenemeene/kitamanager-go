package models

import "fmt"

// PaginationParams holds pagination request parameters
type PaginationParams struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// Validate checks for explicitly invalid pagination values.
// Returns an error if page or limit are negative, or if limit exceeds 100.
// Zero values (Go default for int from query binding) are allowed as they indicate
// "not provided" and will be defaulted by SetDefaults() (page=1, limit=20).
func (p *PaginationParams) Validate() error {
	if p.Page < 0 {
		return fmt.Errorf("page must be positive")
	}
	if p.Limit < 0 {
		return fmt.Errorf("limit must be positive")
	}
	if p.Limit > 100 {
		return fmt.Errorf("limit must not exceed 100")
	}
	return nil
}

// SetDefaults sets default values for pagination parameters
func (p *PaginationParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
}

// Offset calculates the offset for database queries
func (p *PaginationParams) Offset() int {
	return (p.Page - 1) * p.Limit
}

// PaginationLinks contains HATEOAS links for pagination
type PaginationLinks struct {
	Self  string  `json:"self" example:"/api/v1/users?page=2&limit=20"`
	First string  `json:"first" example:"/api/v1/users?page=1&limit=20"`
	Last  string  `json:"last" example:"/api/v1/users?page=5&limit=20"`
	Prev  *string `json:"prev,omitempty" example:"/api/v1/users?page=1&limit=20"`
	Next  *string `json:"next,omitempty" example:"/api/v1/users?page=3&limit=20"`
}

// PaginatedResponse wraps list responses with pagination metadata
type PaginatedResponse[T any] struct {
	Data       []T              `json:"data"`
	Page       int              `json:"page" example:"1"`
	Limit      int              `json:"limit" example:"20"`
	Total      int64            `json:"total" example:"100"`
	TotalPages int              `json:"total_pages" example:"5"`
	Links      *PaginationLinks `json:"_links,omitempty"`
}

// NewPaginatedResponse creates a new paginated response (without links for backwards compatibility)
func NewPaginatedResponse[T any](data []T, page, limit int, total int64) PaginatedResponse[T] {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return PaginatedResponse[T]{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// NewPaginatedResponseWithLinks creates a new paginated response with HATEOAS links
func NewPaginatedResponseWithLinks[T any](data []T, page, limit int, total int64, basePath string) PaginatedResponse[T] {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	links := &PaginationLinks{
		Self:  fmt.Sprintf("%s?page=%d&limit=%d", basePath, page, limit),
		First: fmt.Sprintf("%s?page=1&limit=%d", basePath, limit),
		Last:  fmt.Sprintf("%s?page=%d&limit=%d", basePath, totalPages, limit),
	}

	// Add prev link if not on first page
	if page > 1 {
		prev := fmt.Sprintf("%s?page=%d&limit=%d", basePath, page-1, limit)
		links.Prev = &prev
	}

	// Add next link if not on last page
	if page < totalPages {
		next := fmt.Sprintf("%s?page=%d&limit=%d", basePath, page+1, limit)
		links.Next = &next
	}

	return PaginatedResponse[T]{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		Links:      links,
	}
}
