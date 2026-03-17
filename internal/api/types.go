package api

type SearchRequest struct {
	Query    string        `json:"query"`
	PageSize int           `json:"pageSize,omitempty"`
	Filter   *SearchFilter `json:"filter,omitempty"`
}

type SearchFilter struct {
	Version  string `json:"version,omitempty"`
	Language string `json:"language,omitempty"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Path    string `json:"path"`
	Section string `json:"section,omitempty"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}
