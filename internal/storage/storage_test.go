package storage

import (
	"os"
	"testing"
	"time"

	"achievo/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStorage tests basic storage operations
// Note: This requires a running MongoDB instance
// Set MONGODB_URI environment variable or skip if not available
func TestStorage(t *testing.T) {
	uri := "mongodb://localhost:27017"
	if testURI := os.Getenv("TEST_MONGODB_URI"); testURI != "" {
		uri = testURI
	}

	storage, err := New(uri, "achievo_test", 10)
	if err != nil {
		t.Skipf("Skipping test: MongoDB not available: %v", err)
	}
	defer storage.Close()

	// Test SaveGame
	game := &models.Game{
		ID:               "test-game-1",
		Name:             "Test Game",
		Type:             models.GameTypeNonSteam,
		ExecutablePath:   "/path/to/game.exe",
		ProcessName:      "game.exe",
		InstallDirectory: "/path/to",
		FirstDetected:    time.Now(),
		LastPlayed:       time.Now(),
	}

	if err := storage.SaveGame(game); err != nil {
		t.Fatalf("SaveGame failed: %v", err)
	}

	// Test GetGame
	retrieved, err := storage.GetGame("test-game-1")
	if err != nil {
		t.Fatalf("GetGame failed: %v", err)
	}
	if retrieved == nil || retrieved.Name != "Test Game" {
		t.Fatalf("GetGame returned incorrect game")
	}

	// Test SaveSession
	session := &models.Session{
		ID:        "test-session-1",
		GameID:    "test-game-1",
		ProcessID: 12345,
		StartTime: time.Now(),
		IsActive:  true,
	}

	if err := storage.SaveSession(session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Test GetActiveSession
	active, err := storage.GetActiveSession("test-game-1")
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != "test-session-1" {
		t.Fatalf("GetActiveSession returned incorrect session")
	}

	// Test SaveEvent
	event := &models.Event{
		ID:        primitive.NewObjectID().Hex(),
		Type:      models.EventTypeGameStart,
		GameID:    "test-game-1",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{"test": "data"},
		Synced:    false,
	}

	if err := storage.SaveEvent(event); err != nil {
		t.Fatalf("SaveEvent failed: %v", err)
	}

	// Test GetUnsyncedEvents
	events, err := storage.GetUnsyncedEvents(10)
	if err != nil {
		t.Fatalf("GetUnsyncedEvents failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("GetUnsyncedEvents returned no events")
	}

	// Test MarkEventSynced
	if err := storage.MarkEventSynced(event.ID); err != nil {
		t.Fatalf("MarkEventSynced failed: %v", err)
	}

	// Test UpsertAchievementDefs
	achievements := []models.Achievement{
		{
			ID:          "test-ach-1",
			GameID:      "test-game-1",
			SteamAppID:  "123",
			Name:        "Test Achievement",
			Description: "Test Description",
			FetchedAt:   time.Now(),
		},
	}

	if err := storage.UpsertAchievementDefs(achievements); err != nil {
		t.Fatalf("UpsertAchievementDefs failed: %v", err)
	}
}

