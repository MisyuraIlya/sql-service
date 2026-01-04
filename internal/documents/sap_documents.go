package documents

import "time"

type SapDocumentsQuery struct {
	CardCode      *string
	DateFrom      time.Time
	DateTo        time.Time
	WarehouseCode *string
	DocStatus     *string
	SortBy        string
	SortDir       string
	Page          int
	PageSize      int
}

type SapDocumentsResponse struct {
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	Total    int              `json:"total"`
	Items    []map[string]any `json:"items"`
}
