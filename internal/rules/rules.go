package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// RuleType represents the type of rule
type RuleType string

const (
	RuleTypeFileWatch  RuleType = "file_watch"
	RuleTypeLogPattern RuleType = "log_pattern"
	RuleTypeCounter    RuleType = "counter"
	RuleTypeMemorySig  RuleType = "memory_signature"
)

// Rule represents a single achievement rule
type Rule struct {
	ID       string                 `yaml:"id" json:"id"`
	Type     RuleType               `yaml:"type" json:"type"`
	Config   map[string]interface{} `yaml:"config" json:"config"`
	compiled *CompiledRule
}

// CompiledRule contains compiled rule data
type CompiledRule struct {
	Regex     *regexp.Regexp
	Pattern   string
	Threshold float64
	Counter   *Counter
}

// Counter tracks incremental values
type Counter struct {
	Value     float64
	Threshold float64
	mu        sync.RWMutex
}

// GameRules represents rules for a game
type GameRules struct {
	GameID       string            `yaml:"game_id" json:"game_id"`
	GameName     string            `yaml:"game_name" json:"game_name"`
	Detection    DetectionConfig   `yaml:"detection" json:"detection"`
	Achievements []AchievementRule `yaml:"achievements" json:"achievements"`
}

// DetectionConfig defines how to detect the game
type DetectionConfig struct {
	ProcessName string `yaml:"process_name" json:"process_name"`
	WindowTitle string `yaml:"window_title,omitempty" json:"window_title,omitempty"`
	Executable  string `yaml:"executable,omitempty" json:"executable,omitempty"`
}

// AchievementRule defines rules for unlocking an achievement
type AchievementRule struct {
	ID    string `yaml:"id" json:"id"`
	Name  string `yaml:"name" json:"name"`
	Rules []Rule `yaml:"rules" json:"rules"`
}

// EvaluationContext provides context for rule evaluation
type EvaluationContext struct {
	GameID      string
	FilePath    string
	FileContent string
	LogLine     string
	ProcessID   int
	Timestamp   time.Time
	Counters    map[string]*Counter
	mu          sync.RWMutex
}

// RuleEngine evaluates game-specific rules
type RuleEngine struct {
	rules    map[string]*GameRules
	counters map[string]map[string]*Counter // gameID -> counterID -> Counter
	mu       sync.RWMutex
}

// New creates a new RuleEngine
func New() *RuleEngine {
	return &RuleEngine{
		rules:    make(map[string]*GameRules),
		counters: make(map[string]map[string]*Counter),
	}
}

// LoadRules loads rules from a file with validation
// Returns error if file cannot be loaded, but validation errors are logged as warnings
func (re *RuleEngine) LoadRules(filePath string) error {
	// Validate rule file first (warn on validation errors but continue)
	validator := NewValidator("")
	if err := validator.ValidateRuleFile(filePath); err != nil {
		// Log warning but don't fail - allow loading with validation issues
		// The caller can decide whether to skip or continue
		return fmt.Errorf("rule file validation failed: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read rule file: %w", err)
	}

	var gameRules GameRules

	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &gameRules); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &gameRules); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	// Compile rules
	for i := range gameRules.Achievements {
		for j := range gameRules.Achievements[i].Rules {
			if err := re.compileRule(&gameRules.Achievements[i].Rules[j]); err != nil {
				return fmt.Errorf("failed to compile rule %s: %w", gameRules.Achievements[i].Rules[j].ID, err)
			}
		}
	}

	re.mu.Lock()
	re.rules[gameRules.GameID] = &gameRules
	re.mu.Unlock()

	// Initialize counters for this game
	re.mu.Lock()
	re.counters[gameRules.GameID] = make(map[string]*Counter)
	re.mu.Unlock()

	return nil
}

// compileRule compiles a rule for faster evaluation
func (re *RuleEngine) compileRule(rule *Rule) error {
	compiled := &CompiledRule{}

	switch rule.Type {
	case RuleTypeLogPattern:
		pattern, ok := rule.Config["pattern"].(string)
		if !ok {
			return fmt.Errorf("log_pattern rule requires 'pattern' config")
		}
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		compiled.Regex = regex
		compiled.Pattern = pattern

	case RuleTypeCounter:
		threshold, ok := rule.Config["threshold"].(float64)
		if !ok {
			// Try int
			if thresholdInt, ok := rule.Config["threshold"].(int); ok {
				threshold = float64(thresholdInt)
			} else {
				return fmt.Errorf("counter rule requires 'threshold' config")
			}
		}
		compiled.Threshold = threshold
		compiled.Counter = &Counter{
			Threshold: threshold,
		}

	case RuleTypeFileWatch:
		// File watch rules don't need compilation
		compiled = nil

	case RuleTypeMemorySig:
		// Memory signature rules need pattern compilation
		pattern, ok := rule.Config["pattern"].(string)
		if ok {
			// Pattern is a hex string like "48 89 5C 24 ??"
			compiled.Pattern = pattern
		}
	}

	rule.compiled = compiled
	return nil
}

