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
	rulesDir string
}

// NewRuleGenerator creates a new rule generator
func NewRuleGenerator(rulesDir string) *RuleGenerator {
	return &RuleGenerator{
		rulesDir: rulesDir,
	}
}

// GenerateRuleFile generates a basic rule file for a discovered game
func (rg *RuleGenerator) GenerateRuleFile(game *models.Game) error {
	// Check if rule file already exists
	rulePath := rg.getRuleFilePath(game)
	if _, err := os.Stat(rulePath); err == nil {
		// Rule file already exists, don't overwrite
		return nil
	}

	// Create rule structure
	gameRules := rg.createBasicRules(game)

	// Ensure rules directory exists
	if err := os.MkdirAll(rg.rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
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

// getRuleFilePath returns the path for a game's rule file
func (rg *RuleGenerator) getRuleFilePath(game *models.Game) string {
	// Normalize game name for filename
	filename := strings.ToLower(game.Name)
	filename = strings.ReplaceAll(filename, " ", "-")
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	
	// Remove invalid characters
	invalid := []rune{'<', '>', ':', '"', '|', '?', '*'}
	for _, char := range invalid {
		filename = strings.ReplaceAll(filename, string(char), "")
	}

	return filepath.Join(rg.rulesDir, filename+".yaml")
}

// createBasicRules creates a basic rule structure for a game
func (rg *RuleGenerator) createBasicRules(game *models.Game) *GameRules {
	rules := &GameRules{
		GameID:   game.ID,
		GameName: game.Name,
		Detection: DetectionConfig{
			ProcessName: game.ProcessName,
			Executable:  game.ExecutablePath,
		},
		Achievements: []AchievementRule{},
	}

	// Infer common save/log paths
	savePaths := rg.inferSavePaths(game)
	logPaths := rg.inferLogPaths(game)

	// Add placeholder achievements based on inferred paths
	if len(savePaths) > 0 {
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-save-game",
			Name: "Progress Saved",
			Rules: []Rule{
				{
					ID:   "rule-save-file",
					Type: RuleTypeFileWatch,
					Config: map[string]interface{}{
						"path":      savePaths[0],
						"condition": "modified",
					},
				},
			},
		})
	}

	if len(logPaths) > 0 {
		rules.Achievements = append(rules.Achievements, AchievementRule{
			ID:   "ach-log-detected",
			Name: "Game Activity Detected",
			Rules: []Rule{
				{
					ID:   "rule-log-pattern",
					Type: RuleTypeLogPattern,
					Config: map[string]interface{}{
						"pattern": ".*", // Match any log line
					},
				},
			},
		})
	}

	// Add a comment/placeholder achievement
	rules.Achievements = append(rules.Achievements, AchievementRule{
		ID:   "ach-placeholder",
		Name: "Example Achievement (Customize Me)",
		Rules: []Rule{
			{
				ID:   "rule-placeholder",
				Type: RuleTypeFileWatch,
				Config: map[string]interface{}{
					"path":      "saves/achievement.dat",
					"condition": "exists",
					"comment":   "This is a placeholder. Replace with actual achievement conditions.",
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

