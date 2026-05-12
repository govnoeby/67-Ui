package database

import (
	"github.com/govnoeby/67-Ui/v3/config"
)

// JSONEachQuery returns the SQL fragment for iterating over JSON array elements.
// SQLite uses JSON_EACH, PostgreSQL uses jsonb_array_elements, MySQL uses JSON_TABLE.
func JSONEachQuery() string {
	switch config.GetDBType() {
	case "postgres":
		return "jsonb_array_elements"
	case "mysql":
		return "JSON_TABLE"
	default:
		return "JSON_EACH"
	}
}

// JSONExtract returns the SQL fragment for extracting a value from JSON.
// SQLite: JSON_EXTRACT(json, '$.path')
// PostgreSQL: jsonb_extract_path_text(json, 'path') or json->'path'
// MySQL: JSON_EXTRACT(json, '$.path')
func JSONExtract(field, path string) string {
	switch config.GetDBType() {
	case "postgres":
		return field + "->>'" + path + "'"
	default:
		return "JSON_EXTRACT(" + field + ", '$." + path + "')"
	}
}

// JSONExtractText returns a SQL expression that extracts a text value from JSON.
func JSONExtractText(field, path string) string {
	switch config.GetDBType() {
	case "postgres":
		return field + "->>'" + path + "'"
	default:
		return "JSON_EXTRACT(" + field + ", '$." + path + "')"
	}
}

// IsJSONSupported returns true if the current database supports JSON operations natively.
func IsJSONSupported() bool {
	return config.GetDBType() != "mysql" // MySQL 5.7+ supports JSON
}
