package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// JSON stores an arbitrary JSON document in a native MySQL JSON column.
type JSON []byte

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	return string(j), nil
}

func (j *JSON) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("cannot scan JSON from %T", value)
	}
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON document")
	}
	*j = append((*j)[:0], data...)
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	return j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON document")
	}
	*j = append((*j)[:0], data...)
	return nil
}

func (JSON) GormDataType() string {
	return "json"
}

func (JSON) GormDBDataType(*gorm.DB, *schema.Field) string {
	return "JSON"
}
