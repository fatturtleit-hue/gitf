package handlers

// Pagination holds standard pagination metadata returned alongside list responses.
// The frontend depends on these field names; renaming is a breaking change.
type Pagination struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// NewPagination computes pagination metadata from raw request parameters.
// Caller-supplied bad values (page<1, perPage out of range) are clamped to
// safe defaults; this is the single source of truth for pagination semantics.
func NewPagination(page, perPage, total int) Pagination {
	page, perPage = clampPage(page, perPage)
	totalPages := (total + perPage - 1) / perPage
	if total == 0 {
		totalPages = 0
	}
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// clampPage normalizes pagination input. Exposed (lowercased) for handler reuse
// so list handlers don't re-implement the same clamping logic.
func clampPage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	return page, perPage
}
