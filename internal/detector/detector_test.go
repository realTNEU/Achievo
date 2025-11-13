package detector

import (
	"testing"
	"time"

	"achievo/internal/models"
)

// TestDetector tests game detection functionality
func TestDetector(t *testing.T) {
	detector := New()

	// Register a test game
	game := &models.Game{
		ID:          "test-game-1",
		Name:        "Test Game",
		Type:        models.GameTypeNonSteam,
		ProcessName: "testgame.exe",
		FirstDetected: time.Now(),
		LastPlayed:  time.Now(),
	}

	detector.RegisterGame(game)

	// Test GetKnownGames
	known := detector.GetKnownGames()
	if len(known) != 1 {
		t.Fatalf("Expected 1 known game, got %d", len(known))
	}

	if known[0].ID != "test-game-1" {
		t.Fatalf("Expected game ID 'test-game-1', got '%s'", known[0].ID)
	}

	// Test IsGameRunning (will be false since game isn't actually running)
	if detector.IsGameRunning("test-game-1") {
		t.Fatalf("IsGameRunning should return false for non-running game")
	}

	// Test IsGameRunning for unknown game
	if detector.IsGameRunning("unknown-game") {
		t.Fatalf("IsGameRunning should return false for unknown game")
	}
}

// TestDetectorRegistration tests game registration
func TestDetectorRegistration(t *testing.T) {
	detector := New()

	game1 := &models.Game{
		ID:          "game-1",
		Name:        "Game 1",
		ProcessName: "game1.exe",
		FirstDetected: time.Now(),
		LastPlayed:  time.Now(),
	}

	game2 := &models.Game{
		ID:          "game-2",
		Name:        "Game 2",
		ProcessName: "game2.exe",
		FirstDetected: time.Now(),
		LastPlayed:  time.Now(),
	}

	detector.RegisterGame(game1)
	detector.RegisterGame(game2)

	known := detector.GetKnownGames()
	if len(known) != 2 {
		t.Fatalf("Expected 2 known games, got %d", len(known))
	}
}

