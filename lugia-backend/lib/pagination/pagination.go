package pagination

import (
	"net/http"
	"strconv"

	"lugia/lib/conversions"
)

type PaginationMetadata struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type QueryParams struct {
	Page   int
	Limit  int32
	Offset int32
}

func CalculatePagination(r *http.Request) QueryParams {
	page := 1
	limit := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
	}

	limit32, err := conversions.SafeInt32(limit)
	if err != nil {
		// Fallback to safe default if conversion fails
		limit32 = 50
	}

	offset32, err := conversions.SafeInt32((page - 1) * limit)
	if err != nil {
		// Fallback to safe default if conversion fails
		offset32 = 0
	}

	return QueryParams{
		Page:   page,
		Limit:  limit32,
		Offset: offset32,
	}
}

func CalculateMetadata(page int, limit int32, total int64) PaginationMetadata {
	localLimit := int(limit)
	localTotal := int(total)
	totalPages := (localTotal + localLimit - 1) / localLimit
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationMetadata{
		Page:       page,
		Limit:      localLimit,
		Total:      localTotal,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}
