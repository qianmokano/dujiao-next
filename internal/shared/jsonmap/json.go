// Package jsonmap provides the shared JSON object value used at persistence
// and transport boundaries. It is deliberately independent from GORM models
// and business modules so bounded contexts do not need to import a global
// models package merely to exchange arbitrary JSON objects.
package jsonmap

import (
	"database/sql/driver"
	"encoding/json"
)

// JSON is a JSON object with database/sql serialization support.
type JSON map[string]interface{}

// Value implements driver.Valuer.
func (value JSON) Value() (driver.Value, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

// Scan implements sql.Scanner.
func (value *JSON) Scan(source interface{}) error {
	if source == nil {
		*value = make(JSON)
		return nil
	}
	var bytes []byte
	switch typed := source.(type) {
	case []byte:
		bytes = typed
	case string:
		// 纯 Go SQLite 驱动对 TEXT 列返回 string;外部工具(psql/sqlite3 CLI/维护脚本)
		// 写入的 JSON 行会走这里,静默丢弃会把整段配置读成空对象。
		bytes = []byte(typed)
	default:
		return nil
	}
	return json.Unmarshal(bytes, value)
}
