package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func clearMySQLEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		DBUrlEnvVar,
		MysqlHostEnvVar,
		MysqlPortEnvVar,
		MysqlUserEnvVar,
		MysqlUserEnvVar + "_FILE",
		MysqlPasswordEnvVar,
		MysqlPasswordEnvVar + "_FILE",
		MysqlDBEnvVar,
		MysqlDBEnvVar + "_FILE",
	} {
		t.Setenv(key, "")
	}
}

func TestGetMysqlDSNUsesDatabaseURLFirst(t *testing.T) {
	clearMySQLEnv(t)
	t.Setenv(DBUrlEnvVar, "mysql://url-user:url-pass@url-db:3307/url_name")
	t.Setenv(MysqlHostEnvVar, "ignored-host")

	got, err := getMysqlDSN()
	require.NoError(t, err)
	require.Equal(
		t,
		"url-user:url-pass@tcp(url-db:3307)/url_name?charset=utf8mb4&parseTime=True&loc=UTC",
		got,
	)
}

func TestGetMysqlDSNBuildsFromEnvironment(t *testing.T) {
	clearMySQLEnv(t)
	t.Setenv(MysqlHostEnvVar, "mysql.internal")
	t.Setenv(MysqlPortEnvVar, "3307")
	t.Setenv(MysqlUserEnvVar, "hub")
	t.Setenv(MysqlPasswordEnvVar, "secret")
	t.Setenv(MysqlDBEnvVar, "mcp_hub")

	got, err := getMysqlDSN()
	require.NoError(t, err)
	require.Equal(
		t,
		"hub:secret@tcp(mysql.internal:3307)/mcp_hub?charset=utf8mb4&parseTime=True&loc=UTC",
		got,
	)
}

func TestGetMysqlDSNRequiresConfiguration(t *testing.T) {
	clearMySQLEnv(t)

	got, err := getMysqlDSN()
	require.Empty(t, got)
	require.ErrorContains(t, err, "DATABASE_URL or MYSQL_HOST is required")
}

func TestStartCommandDoesNotExposeSQLiteFlag(t *testing.T) {
	require.Nil(t, startServerCmd.Flags().Lookup("sqlite-db-path"))
}
