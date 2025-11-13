package tracker

import (
	"fmt"
	"time"

	"achievo/internal/models"
	"achievo/internal/storage"
)

// Tracker tracks achievements and emits events
type Tracker struct {
	storage *storage.Storage
}

// New creates a new Tracker
func New(storage *storage.Storage) *Tracker {
	return &Tracker{
		storage: storage,
	}
}

// TrackEvent processes an event and checks for achievement unlocks
func (t *Tracker) TrackEvent(event models.Event) error {
	// Save the event
	if err := t.storage.SaveEvent(&event); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	// If it's an achievement unlock event, update progress
	if event.Type == models.EventTypeAchievementUnlock {
		achievementID, ok := event.Payload["achievement_id"].(string)
		if !ok {
			return fmt.Errorf("achievement_unlock event missing achievement_id")
		}

		progress := &models.AchievementProgress{
			ID:            fmt.Sprintf("%s-%s", event.GameID, achievementID),
			AchievementID: achievementID,
			GameID:        event.GameID,
			Unlocked:      true,
			UnlockedAt:    event.Timestamp,
			Progress:      1.0,
			LastUpdated:   event.Timestamp,
		}

		if err := t.storage.SaveAchievementProgress(progress); err != nil {
			return fmt.Errorf("failed to save achievement progress: %w", err)
		}
	}

	return nil
}

// CheckAchievement checks the progress of a specific achievement
func (t *Tracker) CheckAchievement(gameID, achievementID string) (*models.AchievementProgress, error) {
	progress, err := t.storage.GetAchievementProgress(gameID, achievementID)
	if err != nil {
		return nil, err
	}

	if progress == nil {
		// Return default progress
		return &models.AchievementProgress{
			ID:            fmt.Sprintf("%s-%s", gameID, achievementID),
			AchievementID: achievementID,
			GameID:        gameID,
			Unlocked:      false,
			Progress:      0.0,
			LastUpdated:   time.Now(),
		}, nil
	}

	return progress, nil
}

// UnlockAchievement unlocks an achievement
func (t *Tracker) UnlockAchievement(gameID, achievementID string) error {
	// Check if already unlocked
	progress, err := t.storage.GetAchievementProgress(gameID, achievementID)
	if err != nil {
		return err
	}

	if progress != nil && progress.Unlocked {
		// Already unlocked
		return nil
	}

	// Create unlock event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-%s-%d", gameID, achievementID, time.Now().Unix()),
		Type:      models.EventTypeAchievementUnlock,
		GameID:    gameID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"achievement_id": achievementID,
		},
		Synced: false,
	}

	// Track the event (this will also update progress)
	return t.TrackEvent(event)
}

// UpdateProgress updates achievement progress (for incremental achievements)
func (t *Tracker) UpdateProgress(gameID, achievementID string, progress float64) error {
	if progress < 0.0 || progress > 1.0 {
		return fmt.Errorf("progress must be between 0.0 and 1.0")
	}

	current, err := t.storage.GetAchievementProgress(gameID, achievementID)
	if err != nil {
		return err
	}

	now := time.Now()
	if current == nil {
		current = &models.AchievementProgress{
			ID:            fmt.Sprintf("%s-%s", gameID, achievementID),
			AchievementID: achievementID,
			GameID:        gameID,
			Unlocked:      progress >= 1.0,
			Progress:      progress,
			LastUpdated:   now,
		}
		if progress >= 1.0 {
			current.UnlockedAt = now
		}
	} else {
		current.Progress = progress
		current.Unlocked = progress >= 1.0
		current.LastUpdated = now
		if progress >= 1.0 && !current.Unlocked {
			current.UnlockedAt = now
			// Emit unlock event
			event := models.Event{
				ID:        fmt.Sprintf("event-%s-%s-%d", gameID, achievementID, now.Unix()),
				Type:      models.EventTypeAchievementUnlock,
				GameID:    gameID,
				Timestamp: now,
				Payload: map[string]interface{}{
					"achievement_id": achievementID,
					"progress":       progress,
				},
				Synced: false,
			}
			if err := t.storage.SaveEvent(&event); err != nil {
				return fmt.Errorf("failed to save unlock event: %w", err)
			}
		}
	}

	return t.storage.SaveAchievementProgress(current)
}

// GetProgress returns all achievement progress for a game
func (t *Tracker) GetProgress(gameID string) ([]models.AchievementProgress, error) {
	return t.storage.GetAllProgress(gameID)
}

// TrackStatUpdate tracks a statistic update
func (t *Tracker) TrackStatUpdate(gameID, statName string, value interface{}) error {
	var stat models.Statistic

	switch v := value.(type) {
	case float64:
		stat = models.Statistic{
			ID:          fmt.Sprintf("%s-%s", gameID, statName),
			GameID:      gameID,
			Name:        statName,
			Value:       v,
			LastUpdated: time.Now(),
		}
	case int:
		stat = models.Statistic{
			ID:          fmt.Sprintf("%s-%s", gameID, statName),
			GameID:      gameID,
			Name:        statName,
			ValueInt:    int64(v),
			LastUpdated: time.Now(),
		}
	case int64:
		stat = models.Statistic{
			ID:          fmt.Sprintf("%s-%s", gameID, statName),
			GameID:      gameID,
			Name:        statName,
			ValueInt:    v,
			LastUpdated: time.Now(),
		}
	case string:
		stat = models.Statistic{
			ID:          fmt.Sprintf("%s-%s", gameID, statName),
			GameID:      gameID,
			Name:        statName,
			ValueString: v,
			LastUpdated: time.Now(),
		}
	default:
		return fmt.Errorf("unsupported stat value type: %T", value)
	}

	if err := t.storage.SaveStatistic(&stat); err != nil {
		return fmt.Errorf("failed to save statistic: %w", err)
	}

	// Create stat update event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-%s-%d", gameID, statName, time.Now().Unix()),
		Type:      models.EventTypeStatUpdate,
		GameID:    gameID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"stat_name": statName,
			"value":     value,
		},
		Synced: false,
	}

	return t.storage.SaveEvent(&event)
}

