package monitor

import (
	"fmt"
	"sync"
	"time"

	"achievo/internal/models"

	"github.com/shirou/gopsutil/v3/process"
)

// Monitor tracks game sessions
type Monitor struct {
	sessions map[string]*SessionTracker
	mu       sync.RWMutex
}

// SessionTracker tracks a single game session
type SessionTracker struct {
	Session   *models.Session
	Process   *process.Process
	StartTime time.Time
	LastCheck time.Time
}

// New creates a new Monitor
func New() *Monitor {
	return &Monitor{
		sessions: make(map[string]*SessionTracker),
	}
}

// StartMonitoring starts monitoring a game session
func (m *Monitor) StartMonitoring(gameID string, processID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already monitoring
	if _, exists := m.sessions[gameID]; exists {
		return fmt.Errorf("already monitoring game: %s", gameID)
	}

	proc, err := process.NewProcess(int32(processID))
	if err != nil {
		return fmt.Errorf("failed to get process: %w", err)
	}

	session := &models.Session{
		ID:        fmt.Sprintf("session-%s-%d", gameID, time.Now().Unix()),
		GameID:    gameID,
		ProcessID: processID,
		StartTime: time.Now(),
		IsActive:  true,
	}

	tracker := &SessionTracker{
		Session:   session,
		Process:   proc,
		StartTime: time.Now(),
		LastCheck: time.Now(),
	}

	m.sessions[gameID] = tracker
	return nil
}

// StopMonitoring stops monitoring a game session
func (m *Monitor) StopMonitoring(gameID string) (*models.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tracker, exists := m.sessions[gameID]
	if !exists {
		return nil, fmt.Errorf("not monitoring game: %s", gameID)
	}

	endTime := time.Now()
	tracker.Session.EndTime = endTime
	tracker.Session.Duration = int64(endTime.Sub(tracker.StartTime).Seconds())
	tracker.Session.IsActive = false

	delete(m.sessions, gameID)
	return tracker.Session, nil
}

// GetSession retrieves the current session for a game
func (m *Monitor) GetSession(gameID string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tracker, exists := m.sessions[gameID]
	if !exists {
		return nil, fmt.Errorf("no active session for game: %s", gameID)
	}

	// Update duration
	now := time.Now()
	tracker.Session.Duration = int64(now.Sub(tracker.StartTime).Seconds())
	tracker.LastCheck = now

	return tracker.Session, nil
}

// CheckProcesses verifies that monitored processes are still running
func (m *Monitor) CheckProcesses() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var stopped []string

	for gameID, tracker := range m.sessions {
		// Check if process is still running
		running, err := tracker.Process.IsRunning()
		if err != nil || !running {
			stopped = append(stopped, gameID)
		}
	}

	return stopped
}

// UpdateSession updates session duration
func (m *Monitor) UpdateSession(gameID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tracker, exists := m.sessions[gameID]
	if !exists {
		return fmt.Errorf("no active session for game: %s", gameID)
	}

	now := time.Now()
	tracker.Session.Duration = int64(now.Sub(tracker.StartTime).Seconds())
	tracker.LastCheck = now

	return nil
}

// GetAllSessions returns all active sessions
func (m *Monitor) GetAllSessions() []*models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*models.Session, 0, len(m.sessions))
	for _, tracker := range m.sessions {
		// Update duration before returning
		now := time.Now()
		tracker.Session.Duration = int64(now.Sub(tracker.StartTime).Seconds())
		sessions = append(sessions, tracker.Session)
	}

	return sessions
}
