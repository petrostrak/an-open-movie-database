package data

import (
	"strings"

	"github.com/petrostrak/an-open-movie-database/internal/validator"
)

// Page, PageSize and Sort query string parameters.
//
// Add a SortSafelist field to hold the supported sort values.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page and page_size parameters contain sensible values.
	v.Check(f.Page > 0, "page", "must be greater that zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum that 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	// Check that the sort parameter matches a value in the safelist.
	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// Check that the client-provided Sort field matches on of the entries in our safelist
// and if it does, extract the column name from the Sort field by stripping the leading
// hyphen character (if one exists).
func (f Filters) sortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}

	panic("unsafe sort parameter: " + f.Sort)
}

// Return the sort direction ("ASC" or "DESC") depending on the prefix character of the
// Sort field.
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

// Helper method that returns the page size
func (f Filters) limit() int {
	return f.PageSize
}

// Helper method that returns the offset (specific number of rows to be skipped before
// starting to return records from the query.)
//
// There is the theoretical risk of an integer overflow as we are multiplying two int values
// together. However, this is mitigated by the validation rules we created in our ValidateFilters()
// function, where we enforced maximum values of page_size=100 and page=10000000
func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}
