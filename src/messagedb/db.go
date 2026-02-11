package messagedb

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/radiospiel/critic/simple-go/logger"
)

var reWhitespace = regexp.MustCompile(`\s+`)

const enableRuntimeLog = false

func logRuntime[T any](query string, fun func() T) T {
	if enableRuntimeLog {
		sqlLabel := strings.TrimSpace(reWhitespace.ReplaceAllString(query, " "))
		return logger.Runtime(sqlLabel, fun)
	} else {
		return fun()
	}
}

// exec wraps sql.DB.Exec with timing via logger.Runtime.
func (d *DB) exec(query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := logRuntime(query, func() error {
		var e error
		result, e = d.db.Exec(query, args...)
		return e
	})
	return result, err
}

// ask runs a query expecting a single row and scans the result columns into dest.
func (d *DB) ask(query string, args []any, dest ...any) error {
	return logRuntime(query, func() error {
		return d.db.QueryRow(query, args...).Scan(dest...)
	})
}

// all runs a query and collects all rows using the provided scanner function.
func all[T any](db *DB, query string, scanner func(*sql.Rows) (T, error), args ...any) ([]T, error) {
	var rows *sql.Rows
	err := logRuntime(query, func() error {
		var e error
		rows, e = db.db.Query(query, args...)
		return e
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		item, err := scanner(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}
