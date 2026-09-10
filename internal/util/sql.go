package util

import (
	"database/sql"
	"encoding/json"
)

func GetSQLNullString(str *string) sql.NullString {
	if str == nil || len(*str) == 0 {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *str, Valid: true}
}

func GetSQLNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{Valid: false}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

func GetStringFromSQLNullString(str sql.NullString) string {
	if str.Valid {
		return str.String
	}
	return ""
}

func GetOptionalString(str sql.NullString) *string {
	if str.Valid {
		return &str.String
	}
	return nil
}

func ToRawMessage[T any](cfg *T) json.RawMessage {
	if cfg == nil {
		return json.RawMessage{}
	}
	raw, err := json.Marshal(*cfg)
	if err != nil {
		return json.RawMessage{}
	}
	if string(raw) == "null" {
		return json.RawMessage{}
	}
	return raw
}
