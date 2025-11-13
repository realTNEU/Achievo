package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"achievo/internal/models"
)

// TestDiscoverer_DiscoverGames tests game discovery with mocked folder structures
func TestDiscoverer_DiscoverGames(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "achievo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock game directory structure
	gameDir := filepath.Join(tmpDir, "TestGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatalf("Failed to create game directory: %v", err)
	}

	// Create game executable
	exePath := filepath.Join(gameDir, "testgame.exe")
	if err := os.WriteFile(exePath, []byte("fake exe"), 0644); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}

	// Create save directory
	saveDir := filepath.Join(gameDir, "saves")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		t.Fatalf("Failed to create save directory: %v", err)
	}

	// Create metadata file
	metadataFile := filepath.Join(gameDir, "game.ini")
	if err := os.WriteFile(metadataFile, []byte("[Game]\nName=TestGame"), 0644); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	discoverer := New()

	// Test scanDrive directly with our test directory
	games, err := discoverer.scanDrive(tmpDir)
	if err != nil {
		t.Fatalf("scanDrive failed: %v", err)
	}

	// Verify at least one game was discovered
	if len(games) == 0 {
		t.Error("Expected to discover at least one game, but found none")
	}

	// Verify game properties
	found := false
	for _, game := range games {
		if game.Name == "testgame" || game.ProcessName == "testgame.exe" {
			found = true
			if game.ExecutablePath == "" {
				t.Error("ExecutablePath should be set")
			}
			if game.InstallDirectory == "" {
				t.Error("InstallDirectory should be set")
			}
			if game.Confidence <= 0 {
				t.Error("Confidence should be greater than 0")
			}
			break
		}
	}

	if !found {
		t.Error("Expected game not found in discovery results")
	}
}

// TestDiscoverer_IsExcluded tests path exclusion logic
func TestDiscoverer_IsExcluded(t *testing.T) {
	discoverer := New()

	tests := []struct {
		path     string
		excluded bool
	}{
		{"C:\\Windows\\System32", true},
		{"C:\\Program Files\\Game", true},
		{"C:\\Games\\MyGame", false},
		{"/usr/bin", true},
		{"/home/user/Games", false},
	}

	for _, tt := range tests {
		result := discoverer.isExcluded(tt.path)
		if result != tt.excluded {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.path, result, tt.excluded)
		}
	}
}

// TestDiscoveredGame_ToGame tests conversion to Game model
func TestDiscoveredGame_ToGame(t *testing.T) {
	discGame := DiscoveredGame{
		Name:             "Test Game",
		ExecutablePath:   "/path/to/game.exe",
		InstallDirectory: "/path/to",
		ProcessName:      "game.exe",
		Type:             models.GameTypeNonSteam,
		Confidence:       0.9,
	}

	game := discGame.ToGame()

	if game.Name != "Test Game" {
		t.Errorf("Expected Name 'Test Game', got %q", game.Name)
	}
	if game.ExecutablePath != "/path/to/game.exe" {
		t.Errorf("Expected ExecutablePath '/path/to/game.exe', got %q", game.ExecutablePath)
	}
	if game.Type != models.GameTypeNonSteam {
		t.Errorf("Expected Type non_steam, got %q", game.Type)
	}
	if game.ID == "" {
		t.Error("Game ID should be generated")
	}
}

