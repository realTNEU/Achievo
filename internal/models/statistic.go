package models

import "time"

// Statistic represents a gameplay statistic
type Statistic struct {
	ID          string    `bson:"_id" json:"id"`
	GameID      string    `bson:"game_id" json:"game_id"`
	Name        string    `bson:"name" json:"name"`
	Value       float64   `bson:"value" json:"value"`
	ValueInt    int64     `bson:"value_int,omitempty" json:"value_int,omitempty"`
	ValueString string    `bson:"value_string,omitempty" json:"value_string,omitempty"`
	LastUpdated time.Time `bson:"last_updated" json:"last_updated"`
}
