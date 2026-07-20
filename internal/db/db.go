// Package db provides the MySQL persistence connection used by MCPJungle.
package db

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDBConnection opens a MySQL connection. MCPJungle intentionally has no
// embedded database fallback: a missing or non-MySQL DSN is a startup error.
func NewDBConnection(rawDSN string) (*gorm.DB, error) {
	dsn, err := NormalizeMySQLDSN(rawDSN)
	if err != nil {
		return nil, err
	}

	log.Printf("[db] Using mysql database (%s)", maskDSN(dsn))
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	return database, nil
}

// NormalizeMySQLDSN converts a mysql:// URL or a go-sql-driver native DSN to
// the native form and enforces the connection options MCPJungle relies on.
func NormalizeMySQLDSN(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("MySQL DSN is required")
	}

	var dsn string
	switch {
	case strings.Contains(raw, "://"):
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid MySQL DSN: %w", err)
		}
		if !strings.EqualFold(u.Scheme, "mysql") {
			return "", fmt.Errorf("only MySQL is supported")
		}
		if u.User == nil || u.User.Username() == "" {
			return "", fmt.Errorf("MySQL username is required")
		}
		databaseName := strings.TrimPrefix(u.EscapedPath(), "/")
		if databaseName == "" {
			return "", fmt.Errorf("MySQL database name is required")
		}
		host := u.Hostname()
		if host == "" {
			return "", fmt.Errorf("MySQL host is required")
		}
		port := u.Port()
		if port == "" {
			port = "3306"
		}
		password, _ := u.User.Password()
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s",
			u.User.Username(),
			password,
			host,
			port,
			databaseName,
		)
		if u.RawQuery != "" {
			dsn += "?" + u.RawQuery
		}
	case strings.Contains(raw, "@tcp("):
		dsn = raw
	default:
		return "", fmt.Errorf("only MySQL is supported; use mysql:// URL or go-sql-driver DSN")
	}

	path := dsn
	if question := strings.IndexByte(path, '?'); question >= 0 {
		path = path[:question]
	}
	slash := strings.LastIndexByte(path, '/')
	if slash < 0 || slash == len(path)-1 {
		return "", fmt.Errorf("MySQL database name is required")
	}

	return withMySQLDefaults(dsn), nil
}

func withMySQLDefaults(dsn string) string {
	for _, parameter := range []string{"charset=utf8mb4", "parseTime=True", "loc=UTC"} {
		key := parameter[:strings.IndexByte(parameter, '=')]
		query := ""
		if question := strings.IndexByte(dsn, '?'); question >= 0 {
			query = dsn[question+1:]
		}
		found := false
		for _, pair := range strings.Split(query, "&") {
			if strings.EqualFold(strings.SplitN(pair, "=", 2)[0], key) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn += separator + parameter
	}
	return dsn
}

// maskDSN returns a DSN that is safe to include in startup logs.
func maskDSN(dsn string) string {
	at := strings.LastIndex(dsn, "@")
	if at <= 0 {
		return dsn
	}
	head := dsn[:at]
	if colon := strings.Index(head, ":"); colon >= 0 {
		head = head[:colon]
	}
	return head + ":***@" + dsn[at+1:]
}
