package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"achievo/internal/models"

	"github.com/shirou/gopsutil/v3/process"
)

// Detector detects running games
type Detector struct {
	knownGames map[string]*models.Game
	processes  map[int]*process.Process
}

// New creates a new Detector
func New() *Detector {
	return &Detector{
		knownGames: make(map[string]*models.Game),
		processes:  make(map[int]*process.Process),
	}
}

// RegisterGame registers a known game
func (d *Detector) RegisterGame(game *models.Game) {
	d.knownGames[game.ID] = game
}

// Detect scans running processes and returns detected games
func (d *Detector) Detect() ([]*models.Game, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	var detected []*models.Game
	detectedMap := make(map[string]bool)

	for _, proc := range processes {
		// Get process name
		name, err := proc.Name()
		if err != nil {
			continue
		}

		// Check against known games
		for _, game := range d.knownGames {
			if strings.EqualFold(name, game.ProcessName) {
				// Additional check: executable path if available
				exe, err := proc.Exe()
				if err == nil && game.ExecutablePath != "" {
					if !strings.EqualFold(exe, game.ExecutablePath) {
						continue
					}
				}

				// Check if already detected
				if !detectedMap[game.ID] {
					detected = append(detected, game)
					detectedMap[game.ID] = true
					d.processes[int(proc.Pid)] = proc
				}
				break
			}
		}
	}

	return detected, nil
}

// IsGameRunning checks if a specific game is currently running
func (d *Detector) IsGameRunning(gameID string) bool {
	game, exists := d.knownGames[gameID]
	if !exists {
		return false
	}

	processes, err := process.Processes()
	if err != nil {
		return false
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		if strings.EqualFold(name, game.ProcessName) {
			// Verify executable path if available
			if game.ExecutablePath != "" {
				exe, err := proc.Exe()
				if err == nil {
					if strings.EqualFold(exe, game.ExecutablePath) {
						return true
					}
				}
			} else {
				return true
			}
		}
	}

	return false
}

// GetProcessID returns the process ID for a running game
func (d *Detector) GetProcessID(gameID string) (int, error) {
	game, exists := d.knownGames[gameID]
	if !exists {
		return 0, fmt.Errorf("game not registered: %s", gameID)
	}

	processes, err := process.Processes()
	if err != nil {
		return 0, err
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}

		if strings.EqualFold(name, game.ProcessName) {
			if game.ExecutablePath != "" {
				exe, err := proc.Exe()
				if err == nil {
					if strings.EqualFold(exe, game.ExecutablePath) {
						return int(proc.Pid), nil
					}
				}
			} else {
				return int(proc.Pid), nil
			}
		}
	}

	return 0, fmt.Errorf("game not running: %s", gameID)
}

// DetectGameFromProcess attempts to detect a game from a process
func (d *Detector) DetectGameFromProcess(proc *process.Process) (*models.Game, error) {
	name, err := proc.Name()
	if err != nil {
		return nil, err
	}

	exe, err := proc.Exe()
	if err != nil {
		return nil, err
	}

	// Check against known games
	for _, game := range d.knownGames {
		if strings.EqualFold(name, game.ProcessName) {
			if game.ExecutablePath != "" {
				if strings.EqualFold(exe, game.ExecutablePath) {
					return game, nil
				}
			} else {
				return game, nil
			}
		}
	}

	// Try to create a new game entry from process
	// This handles unknown games
	gameID := fmt.Sprintf("game-%s-%d", strings.ToLower(name), time.Now().Unix())
	game := &models.Game{
		ID:               gameID,
		Name:             name,
		Type:             models.GameTypeNonSteam,
		ExecutablePath:   exe,
		ProcessName:      name,
		InstallDirectory: filepath.Dir(exe),
		FirstDetected:    time.Now(),
		LastPlayed:       time.Now(),
	}

	return game, nil
}

// GetKnownGames returns all registered games
func (d *Detector) GetKnownGames() []*models.Game {
	games := make([]*models.Game, 0, len(d.knownGames))
	for _, game := range d.knownGames {
		games = append(games, game)
	}
	return games
}

// LoadGamesFromDirectory loads game definitions from a directory
func (d *Detector) LoadGamesFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		// Game loading will be handled by rule engine
		// This is a placeholder for future game definition loading
		_ = filepath.Join(dir, entry.Name())
	}

	return nil
}
