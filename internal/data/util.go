package data

import "strings"

func splitSQL(sql string) []string {
	var queries []string
	// Basic implementation: split by semicolon + newline or just semicolon at end of line
	// This schema is formatted well enough that we can split by `;\n` or `;` but trimming spaces.
	parts := strings.Split(sql, ";")
	for _, p := range parts {
		q := strings.TrimSpace(p)
		if q != "" {
			queries = append(queries, q)
		}
	}
	return queries
}
