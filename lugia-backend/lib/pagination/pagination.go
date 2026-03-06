package pagination

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
