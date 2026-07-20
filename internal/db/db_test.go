package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeMySQLDSN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL",
			input: "mysql://user:pass@db:3306/hub",
			want:  "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC",
		},
		{
			name:  "native DSN",
			input: "user:pass@tcp(db:3306)/hub",
			want:  "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC",
		},
		{
			name:  "preserves existing parameters",
			input: "user:pass@tcp(db:3306)/hub?timeout=5s",
			want:  "user:pass@tcp(db:3306)/hub?timeout=5s&charset=utf8mb4&parseTime=True&loc=UTC",
		},
		{
			name:  "does not duplicate defaults",
			input: "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC",
			want:  "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeMySQLDSN(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeMySQLDSNRejectsUnsupportedValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		error string
	}{
		{name: "empty", input: "", error: "MySQL DSN is required"},
		{name: "PostgreSQL", input: "postgres://user:pass@db:5432/hub", error: "only MySQL is supported"},
		{name: "unsupported URL", input: "mongodb://user:pass@db:27017/hub", error: "only MySQL is supported"},
		{name: "URL without database", input: "mysql://user:pass@db:3306", error: "database name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeMySQLDSN(tt.input)
			require.Empty(t, got)
			require.ErrorContains(t, err, tt.error)
		})
	}
}

func TestNewDBConnectionRejectsEmptyDSN(t *testing.T) {
	database, err := NewDBConnection("")
	require.Nil(t, database)
	require.ErrorContains(t, err, "MySQL DSN is required")
}

func TestNewDBConnectionRejectsPostgresDSN(t *testing.T) {
	database, err := NewDBConnection("postgres://user:pass@db:5432/hub")
	require.Nil(t, database)
	require.ErrorContains(t, err, "only MySQL is supported")
}

func TestMaskDSNRedactsPassword(t *testing.T) {
	masked := maskDSN("user:secret@tcp(db:3306)/hub")
	require.Equal(t, "user:***@tcp(db:3306)/hub", masked)
	require.NotContains(t, masked, "secret")
}
