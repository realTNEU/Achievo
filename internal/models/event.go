package models

import "time"

// EventType represents the type of event
type EventType string

const (
	EventTypeAchievementUnlock EventType = "achievement_unlock"
	EventTypeStatUpdate        EventType = "stat_update"
	EventTypeGameStart         EventType = "game_start"
	EventTypeGameStop          EventType = "game_stop"
	EventTypeSessionEnd        EventType = "session_end"
)

// Event represents an event in the system
type Event struct {
	ID        string                 `bson:"_id" json:"id"`
	Type      EventType              `bson:"type" json:"type"`
	GameID    string                 `bson:"game_id" json:"game_id"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	Payload   map[string]interface{} `bson:"payload" json:"payload"`
	Synced    bool                   `bson:"synced" json:"synced"`
	SyncedAt  time.Time              `bson:"synced_at,omitempty" json:"synced_at,omitempty"`
}

