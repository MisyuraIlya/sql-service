package sqlproxy

func Flatten(out *QueryResponse) *FlatQueryResponse {
	if out == nil {
		return &FlatQueryResponse{
			Rows: make([]map[string]any, 0),
		}
	}

	rows := make([]map[string]any, 0)
	if len(out.ResultSets) > 0 && out.ResultSets[0].Rows != nil {
		rows = out.ResultSets[0].Rows
	}

	warn := out.WarningNote
	if warn == "" && len(out.ResultSets) > 1 {
		warn = "query returned multiple result sets; only the first result set is returned in 'rows'"
	}

	return &FlatQueryResponse{
		DBName:      out.DBName,
		DurationMs:  out.DurationMs,
		RowsTotal:   out.RowsTotal,
		Rows:        rows,
		WarningNote: warn,
	}
}
