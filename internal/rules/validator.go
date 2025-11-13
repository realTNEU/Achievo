package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Validator validates rule files against schema
type Validator struct {
	schemaPath string
}

// NewValidator creates a new validator
func NewValidator(schemaPath string) *Validator {
	return &Validator{
		schemaPath: schemaPath,
	}
}

// ValidateRuleFile validates a rule file against the JSON schema
func (v *Validator) ValidateRuleFile(filePath string) error {
	// Read rule file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read rule file: %w", err)
	}

	// Parse YAML/JSON
	var gameRules GameRules
	ext := filepath.Ext(filePath)
	
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &gameRules); err != nil {
			return fmt.Errorf("invalid YAML: %w", err)
		}
	} else if ext == ".json" {
		if err := json.Unmarshal(data, &gameRules); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	// Basic validation
	if err := v.validateStructure(&gameRules); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// validateStructure performs basic structural validation
func (v *Validator) validateStructure(rules *GameRules) error {
	if rules.GameID == "" {
		return fmt.Errorf("game_id is required")
	}
	if rules.GameName == "" {
		return fmt.Errorf("game_name is required")
	}
	if rules.Detection.ProcessName == "" {
		return fmt.Errorf("detection.process_name is required")
	}

	// Validate achievements
	for i, ach := range rules.Achievements {
		if ach.ID == "" {
			return fmt.Errorf("achievements[%d].id is required", i)
		}
		if ach.Name == "" {
			return fmt.Errorf("achievements[%d].name is required", i)
		}
		if len(ach.Rules) == 0 {
			return fmt.Errorf("achievements[%d].rules cannot be empty", i)
		}

		// Validate rules
		for j, rule := range ach.Rules {
			if rule.ID == "" {
				return fmt.Errorf("achievements[%d].rules[%d].id is required", i, j)
			}
			if rule.Type == "" {
				return fmt.Errorf("achievements[%d].rules[%d].type is required", i, j)
			}
			if rule.Config == nil {
				return fmt.Errorf("achievements[%d].rules[%d].config is required", i, j)
			}
		}
	}

	return nil
}

