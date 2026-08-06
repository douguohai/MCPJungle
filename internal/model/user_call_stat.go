package model

import "gorm.io/gorm"

// UserCallStat records how many MCP tool calls a user has made against each
// upstream server. Only the count is stored — never tool names, arguments, or
// results — so call content is never persisted.
type UserCallStat struct {
	gorm.Model
	UserID     uint   `json:"user_id" gorm:"uniqueIndex:idx_user_server_date,priority:1;not null"`
	ServerName string `json:"server_name" gorm:"type:varchar(255);uniqueIndex:idx_user_server_date,priority:2;not null"`
	// Date is the calendar day (YYYY-MM-DD, UTC) of the calls, enabling trend
	// reports over time without recording any call content.
	Date  string `json:"date" gorm:"type:varchar(10);uniqueIndex:idx_user_server_date,priority:3;not null"`
	Count uint64 `json:"count"`
}
