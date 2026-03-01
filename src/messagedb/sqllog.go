package messagedb

import (
	"database/sql"
	"regexp"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

const sqlSlowThreshold = 10 * time.Millisecond

var sqlLog = logger.WithTopic("sql")

var compactRe = regexp.MustCompile(`\s+`)

func compact(query string) string {
	return compactRe.ReplaceAllString(query, " ")
}

func ms(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}

func logSlow(start time.Time, op, query string, args ...any) {
	if time.Since(start) >= sqlSlowThreshold {
		sqlLog.Info("%.1fms %s %s args=%v", ms(start), op, compact(query), args)
	}
}

func (db *DB) Select(dest any, query string, args ...any) error {
	start := time.Now()
	err := db.db.Select(dest, query, args...)
	logSlow(start, "SELECT", query, args)
	return err
}

func (db *DB) Get(dest any, query string, args ...any) error {
	start := time.Now()
	err := db.db.Get(dest, query, args...)
	logSlow(start, "GET", query, args)
	return err
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := db.db.Exec(query, args...)
	logSlow(start, "EXEC", query, args)
	return result, err
}

func (db *DB) NamedExec(query string, arg any) (sql.Result, error) {
	start := time.Now()
	result, err := db.db.NamedExec(query, arg)
	if time.Since(start) >= sqlSlowThreshold {
		sqlLog.Info("%.1fms NAMEDEXEC %s", ms(start), compact(query))
	}
	return result, err
}
