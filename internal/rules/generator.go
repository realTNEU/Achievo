package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"achievo/internal/models"
	"gopkg.in/yaml.v3"
)

// RuleGenerator generates rule files for discovered games
type RuleGenerator struct {
	rulesDir    string
	autoRulesDir string // Separate directory for auto-generated rules
}

// NewRuleGenerator creates a new rule generator
func NewRuleGenerator(rulesDir string) *RuleGenerator {
	return &RuleGenerator{
		rulesDir:     rulesDir,
		autoRulesDir: filepath.Join(rulesDir, "auto"),
	}
}

// GenerateRuleFile generates a basic rule file for a discovered game in the auto directory
func (rg *RuleGenerator) GenerateRuleFile(game *models.Game) error {
	// Check if manual rule file already exists (don't auto-generate if manual exists)
	manualPath := rg.GetManualRuleFilePath(game)
	if _, err := os.Stat(manualPath); err == nil {
		// Manual rule file exists, don't create auto-generated one
		return nil
	}

	// Check if auto-generated rule file already exists
	rulePath := rg.GetAutoRuleFilePath(game)
	if _, err := os.Stat(rulePath); err == nil {
		// Auto rule file already exists, don't overwrite
		return nil
	}

	// Create rule structure
	gameRules := rg.createBasicRules(game)

	// Ensure auto rules directory exists
	if err := os.MkdirAll(rg.autoRulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create auto rules directory: %w", err)
	}

	// Write YAML file
	data, err := yaml.Marshal(gameRules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := os.WriteFile(rulePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	return nil
}

// GetAutoRuleFilePath returns the path for an auto-generated game rule file
// Uses sanitized filename based on executable name
func (rg *RuleGenerator) GetAutoRuleFilePath(game *models.Game) string {
	filename := rg.GetSanitizedRuleFileName(game)
	return filepath.Join(rg.autoRulesDir, filename)
}

// GetManualRuleFilePath returns the path for a manual game rule file
func (rg *RuleGenerator) GetManualRuleFilePath(game *models.Game) string {
	filename := rg.GetSanitizedRuleFileName(game)
	return filepath.Join(rg.rulesDir, filename)
}

// normalizeFilename normalizes a game name for use in filenames
// Creates sanitized identifier like "game-exename"
func (rg *RuleGenerator) normalizeFilename(name string) string {
	filename := strings.ToLower(name)
	filename = strings.ReplaceAll(filename, " ", "-")
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	
	// Remove invalid characters
	invalid := []rune{'<', '>', ':', '"', '|', '?', '*', '.', ','}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, string(char), "")
	}

	return filename
}

// GetSanitizedRuleFileName creates a sanitized filename from game executable name
// Format: "game-[executablename].yaml"
func (rg *RuleGenerator) GetSanitizedRuleFileName(game *models.Game) string {
	// Use process name (executable name) for better identification
	exeName := game.ProcessName
	if exeName == "" {
		// Fallback to game name
		exeName = game.Name
	}
	
	// Remove extension
	exeName = strings.TrimSuffix(exeName, ".exe")
	exeName = strings.TrimSuffix(exeName, ".EXE")
	
	// Normalize
	normalized := rg.normalizeFilename(exeName)
	
	// Ensure it starts with "game-"
	if !strings.HasPrefix(normalized, "game-") {
		normalized = "game-" + normalized
	}
	
	return normalized + ".yaml"
}

