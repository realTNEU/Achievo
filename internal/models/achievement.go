package models

import "time"

// Achievement represents an achievement definition
type Achievement struct {
	ID          string    `bson:"_id" json:"id"`
	GameID      string    `bson:"game_id" json:"game_id"`
	SteamAppID  string    `bson:"steam_app_id,omitempty" json:"steam_app_id,omitempty"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	IconURL     string    `bson:"icon_url,omitempty" json:"icon_url,omitempty"`
	IconLockedURL string  `bson:"icon_locked_url,omitempty" json:"icon_locked_url,omitempty"`
	IsHidden    bool      `bson:"is_hidden" json:"is_hidden"`
	UnlockPercent float64 `bson:"unlock_percent,omitempty" json:"unlock_percent,omitempty"` // from Steam API
	FetchedAt   time.Time `bson:"fetched_at" json:"fetched_at"`
}

// AchievementProgress tracks user's progress on achievements
type AchievementProgress struct {
	ID            string    `bson:"_id" json:"id"`
	AchievementID string    `bson:"achievement_id" json:"achievement_id"`
	GameID        string    `bson:"game_id" json:"game_id"`
	Unlocked      bool      `bson:"unlocked" json:"unlocked"`
	UnlockedAt    time.Time `bson:"unlocked_at,omitempty" json:"unlocked_at,omitempty"`
	Progress      float64   `bson:"progress" json:"progress"` // 0.0 to 1.0 for incremental achievements
	LastUpdated   time.Time `bson:"last_updated" json:"last_updated"`
}

