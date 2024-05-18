package data

import (
	"math"
	"strings"

	"github.com/yousifsabah0/blackbox/internal/validator"
)

type MetaData struct {
	CurrentPage int `json:"current_page"`
	PageSize    int `json:"page_size"`
	FirstPage   int `json:"first_page"`
	LastPage    int `json:"last_page"`
	Total       int `json:"total"`
}

func calculateMetaData(total, page, pageSize int) MetaData {
	if total == 0 {
		return MetaData{}
	}

	return MetaData{
		CurrentPage: page,
		PageSize:    pageSize,
		FirstPage:   1,
		LastPage:    int(math.Ceil(float64(total) / float64(pageSize))),
		Total:       total,
	}

}

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

func (f Filters) sortColumn() string {
	for _, value := range f.SortSafeList {
		if f.Sort == value {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	panic("unsafe sort parameter: " + f.Sort)
}

func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "page", "must be greater than 0")
	v.Check(f.Page < 10_000_000, "page", "must be less than 10_000_000")

	v.Check(f.Page > 0, "page_size", "must be greater than 0")
	v.Check(f.Page < 100, "page_size", "must be less than 100")

	v.Check(validator.In(f.Sort, f.SortSafeList...), "sort", "invalid sort value")
}
