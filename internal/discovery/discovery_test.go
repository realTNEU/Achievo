package discovery

import (
	"os"
	"path/filepath"
	"strings"
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

	// Create mock game directory structure with a proper game name
	gameDir := filepath.Join(tmpDir, "MyAwesomeGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatalf("Failed to create game directory: %v", err)
	}

	// Create game executable with a proper game name (not generic)
	exePath := filepath.Join(gameDir, "myawesomegame.exe")
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
	if err := os.WriteFile(metadataFile, []byte("[Game]\nName=MyAwesomeGame"), 0644); err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}

	// Create a system utility directory that should be filtered out
	systemDir := filepath.Join(tmpDir, "SystemTool")
	if err := os.MkdirAll(systemDir, 0755); err != nil {
		t.Fatalf("Failed to create system directory: %v", err)
	}
	systemExe := filepath.Join(systemDir, "gui-32.exe")
	if err := os.WriteFile(systemExe, []byte("system tool"), 0644); err != nil {
		t.Fatalf("Failed to create system executable: %v", err)
	}

	discoverer := New()

	// Test isLikelyGameDirectory first
	isGameDir := discoverer.isLikelyGameDirectory(gameDir)
	if !isGameDir {
		t.Logf("Directory %s was not identified as a game directory - checking why", gameDir)
		// Check what's in the directory
		entries, _ := os.ReadDir(gameDir)
		hasExe := false
		hasGameFiles := false
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".exe" {
				hasExe = true
				t.Logf("  Found exe: %s", e.Name())
			}
			if e.Name() == "game.ini" {
				hasGameFiles = true
				t.Logf("  Found game file: %s", e.Name())
			}
		}
		t.Logf("  hasExe: %v, hasGameFiles: %v", hasExe, hasGameFiles)
	} else {
		t.Logf("Directory %s was correctly identified as a game directory", gameDir)
	}

	// Test analyzeGameDirectory directly to see what happens
	game, err := discoverer.analyzeGameDirectory(gameDir)
	if err != nil {
		t.Logf("analyzeGameDirectory returned error: %v", err)
	} else if game == nil {
		t.Logf("analyzeGameDirectory returned nil game")
	} else {
		t.Logf("analyzeGameDirectory returned game: %s (exe: %s, path: %s)",
			game.Name, game.ProcessName, game.ExecutablePath)
		// Check if it would be filtered
		if discoverer.isSystemPath(game.ExecutablePath) {
			t.Logf("  Would be filtered by isSystemPath")
		}
		if discoverer.isSystemExe(game.ProcessName) {
			t.Logf("  Would be filtered by isSystemExe")
		}
	}

	// Test scanDrive directly with our test directory
	games, err := discoverer.scanDrive(tmpDir)
	if err != nil {
		t.Fatalf("scanDrive failed: %v", err)
	}

	// Log what was found for debugging
	if len(games) == 0 {
		t.Logf("No games found. Directory structure: %s", gameDir)
		// Check if directory exists and has expected files
		entries, _ := os.ReadDir(gameDir)
		for _, e := range entries {
			t.Logf("  Found: %s (dir: %v)", e.Name(), e.IsDir())
		}
	}

	// Verify game properties
	found := false
	for _, game := range games {
		// Should not find system utilities
		if strings.Contains(game.ProcessName, "gui-32") {
			t.Errorf("System utility should have been filtered out: %s", game.ProcessName)
		}
		// Check for our test game (name might be normalized)
		if strings.Contains(strings.ToLower(game.Name), "myawesomegame") ||
			strings.Contains(strings.ToLower(game.ProcessName), "myawesomegame") {
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
		t.Errorf("Expected game not found in discovery results. Found games: %v",
			func() []string {
				var names []string
				for _, g := range games {
					names = append(names, g.Name+" ("+g.ProcessName+")")
				}
				return names
			}())
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
