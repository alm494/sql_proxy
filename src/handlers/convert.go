package handlers

import (
	"database/sql"
	"sql-proxy/src/db"
)

// Converts SQL query result to json
func convertRows(rows *sql.Rows, columns *[]string) (*[]map[string]any, uint32, bool) {
	var rowsCount uint32 = 0
	colsCount := len(*columns)
	tableData := make([]map[string]any, 0)
	values := make([]any, colsCount)
	valuePtrs := make([]any, colsCount)
	exceedsMaxRows := false

	for rows.Next() {
		for i := range *columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]any)
		for i, col := range *columns {
			var v any
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		if rowsCount > db.MaxRows {
			exceedsMaxRows = true
			break
		}
		tableData = append(tableData, entry)
		rowsCount++
	}
	return &tableData, rowsCount, exceedsMaxRows
}
