package documents

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type sapDocTable struct {
	DocType string
	Header  string
	Lines   string
}

type sapDocumentKey struct {
	DocType  string
	DocEntry int64
}

type sqlQuery struct {
	Query string
	Args  []any
}

var sapDocTables = []sapDocTable{
	{DocType: "Orders", Header: "ORDR", Lines: "RDR1"},
	{DocType: "Invoices", Header: "OINV", Lines: "INV1"},
	{DocType: "Returns", Header: "ORDN", Lines: "RDN1"},
	{DocType: "Quotations", Header: "OQUT", Lines: "QUT1"},
	{DocType: "DeliveryNotes", Header: "ODLN", Lines: "DLN1"},
}

var sapDocTableByType = func() map[string]sapDocTable {
	lookup := make(map[string]sapDocTable, len(sapDocTables))
	for _, entry := range sapDocTables {
		lookup[entry.DocType] = entry
	}
	return lookup
}()

func (r *DocumentRrepository) GetSapDocuments(ctx context.Context, query SapDocumentsQuery) (SapDocumentsResponse, error) {
	countQuery, keysQuery, err := buildSapDocumentsQueries(r.Db.Dialect, query)
	if err != nil {
		return SapDocumentsResponse{}, err
	}

	var total int
	if err := r.Db.QueryRowContext(ctx, countQuery.Query, countQuery.Args...).Scan(&total); err != nil {
		return SapDocumentsResponse{}, err
	}

	keysRows, err := r.Db.QueryContext(ctx, keysQuery.Query, keysQuery.Args...)
	if err != nil {
		return SapDocumentsResponse{}, err
	}
	defer keysRows.Close()

	keys := make([]sapDocumentKey, 0, query.PageSize)
	docEntriesByType := make(map[string][]int64)
	for keysRows.Next() {
		var (
			docType  string
			docEntry int64
			docDate  time.Time
		)
		if err := keysRows.Scan(&docType, &docEntry, &docDate); err != nil {
			return SapDocumentsResponse{}, err
		}
		keys = append(keys, sapDocumentKey{DocType: docType, DocEntry: docEntry})
		docEntriesByType[docType] = append(docEntriesByType[docType], docEntry)
	}
	if err := keysRows.Err(); err != nil {
		return SapDocumentsResponse{}, err
	}

	rowsByKey := make(map[string]map[string]any, len(keys))
	for docType, entries := range docEntriesByType {
		tableDef, ok := sapDocTableByType[docType]
		if !ok {
			return SapDocumentsResponse{}, fmt.Errorf("unsupported docType: %s", docType)
		}

		entries = uniqueInt64s(entries)
		if len(entries) == 0 {
			continue
		}

		selectQuery, selectArgs, err := buildSapDocumentsSelect(r.Db.Dialect, tableDef.Header, entries)
		if err != nil {
			return SapDocumentsResponse{}, err
		}

		rows, err := r.Db.QueryContext(ctx, selectQuery, selectArgs...)
		if err != nil {
			return SapDocumentsResponse{}, err
		}

		rowMap, scanErr := scanRowsByDocEntry(rows, docType)
		closeErr := rows.Close()
		if scanErr != nil {
			return SapDocumentsResponse{}, scanErr
		}
		if closeErr != nil {
			return SapDocumentsResponse{}, closeErr
		}

		for key, value := range rowMap {
			rowsByKey[key] = value
		}
	}

	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		mapKey := sapDocumentKeyString(key.DocType, key.DocEntry)
		if row, ok := rowsByKey[mapKey]; ok {
			items = append(items, row)
		}
	}

	return SapDocumentsResponse{
		Page:     query.Page,
		PageSize: query.PageSize,
		Total:    total,
		Items:    items,
	}, nil
}

func buildSapDocumentsQueries(dialect string, query SapDocumentsQuery) (sqlQuery, sqlQuery, error) {
	switch strings.ToLower(dialect) {
	case "", "mssql":
		return buildSapDocumentsQueriesMSSQL(query)
	case "hana":
		return buildSapDocumentsQueriesHANA(query)
	default:
		return sqlQuery{}, sqlQuery{}, fmt.Errorf("unsupported db dialect: %s", dialect)
	}
}