// Evaluate evaluates rules for a game given a context
func (re *RuleEngine) Evaluate(gameID string, context *EvaluationContext) ([]string, error) {
	re.mu.RLock()
	gameRules, exists := re.rules[gameID]
	re.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no rules found for game: %s", gameID)
	}

	// Initialize counters if needed
	re.mu.Lock()
	if context.Counters == nil {
		context.Counters = make(map[string]*Counter)
	}
	if gameCounters, exists := re.counters[gameID]; exists {
		for id, counter := range gameCounters {
			if _, exists := context.Counters[id]; !exists {
				context.Counters[id] = counter
			}
		}
	}
	re.mu.Unlock()

	var unlocked []string

	for _, achRule := range gameRules.Achievements {
		// All rules for an achievement must pass (AND logic)
		allPassed := true

		for _, rule := range achRule.Rules {
			passed, err := re.evaluateRule(&rule, context)
			if err != nil {
				return nil, fmt.Errorf("error evaluating rule %s: %w", rule.ID, err)
			}

			if !passed {
				allPassed = false
				break
			}
		}

		if allPassed {
			unlocked = append(unlocked, achRule.ID)
		}
	}

	return unlocked, nil
}

// evaluateRule evaluates a single rule
func (re *RuleEngine) evaluateRule(rule *Rule, context *EvaluationContext) (bool, error) {
	switch rule.Type {
	case RuleTypeFileWatch:
		return re.evaluateFileWatch(rule, context)
	case RuleTypeLogPattern:
		return re.evaluateLogPattern(rule, context)
	case RuleTypeCounter:
		return re.evaluateCounter(rule, context)
	case RuleTypeMemorySig:
		return re.evaluateMemorySig(rule, context)
	default:
		return false, fmt.Errorf("unknown rule type: %s", rule.Type)
	}
}

// evaluateFileWatch evaluates a file watch rule
func (re *RuleEngine) evaluateFileWatch(rule *Rule, _ *EvaluationContext) (bool, error) {
	path, ok := rule.Config["path"].(string)
	if !ok {
		return false, fmt.Errorf("file_watch rule requires 'path' config")
	}

	condition, _ := rule.Config["condition"].(string)
	if condition == "" {
		condition = "exists"
	}

	switch condition {
	case "exists":
		_, err := os.Stat(path)
		return err == nil, nil
	case "modified":
		// Check if file was modified recently
		info, err := os.Stat(path)
		if err != nil {
			return false, nil
		}
		// Check if modified within last minute
		return time.Since(info.ModTime()) < time.Minute, nil
	case "contains":
		content, ok := rule.Config["content"].(string)
		if !ok {
			return false, fmt.Errorf("file_watch 'contains' condition requires 'content' config")
		}
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return false, nil
		}
		return regexp.MustCompile(content).Match(fileContent), nil
	default:
		return false, fmt.Errorf("unknown file_watch condition: %s", condition)
	}
}

// evaluateLogPattern evaluates a log pattern rule
func (re *RuleEngine) evaluateLogPattern(rule *Rule, context *EvaluationContext) (bool, error) {
	if rule.compiled == nil || rule.compiled.Regex == nil {
		return false, fmt.Errorf("log_pattern rule not compiled")
	}

	if context.LogLine == "" {
		return false, nil
	}

	matched := rule.compiled.Regex.MatchString(context.LogLine)

	// If this is a counter source, increment counter
	if counterID, ok := rule.Config["counter_id"].(string); ok && matched {
		re.mu.Lock()
		if context.Counters == nil {
			context.Counters = make(map[string]*Counter)
		}
		if counter, exists := context.Counters[counterID]; exists {
			counter.mu.Lock()
			counter.Value++
			counter.mu.Unlock()
		} else {
			counter = &Counter{}
			context.Counters[counterID] = counter
			counter.mu.Lock()
			counter.Value = 1
			counter.mu.Unlock()
		}
		re.mu.Unlock()
	}

	return matched, nil
}

// evaluateCounter evaluates a counter rule
func (re *RuleEngine) evaluateCounter(rule *Rule, context *EvaluationContext) (bool, error) {
	if rule.compiled == nil || rule.compiled.Counter == nil {
		return false, fmt.Errorf("counter rule not compiled")
	}

	counterID, ok := rule.Config["counter_id"].(string)
	if !ok {
		return false, fmt.Errorf("counter rule requires 'counter_id' config")
	}

	re.mu.RLock()
	counter, exists := context.Counters[counterID]
	re.mu.RUnlock()

	if !exists {
		return false, nil
	}

	counter.mu.RLock()
	value := counter.Value
	counter.mu.RUnlock()

	return value >= rule.compiled.Threshold, nil
}

// evaluateMemorySig evaluates a memory signature rule
func (re *RuleEngine) evaluateMemorySig(_ *Rule, _ *EvaluationContext) (bool, error) {
	// Memory signature scanning is a placeholder
	// Full implementation would require process memory reading
	// This is a read-only, non-invasive operation
	// For v1, we'll return false as it requires platform-specific implementation
	return false, fmt.Errorf("memory signature scanning not yet implemented")
}

// GetRules returns rules for a game
func (re *RuleEngine) GetRules(gameID string) (*GameRules, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules, exists := re.rules[gameID]
	if !exists {
		return nil, fmt.Errorf("no rules found for game: %s", gameID)
	}

	return rules, nil
}

// GetCounter returns a counter for a game
func (re *RuleEngine) GetCounter(gameID, counterID string) (*Counter, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	gameCounters, exists := re.counters[gameID]
	if !exists {
		return nil, fmt.Errorf("no counters found for game: %s", gameID)
	}

	counter, exists := gameCounters[counterID]
	if !exists {
		return nil, fmt.Errorf("counter not found: %s", counterID)
	}

	return counter, nil
}
