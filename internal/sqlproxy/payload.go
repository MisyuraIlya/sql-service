package sqlproxy

type DBConnDTO struct {
	Server   string `json:"server"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type QueryRequest struct {
	DBName    string         `json:"dbName"`
	DB        DBConnDTO      `json:"db"`
	Query     string         `json:"query"`
	Params    map[string]any `json:"params"`
	TimeoutMs int            `json:"timeoutMs,omitempty"`
}

type ResultSet struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

type QueryResponse struct {
	DBName      string      `json:"dbName"`
	DurationMs  int64       `json:"durationMs"`
	ResultSets  []ResultSet `json:"resultSets"`
	RowsTotal   int         `json:"rowsTotal"`
	WarningNote string      `json:"warningNote,omitempty"`
}
