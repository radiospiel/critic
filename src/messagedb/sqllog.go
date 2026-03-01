package messagedb

import (
	"database/sql"
	"regexp"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

const sqlLogLevel = logger.INFO // change to logger.DEBUG to quiet SQL logging

var sqlLog = logger.WithTopic("sql").WithLevel(sqlLogLevel)

var compactRe = regexp.MustCompile(`\s+`)

func compact(query string) string {
	return compactRe.ReplaceAllString(query, " ")
}

func ms(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000.0
}

func (db *DB) Select(dest any, query string, args ...any) error {
	start := time.Now()
	err := db.db.Select(dest, query, args...)
	sqlLog.Info("%.1fms SELECT %s args=%v", ms(start), compact(query), args)
	return err
}

func (db *DB) Get(dest any, query string, args ...any) error {
	start := time.Now()
	err := db.db.Get(dest, query, args...)
	sqlLog.Info("%.1fms GET %s args=%v", ms(start), compact(query), args)
	return err
}

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := db.db.Exec(query, args...)
	sqlLog.Info("%.1fms EXEC %s args=%v", ms(start), compact(query), args)
	return result, err
}

func (db *DB) NamedExec(query string, arg any) (sql.Result, error) {
	start := time.Now()
	result, err := db.db.NamedExec(query, arg)
	sqlLog.Info("%.1fms NAMEDEXEC %s", ms(start), compact(query))
	return result, err
}
