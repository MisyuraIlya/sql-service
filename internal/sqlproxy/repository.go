package sqlproxy

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
	"unicode/utf8"

	_ "github.com/denisenkom/go-mssqldb"
)

type Repository struct {
}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) buildConnString(db DBConnDTO) (string, error) {
	if db.Server == "" || db.Database == "" || db.User == "" {
		return "", fmt.Errorf("db.server, db.database, db.user are required")
	}

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(db.User, db.Password),
		Host:   db.Server,
	}

	q := url.Values{}
	q.Set("database", db.Database)

	q.Set("encrypt", "disable")

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (r *Repository) openDB(ctx context.Context, dbCfg DBConnDTO) (*sql.DB, error) {
	connStr, err := r.buildConnString(dbCfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func normalizeJSONNumber(v any) any {
	switch x := v.(type) {
	case float64:
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	default:
		return v
	}
}

func toNamedArgs(params map[string]any) ([]any, error) {
	if params == nil {
		return nil, nil
	}
	args := make([]any, 0, len(params))
	for k, v := range params {
		if err := ValidateParamName(k); err != nil {
			return nil, err
		}
		args = append(args, sql.Named(k, normalizeJSONNumber(v)))
	}
	return args, nil
}

func anyToJSONSafe(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		if utf8.Valid(x) {
			return string(x)
		}
		return map[string]any{
			"type":   "bytes",
			"base64": base64.StdEncoding.EncodeToString(x),
		}
	case time.Time:
		return x.Format(time.RFC3339Nano)
	default:
		return x
	}
}

func scanResultSets(rows *sql.Rows) ([]ResultSet, int, error) {
	totalRows := 0
	var resultSets []ResultSet

	for {
		cols, err := rows.Columns()
		if err != nil {
			return nil, 0, err
		}

		rs := ResultSet{
			Columns: cols,
			Rows:    make([]map[string]any, 0, 64),
		}

		for rows.Next() {
			totalRows++

			raw := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range raw {
				ptrs[i] = &raw[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, 0, err
			}

			rowMap := make(map[string]any, len(cols))
			for i, c := range cols {
				rowMap[c] = anyToJSONSafe(raw[i])
			}
			rs.Rows = append(rs.Rows, rowMap)
		}

		if err := rows.Err(); err != nil {
			return nil, 0, err
		}

		resultSets = append(resultSets, rs)

		if !rows.NextResultSet() {
			break
		}
	}

	return resultSets, totalRows, nil
}

func (r *Repository) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	args, err := toNamedArgs(req.Params)
	if err != nil {
		return nil, err
	}

	db, err := r.openDB(ctx, req.DB)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	start := time.Now()
	rows, err := db.QueryContext(ctx, req.Query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resultSets, totalRows, err := scanResultSets(rows)
	if err != nil {
		return nil, err
	}

	return &QueryResponse{
		DBName:     req.DBName,
		DurationMs: time.Since(start).Milliseconds(),
		ResultSets: resultSets,
		RowsTotal:  totalRows,
	}, nil
}
