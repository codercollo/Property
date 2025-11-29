package data

import (
	"math"
	"strings"

	"github.com/codercollo/property/backend/internal/validator"
)

// A new Metadata struct for holding the pagination metadata
type Metadata struct {
	CurrentPage   int `json:"current_page,omitempty"`
	PageSize      int `json:"page_size,omitempty"`
	FirstPage     int `json:"first_page,omitempty"`
	LastPage      int `json:"last_page,omitempty"`
	TotalListings int `json:"total_listings,omitempty"`
}

// Filters holds pagination and sorting info for property queries
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

// Returns the column name to sort by if its's in the safelist
func (f Filters) sortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}

	}
	panic("unsafe sort parameter: " + f.Sort)
}

// Returns the sort direction ("ASC" or "DESC") based on the '-' prefix
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

// limit returns the number of records to fetch (PageSize)
func (f Filters) limit() int {
	return f.PageSize
}

// offset returns the number od records to skip based on the current page
func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

// calculateMetadata returns pagination info given total listings, page and pageSize
// LastPage is rounded up using math.Ceil, Returns empty Metadata of no listing
func calculateMetadata(totalListings, page, pageSize int) Metadata {
	if totalListings == 0 {
		return Metadata{}
	}

	return Metadata{
		CurrentPage:   page,
		PageSize:      pageSize,
		FirstPage:     1,
		LastPage:      int(math.Ceil(float64(totalListings) / float64(pageSize))),
		TotalListings: totalListings,
	}
}

// ValidateFilters checks that pagination and sorting parameters are valid
func ValidateFilters(v *validator.Validator, f Filters) {
	//Page must be . 0 and <= 10 million
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")

	//PageSize must be > 0 and <= 100
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	// Sort must be one of the allowed values
	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}
