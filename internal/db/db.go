// Package db provides database functionality for the MCPJungle application.
package db

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TODO: Turn this into a singleton class.
// Only one database connection should be created and used throughout the application.

const (
	dbFilename           = "mcpjungle.db"
	deprecatedDBFilename = "mcp.db"
)

// getSQLiteDBPath determines which SQLite database file to use.
// It prioritizes the new mcpjungle.db file, but falls back to the old mcp.db file for backward compatibility.
func getSQLiteDBPath() string {
	// Check if the new database file exists
	if _, err := os.Stat(dbFilename); err == nil {
		return dbFilename
	}

	// Check if the old database file exists (backward compatibility)
	if _, err := os.Stat(deprecatedDBFilename); err == nil {
		log.Printf("[db] WARNING: Using deprecated database file '%s'. Please consider renaming it to '%s' for future compatibility.", deprecatedDBFilename, dbFilename)
		return deprecatedDBFilename
	}

	// Neither exists, use the new file name
	return dbFilename
}

// resolveSQLiteDBPath determines the SQLite database path to use based on the provided configuration.
// If a configured path is provided, it uses that. Otherwise, it falls back to the default filename in the current directory.
func resolveSQLiteDBPath(configuredPath string) string {
	if configuredPath != "" {
		return configuredPath
	}
	return getSQLiteDBPath()
}

// NewDBConnection creates a new database connection based on the provided DSN.
// If the DSN is empty, it falls back to an embedded SQLite database.
// Otherwise the database kind is auto-detected from the DSN by detectDialector:
//   - MySQL  when the DSN uses the "mysql://"/"mysql:" scheme or the
//     go-sql-driver native "user:pass@tcp(host:port)/db" format;
//   - PostgreSQL for everything else (preserves the original behaviour).
//
// For backward compatibility, SQLite uses an existing "mcp.db" file if present,
// otherwise it creates/uses "mcpjungle.db".
func NewDBConnection(dsn string, sqliteDBPath string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if dsn == "" {
		dbPath := resolveSQLiteDBPath(sqliteDBPath)
		log.Printf("[db] Using sqlite database at %s", dbPath)
		dialector = sqlite.Open(fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL", dbPath))
	} else {
		dialector = detectDialector(dsn)
		log.Printf("[db] Using %s database (%s)", dialector.Name(), maskDSN(dsn))
	}

	c := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(dialector, c)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// detectDialector selects the GORM dialector based on the DSN's scheme/format.
// MySQL is selected when the DSN uses a "mysql:" scheme (e.g. "mysql://..." or
// "mysql:user:pass@...") or already matches the go-sql-driver native format
// containing "@tcp(...)". Any other DSN is treated as PostgreSQL.
func detectDialector(dsn string) gorm.Dialector {
	switch {
	case strings.HasPrefix(dsn, "mysql:"), strings.Contains(dsn, "@tcp("):
		return mysql.Open(mysqlDSNFromURL(dsn))
	default:
		return postgres.Open(dsn)
	}
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
