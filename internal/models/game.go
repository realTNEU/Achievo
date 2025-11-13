package models

import "time"

// GameType represents the type of game
type GameType string

const (
	GameTypeSteam    GameType = "steam"
	GameTypeNonSteam GameType = "non_steam"
)

// Game represents a detected game
type Game struct {
	ID               string    `bson:"_id" json:"id"`
	Name             string    `bson:"name" json:"name"`
	Type             GameType  `bson:"type" json:"type"`
	SteamAppID       string    `bson:"steam_app_id,omitempty" json:"steam_app_id,omitempty"`
	ExecutablePath   string    `bson:"executable_path" json:"executable_path"`
	ProcessName      string    `bson:"process_name" json:"process_name"`
	InstallDirectory string    `bson:"install_directory" json:"install_directory"`
	WindowTitle      string    `bson:"window_title,omitempty" json:"window_title,omitempty"`
	FirstDetected    time.Time `bson:"first_detected" json:"first_detected"`
	LastPlayed       time.Time `bson:"last_played" json:"last_played"`
	TotalPlaytime    int64     `bson:"total_playtime" json:"total_playtime"` // in seconds
}
