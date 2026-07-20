package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONImplementsDatabaseRoundTrip(t *testing.T) {
	original := JSON(`{"name":"mcp","enabled":true}`)

	value, err := original.Value()
	require.NoError(t, err)
	require.Equal(t, `{"name":"mcp","enabled":true}`, value)

	var scanned JSON
	require.NoError(t, scanned.Scan([]byte(`{"items":[1,2]}`)))
	require.JSONEq(t, `{"items":[1,2]}`, string(scanned))
}

func TestJSONRejectsInvalidDocument(t *testing.T) {
	_, err := JSON(`not-json`).Value()
	require.ErrorContains(t, err, "invalid JSON")
}

func TestJSONMarshalsAsDocumentInsteadOfBase64(t *testing.T) {
	encoded, err := json.Marshal(JSON(`{"name":"mcp"}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"mcp"}`, string(encoded))
}

func TestJSONUsesMySQLJSONColumn(t *testing.T) {
	require.Equal(t, "json", JSON(nil).GormDataType())
}
