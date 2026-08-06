package db

import (
	"os"
	"testing"

	"github.com/mcpjungle/mcpjungle/pkg/testhelpers"
)

// mysqlTestDSNEnvVar names the environment variable that provides a live MySQL
// DSN for integration-style tests in this package. When unset, tests that need
// a real database are skipped — MySQL has no in-memory mode, unlike SQLite.
const mysqlTestDSNEnvVar = "TEST_MYSQL_DSN"

func TestNewDBConnection_EmptyDSN_Error(t *testing.T) {
	// MySQL is the only supported database, so an empty DSN must error instead
	// of silently falling back to a local file.
	dbConn, err := NewDBConnection("")
	testhelpers.AssertError(t, err)
	if dbConn != nil {
		t.Errorf("Expected db to be nil, got %v", dbConn)
	}
}

func TestNewDBConnection_MySQL(t *testing.T) {
	dsn := os.Getenv(mysqlTestDSNEnvVar)
	if dsn == "" {
		t.Skipf("%s not set, skipping live MySQL connection test", mysqlTestDSNEnvVar)
	}

	dbConn, err := NewDBConnection(dsn)
	if err != nil {
		t.Fatalf("NewDBConnection failed: %v", err)
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		t.Fatalf("failed to get *sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping mysql: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close connection: %v", err)
	}
}

func TestMysqlDSNFromURL(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "mysql url with port",
			dsn:  "mysql://user:pass@localhost:3306/mcpjungle",
			want: "user:pass@tcp(localhost:3306)/mcpjungle?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "mysql url without port defaults to 3306",
			dsn:  "mysql://user:pass@localhost/mcpjungle",
			want: "user:pass@tcp(localhost:3306)/mcpjungle?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "native dsn gets default params appended",
			dsn:  "user:pass@tcp(localhost:3306)/db",
			want: "user:pass@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mysqlDSNFromURL(tt.dsn)
			testhelpers.AssertEqual(t, tt.want, got)
		})
	}
}
