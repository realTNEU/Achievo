package models

import "time"

// Session represents a game playing session
type Session struct {
	ID        string    `bson:"_id" json:"id"`
	GameID    string    `bson:"game_id" json:"game_id"`
	ProcessID int       `bson:"process_id" json:"process_id"`
	StartTime time.Time `bson:"start_time" json:"start_time"`
	EndTime   time.Time `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Duration  int64     `bson:"duration" json:"duration"` // in seconds
	IsActive  bool      `bson:"is_active" json:"is_active"`
}