// createBasicRules creates a basic rule structure for a game with comprehensive placeholders
func (rg *RuleGenerator) createBasicRules(game *models.Game) *GameRules {
	rules := &GameRules{
		GameID:   game.ID,
		GameName: game.Name,
		Detection: DetectionConfig{
			ProcessName: game.ProcessName,
			Executable:  game.ExecutablePath, // Full executable path placeholder
		},
		Achievements: []AchievementRule{},
	}

	// Add Steam App ID if available
	if game.SteamAppID != "" {
		// Note: We can't add this to DetectionConfig directly, but it's in the game model
	}

	// Infer common save/log paths
	savePaths := rg.inferSavePaths(game)
	logPaths := rg.inferLogPaths(game)

	// Add placeholder achievement for save file detection
	if len(savePaths) > 0 {
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-save-game",
			Name: "Progress Saved (Auto-detected)",
			Rules: []Rule{
				{
					ID:   "rule-save-file",
					Type: RuleTypeFileWatch,
					Config: map[string]interface{}{
						"path":      savePaths[0],
						"condition": "modified",
						"comment":   "Auto-detected save file path. Adjust if incorrect.",
					},
				},
			},
		})
	} else {
		// Add placeholder with default save directory guess
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-save-game",
			Name: "Progress Saved (Placeholder)",
			Rules: []Rule{
				{
					ID:   "rule-save-file",
					Type: RuleTypeFileWatch,
					Config: map[string]interface{}{
						"path":      "saves/*.sav",
						"condition": "modified",
						"comment":   "Default save path placeholder. Update with actual save file location.",
					},
				},
			},
		})
	}

	// Add placeholder achievement for log file watching
	if len(logPaths) > 0 {
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-log-detected",
			Name: "Game Activity Detected (Auto-detected)",
			Rules: []Rule{
				{
					ID:   "rule-log-pattern",
					Type: RuleTypeLogPattern,
					Config: map[string]interface{}{
						"pattern": ".*",
						"file":    logPaths[0],
						"comment": "Auto-detected log file. Customize pattern for specific achievements.",
					},
				},
			},
		})
	} else {
		// Add placeholder with default log file guess
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-log-detected",
			Name: "Game Activity Detected (Placeholder)",
			Rules: []Rule{
				{
					ID:   "rule-log-pattern",
					Type: RuleTypeLogPattern,
					Config: map[string]interface{}{
						"pattern": ".*",
						"file":    "logs/game.log",
						"comment": "Default log file placeholder. Update with actual log file path and pattern.",
					},
				},
			},
		})
	}

	// Add placeholder achievement with memory signature section (disabled by default)
	rules.Achievements = append(rules.Achievements, AchievementRule{
		ID:   "ach-memory-example",
		Name: "Memory Signature Example (Placeholder - Disabled)",
		Rules: []Rule{
			{
				ID:   "rule-memory-sig",
				Type: RuleTypeMemorySig,
				Config: map[string]interface{}{
					"pattern": "48 89 5C 24 ?? ?? 00",
					"offset":  0,
					"comment": "Memory signature scanning is disabled by default. Enable in config and provide actual memory pattern.",
					"enabled": false,
				},
			},
		},
	})

	// Add a template achievement for custom rules
	rules.Achievements = append(rules.Achievements, AchievementRule{
		ID:   "ach-custom-template",
		Name: "Custom Achievement Template",
		Rules: []Rule{
			{
				ID:   "rule-template",
				Type: RuleTypeFileWatch,
				Config: map[string]interface{}{
					"path":      "path/to/achievement/indicator",
					"condition": "exists",
					"comment":   "Template rule. Replace with actual achievement detection logic.",
				},
			},
		},
	})

	return rules
}

// inferSavePaths infers likely save file paths
func (rg *RuleGenerator) inferSavePaths(game *models.Game) []string {
	var paths []string
	baseDir := game.InstallDirectory

	// Common save directory patterns
	commonDirs := []string{"saves", "save", "Save", "SAVE", "SaveGames", "SaveData"}
	commonFiles := []string{"*.sav", "*.save", "*.dat", "*.savegame"}

	// Check for save directories
	for _, dir := range commonDirs {
		saveDir := filepath.Join(baseDir, dir)
		if _, err := os.Stat(saveDir); err == nil {
			// Found save directory, add patterns
			for _, pattern := range commonFiles {
				paths = append(paths, filepath.Join(dir, pattern))
			}
		}
	}

	// Check AppData (Windows) or ~/.local/share (Linux)
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			// Try game name in AppData
			gameSavePath := filepath.Join(appData, game.Name, "*.sav")
			if _, err := os.Stat(filepath.Dir(gameSavePath)); err == nil {
				paths = append(paths, gameSavePath)
			}
		}
	}

	return paths
}

// inferLogPaths infers likely log file paths
func (rg *RuleGenerator) inferLogPaths(game *models.Game) []string {
	var paths []string
	baseDir := game.InstallDirectory

	// Common log directory patterns
	commonDirs := []string{"logs", "log", "Logs", "LOG"}
	commonFiles := []string{"*.log", "*.txt"}

	for _, dir := range commonDirs {
		logDir := filepath.Join(baseDir, dir)
		if _, err := os.Stat(logDir); err == nil {
			for _, pattern := range commonFiles {
				paths = append(paths, filepath.Join(dir, pattern))
			}
		}
	}

	// Check for log files in root directory
	for _, pattern := range commonFiles {
		logPath := filepath.Join(baseDir, pattern)
		if matches, _ := filepath.Glob(logPath); len(matches) > 0 {
			paths = append(paths, filepath.Base(pattern))
		}
	}

	return paths
}

