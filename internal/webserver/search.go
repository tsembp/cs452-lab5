
package webserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
)

// RootHandler serves HTML content loaded from a file
func (s *WebServer) RootHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.htmlPath)
}

// SearchHandler handles GET requests to the `/api/search` endpoint.
// It expects a query parameter `?q=...` containing space-separated keywords.
func (s *WebServer) SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	keywords := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})

	results, err := s.index.SearchInMemoryTopK(keywords, s.topK)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
