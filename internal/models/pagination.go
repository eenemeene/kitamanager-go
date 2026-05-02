package models

import (
	"fmt"
	"net/url"
)

// MaxPage caps how high the page index can go. Closes audit finding
// R-M-2 (security review 2026-05-01): Limit was capped at 100 but
// Page had no ceiling, so a request like ?page=2147483640&limit=100
// forced PostgreSQL to compute OFFSET ~= 2.1e11 — a CPU-amplification
// DoS where the attacker pays a query string and the DB pays a full
// index scan. 100_000 is far past any realistic legitimate paging
// (a user clicking "next" 100k times) but well below the row-count
// at which OFFSET starts to bite.
const MaxPage = 100_000

// PaginationParams holds pagination request parameters
type PaginationParams struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

// Validate checks for explicitly invalid pagination values.
// Returns an error if page or limit are negative, if limit exceeds
// 100, or if page exceeds MaxPage. Zero values (Go default for int
// from query binding) are allowed as they indicate "not provided"
// and will be defaulted by SetDefaults() (page=1, limit=20).
func (p *PaginationParams) Validate() error {
	if p.Page < 0 {
		return fmt.Errorf("page must be positive")
	}
	if p.Page > MaxPage {
		return fmt.Errorf("page must not exceed %d", MaxPage)
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

// NewPaginatedResponseWithLinks creates a new paginated response with HATEOAS links.
// It preserves any filter query parameters (search, section_id, etc.) from rawQuery
// in the generated links, replacing only page and limit.
func NewPaginatedResponseWithLinks[T any](data []T, page, limit int, total int64, basePath string, rawQuery string) PaginatedResponse[T] {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Parse existing query params and strip pagination keys
	filterParams := stripPaginationParams(rawQuery)

	links := &PaginationLinks{
		Self:  buildPaginatedURL(basePath, filterParams, page, limit),
		First: buildPaginatedURL(basePath, filterParams, 1, limit),
		Last:  buildPaginatedURL(basePath, filterParams, totalPages, limit),
	}

	// Add prev link if not on first page
	if page > 1 {
		prev := buildPaginatedURL(basePath, filterParams, page-1, limit)
		links.Prev = &prev
	}

	// Add next link if not on last page
	if page < totalPages {
		next := buildPaginatedURL(basePath, filterParams, page+1, limit)
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

// stripPaginationParams removes page and limit from a raw query string
// and returns the remaining parameters as url.Values.
func stripPaginationParams(rawQuery string) url.Values {
	params, _ := url.ParseQuery(rawQuery)
	params.Del("page")
	params.Del("limit")
	return params
}

// buildPaginatedURL constructs a URL with filter params plus page/limit.
func buildPaginatedURL(basePath string, filterParams url.Values, page, limit int) string {
	q := make(url.Values, len(filterParams)+2)
	for k, v := range filterParams {
		q[k] = v
	}
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	return basePath + "?" + q.Encode()
}
