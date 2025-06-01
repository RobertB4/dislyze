package search

import (
	"net/http"
	"strings"
)

func ValidateSearchTerm(r *http.Request, maxLength int) string {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	
	if len(search) > maxLength {
		search = search[:maxLength]
	}
	
	return search
}