func buildSapDocumentsQueriesMSSQL(query SapDocumentsQuery) (sqlQuery, sqlQuery, error) {
	unionSQL := buildSapDocumentsUnionMSSQL()
	orderClause, err := buildSapDocumentsOrderBy(query.SortBy, query.SortDir)
	if err != nil {
		return sqlQuery{}, sqlQuery{}, err
	}

	baseArgs := []any{
		sql.Named("dateFrom", query.DateFrom),
		sql.Named("dateTo", query.DateTo),
		sql.Named("cardCode", optionalStringArg(query.CardCode)),
		sql.Named("docStatus", optionalStringArg(query.DocStatus)),
		sql.Named("warehouseCode", optionalStringArg(query.WarehouseCode)),
	}

	countQuery := sqlQuery{
		Query: fmt.Sprintf("SELECT COUNT(1) FROM (%s) AS k", unionSQL),
		Args:  baseArgs,
	}

	offset := (query.Page - 1) * query.PageSize
	keysQuery := sqlQuery{
		Query: fmt.Sprintf(
			"SELECT docType, DocEntry, DocDate FROM (%s) AS k ORDER BY %s OFFSET @offset ROWS FETCH NEXT @pageSize ROWS ONLY",
			unionSQL,
			orderClause,
		),
		Args: append(append([]any{}, baseArgs...), sql.Named("offset", offset), sql.Named("pageSize", query.PageSize)),
	}

	return countQuery, keysQuery, nil
}

func buildSapDocumentsQueriesHANA(query SapDocumentsQuery) (sqlQuery, sqlQuery, error) {
	unionSQL, baseArgs := buildSapDocumentsUnionHANA(query)
	orderClause, err := buildSapDocumentsOrderBy(query.SortBy, query.SortDir)
	if err != nil {
		return sqlQuery{}, sqlQuery{}, err
	}

	countQuery := sqlQuery{
		Query: fmt.Sprintf("SELECT COUNT(1) FROM (%s) AS k", unionSQL),
		Args:  baseArgs,
	}

	offset := (query.Page - 1) * query.PageSize
	keysArgs := append(append([]any{}, baseArgs...), query.PageSize, offset)
	keysQuery := sqlQuery{
		Query: fmt.Sprintf(
			"SELECT docType, DocEntry, DocDate FROM (%s) AS k ORDER BY %s LIMIT ? OFFSET ?",
			unionSQL,
			orderClause,
		),
		Args: keysArgs,
	}

	return countQuery, keysQuery, nil
}

