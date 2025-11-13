package rules

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"achievo/internal/models"
)

// TestRuleGenerator_GenerateRuleFile tests rule file generation and schema compliance
func TestRuleGenerator_GenerateRuleFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "achievo-rules-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	generator := NewRuleGenerator(tmpDir)

	game := &models.Game{
		ID:               "test-game-123",
		Name:             "Test Game",
		Type:             models.GameTypeNonSteam,
		ExecutablePath:   "/path/to/game.exe",
		ProcessName:      "game.exe",
		InstallDirectory: "/path/to",
		FirstDetected:    time.Now(),
		LastPlayed:       time.Now(),
	}

	// Generate rule file
	if err := generator.GenerateRuleFile(game); err != nil {
		t.Fatalf("GenerateRuleFile failed: %v", err)
	}

	// Verify file was created
	autoPath := generator.GetAutoRuleFilePath(game)
	if _, err := os.Stat(autoPath); os.IsNotExist(err) {
		t.Fatalf("Rule file was not created at %s", autoPath)
	}

	// Verify filename format
	expectedName := "game-game.yaml"
	if filepath.Base(autoPath) != expectedName {
		t.Errorf("Expected filename %q, got %q", expectedName, filepath.Base(autoPath))
	}

	// Validate the generated file
	validator := NewValidator("")
	if err := validator.ValidateRuleFile(autoPath); err != nil {
		t.Errorf("Generated rule file failed validation: %v", err)
	}
}

// TestRuleGenerator_GetSanitizedRuleFileName tests filename sanitization
func TestRuleGenerator_GetSanitizedRuleFileName(t *testing.T) {
	generator := NewRuleGenerator("")

	tests := []struct {
		game     *models.Game
		expected string
	}{
		{
			&models.Game{ProcessName: "mygame.exe", Name: "My Game"},
			"game-mygame.yaml",
		},
		{
			&models.Game{ProcessName: "Test Game.exe", Name: "Test Game"},
			"game-test-game.yaml",
		},
		{
			&models.Game{ProcessName: "", Name: "Complex/Game: Name"},
			"game-complex-game-name.yaml",
		},
	}

	for _, tt := range tests {
		result := generator.GetSanitizedRuleFileName(tt.game)
		if result != tt.expected {
			t.Errorf("GetSanitizedRuleFileName() = %q, want %q", result, tt.expected)
		}
	}
}

// TestRuleGenerator_ManualRulePrecedence tests that manual rules prevent auto-generation
func TestRuleGenerator_ManualRulePrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "achievo-rules-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	generator := NewRuleGenerator(tmpDir)

	game := &models.Game{
		ID:          "test-game-123",
		Name:        "Test Game",
		ProcessName: "game.exe",
	}

	// Create manual rule file
	manualPath := generator.GetManualRuleFilePath(game)
	if err := os.MkdirAll(filepath.Dir(manualPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(manualPath, []byte("manual rule"), 0644); err != nil {
		t.Fatalf("Failed to create manual rule: %v", err)
	}

	// Try to generate - should skip because manual exists
	if err := generator.GenerateRuleFile(game); err != nil {
		t.Fatalf("GenerateRuleFile should not error when manual exists: %v", err)
	}

	// Verify auto file was not created
	autoPath := generator.GetAutoRuleFilePath(game)
	if _, err := os.Stat(autoPath); err == nil {
		t.Error("Auto rule file should not be created when manual rule exists")
	}
}

