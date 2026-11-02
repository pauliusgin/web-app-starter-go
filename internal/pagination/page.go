// Package pagination describes the page of a list a caller asks for and where that page sits in the whole list.
package pagination

// PageRequest is a 1-based page of a list, holding at most Limit items.
type PageRequest struct {
	Page  int
	Limit int
}

// Offset is how many items precede the requested page.
func (p PageRequest) Offset() int {
	return (p.Page - 1) * p.Limit
}

// PageMetadata places a returned page within the whole list of Total items.
type PageMetadata struct {
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

// NewPageMetadata describes the requested page of a list holding total items; an empty list has zero pages.
func NewPageMetadata(request PageRequest, total int) PageMetadata {
	totalPages := (total + request.Limit - 1) / request.Limit

	return PageMetadata{
		Total:      total,
		Page:       request.Page,
		Limit:      request.Limit,
		TotalPages: totalPages,
	}
}