func buildSapDocumentsUnionMSSQL() string {
	parts := make([]string, 0, len(sapDocTables))
	for _, entry := range sapDocTables {
		part := fmt.Sprintf(`SELECT '%s' AS docType, H.DocEntry, H.DocDate
FROM %s H
WHERE H.DocDate >= @dateFrom AND H.DocDate <= @dateTo
  AND (@cardCode IS NULL OR H.CardCode = @cardCode)
  AND (@docStatus IS NULL OR H.DocStatus = @docStatus)
  AND (@warehouseCode IS NULL OR EXISTS (
        SELECT 1 FROM %s L
        WHERE L.DocEntry = H.DocEntry AND L.WhsCode = @warehouseCode
      ))`, entry.DocType, entry.Header, entry.Lines)
		parts = append(parts, part)
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

func buildSapDocumentsUnionHANA(query SapDocumentsQuery) (string, []any) {
	parts := make([]string, 0, len(sapDocTables))
	args := make([]any, 0, len(sapDocTables)*8)

	cardCode := optionalStringArg(query.CardCode)
	docStatus := optionalStringArg(query.DocStatus)
	warehouseCode := optionalStringArg(query.WarehouseCode)

	for _, entry := range sapDocTables {
		part := fmt.Sprintf(`SELECT '%s' AS docType, H.DocEntry, H.DocDate
FROM %s H
WHERE H.DocDate >= ? AND H.DocDate <= ?
  AND (? IS NULL OR H.CardCode = ?)
  AND (? IS NULL OR H.DocStatus = ?)
  AND (? IS NULL OR EXISTS (
        SELECT 1 FROM %s L
        WHERE L.DocEntry = H.DocEntry AND L.WhsCode = ?
      ))`, entry.DocType, entry.Header, entry.Lines)
		parts = append(parts, part)
		args = append(args,
			query.DateFrom,
			query.DateTo,
			cardCode,
			cardCode,
			docStatus,
			docStatus,
			warehouseCode,
			warehouseCode,
		)
	}

	return strings.Join(parts, "\nUNION ALL\n"), args
}

func buildSapDocumentsOrderBy(sortBy, sortDir string) (string, error) {
	direction := strings.ToUpper(sortDir)
	switch strings.ToLower(sortDir) {
	case "asc", "desc":
	default:
		return "", fmt.Errorf("invalid sortDir")
	}

	column := "DocDate"
	switch strings.ToLower(sortBy) {
	case "", "docdate":
		column = "DocDate"
	case "docentry":
		column = "DocEntry"
	default:
		return "", fmt.Errorf("invalid sortBy")
	}

	tieBreaker := "DocEntry"
	if column == "DocEntry" {
		tieBreaker = "DocDate"
	}

	return fmt.Sprintf("%s %s, %s %s", column, direction, tieBreaker, direction), nil
}

func buildSapDocumentsSelect(dialect, table string, entries []int64) (string, []any, error) {
	if len(entries) == 0 {
		return "", nil, fmt.Errorf("empty DocEntry list for %s", table)
	}

	switch strings.ToLower(dialect) {
	case "", "mssql":
		placeholders := make([]string, len(entries))
		args := make([]any, 0, len(entries))
		for i, entry := range entries {
			name := fmt.Sprintf("docEntry%d", i)
			placeholders[i] = "@" + name
			args = append(args, sql.Named(name, entry))
		}
		return fmt.Sprintf("SELECT * FROM %s WHERE DocEntry IN (%s)", table, strings.Join(placeholders, ", ")), args, nil
	case "hana":
		placeholders := make([]string, len(entries))
		args := make([]any, 0, len(entries))
		for i, entry := range entries {
			placeholders[i] = "?"
			args = append(args, entry)
		}
		return fmt.Sprintf("SELECT * FROM %s WHERE DocEntry IN (%s)", table, strings.Join(placeholders, ", ")), args, nil
	default:
		return "", nil, fmt.Errorf("unsupported db dialect: %s", dialect)
	}
}

func scanRowsByDocEntry(rows *sql.Rows, docType string) (map[string]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	docEntryIndex := -1
	for i, column := range columns {
		if strings.EqualFold(column, "DocEntry") {
			docEntryIndex = i
			break
		}
	}
	if docEntryIndex == -1 {
		return nil, fmt.Errorf("DocEntry column not found for %s", docType)
	}

	result := make(map[string]map[string]any)
	for rows.Next() {
		raw := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]any, len(columns)+1)
		for i, column := range columns {
			rowMap[column] = normalizeSQLValue(raw[i])
		}
		rowMap["docType"] = docType

		key, ok := docEntryKeyFromValue(docType, raw[docEntryIndex])
		if !ok {
			return nil, fmt.Errorf("invalid DocEntry value for %s", docType)
		}
		result[key] = rowMap
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func normalizeSQLValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	default:
		return typed
	}
}

func docEntryKeyFromValue(docType string, value any) (string, bool) {
	switch typed := value.(type) {
	case int64:
		return sapDocumentKeyString(docType, typed), true
	case int32:
		return sapDocumentKeyString(docType, int64(typed)), true
	case int:
		return sapDocumentKeyString(docType, int64(typed)), true
	case float64:
		return sapDocumentKeyString(docType, int64(typed)), true
	case float32:
		return sapDocumentKeyString(docType, int64(typed)), true
	case []byte:
		return docEntryKeyFromString(docType, string(typed))
	case string:
		return docEntryKeyFromString(docType, typed)
	default:
		return "", false
	}
}

func docEntryKeyFromString(docType, value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return sapDocumentKeyString(docType, parsed), true
	}
	return docType + ":" + value, true
}

func sapDocumentKeyString(docType string, docEntry int64) string {
	return docType + ":" + strconv.FormatInt(docEntry, 10)
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func optionalStringArg(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}
