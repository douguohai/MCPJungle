// Package db provides database functionality for the MCPJungle application.
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

// TODO: Turn this into a singleton class.
// Only one database connection should be created and used throughout the application.

// NewDBConnection creates a new MySQL database connection from the provided DSN.
// The DSN may be a "mysql://..." URL or a go-sql-driver native DSN
// ("user:pass@tcp(host:port)/db"). An empty DSN is an error — MySQL is the only
// supported database, so there is no longer a local-file fallback.
func NewDBConnection(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("no database DSN configured: set DATABASE_URL or the MYSQL_* environment variables (MySQL is the only supported database)")
	}
	mysqlDSN := mysqlDSNFromURL(dsn)
	log.Printf("[db] Using mysql database (%s)", maskDSN(dsn))

	c := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(mysql.Open(mysqlDSN), c)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// mysqlDSNFromURL normalizes a MySQL DSN into the go-sql-driver native format
// (user:pass@tcp(host:port)/db?params). It accepts "mysql://..." URLs and DSNs
// that are already native. charset=utf8mb4 and parseTime=True are enforced so
// that JSON columns and time values behave correctly under MySQL.
func mysqlDSNFromURL(raw string) string {
	// Already-native go-sql-driver format.
	if strings.Contains(raw, "@tcp(") {
		return withMySQLDefaults(raw)
	}

	// Normalize to a form url.Parse can handle. A bare "user:pass@host"
	// (no "://") would be mis-parsed because url.Parse treats "user:" as
	// the scheme, so ensure there is a scheme:// prefix.
	parseURL := raw
	if !strings.Contains(parseURL, "://") {
		parseURL = "mysql://" + strings.TrimPrefix(parseURL, "mysql:")
	}

	u, err := url.Parse(parseURL)
	if err != nil {
		return withMySQLDefaults(raw)
	}

	user := u.User.Username()
	pass, _ := u.User.Password()
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "3306"
	}
	dbName := strings.TrimPrefix(u.Path, "/")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, dbName)
	return withMySQLDefaults(dsn)
}

// withMySQLDefaults ensures charset/parseTime/loc params are present without
// duplicating ones the caller already supplied.
func withMySQLDefaults(dsn string) string {
	for _, p := range []string{"charset=utf8mb4", "parseTime=True", "loc=Local"} {
		key := p[:strings.IndexByte(p, '=')]
		if !strings.Contains(dsn, key+"=") {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			dsn += sep + p
		}
	}
	return dsn
}

// maskDSN returns a copy of the DSN with the password redacted so it is safe to
// log. If the DSN has no credentials, it is returned unchanged.
func maskDSN(dsn string) string {
	at := strings.LastIndex(dsn, "@")
	if at <= 0 {
		return dsn
	}
	head := dsn[:at]
	if i := strings.Index(head, ":"); i >= 0 {
		head = head[:i]
	}
	return head + ":***@" + dsn[at+1:]
}
