package monitor

import (
	"fmt"
	"testing"
	"time"
)

// TestMonitor tests session monitoring with mocked process lifecycle
func TestMonitor(t *testing.T) {
	monitor := New()

	gameID := "test-game-1"
	processID := 12345

	// Test StartMonitoring
	if err := monitor.StartMonitoring(gameID, processID); err != nil {
		t.Fatalf("StartMonitoring failed: %v", err)
	}

	// Test GetSession
	session, err := monitor.GetSession(gameID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if session.GameID != gameID {
		t.Fatalf("Expected GameID %s, got %s", gameID, session.GameID)
	}

	if session.ProcessID != processID {
		t.Fatalf("Expected ProcessID %d, got %d", processID, session.ProcessID)
	}

	if !session.IsActive {
		t.Fatalf("Session should be active")
	}

	// Wait a bit to test duration calculation
	time.Sleep(100 * time.Millisecond)

	// Test UpdateSession
	if err := monitor.UpdateSession(gameID); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	updatedSession, err := monitor.GetSession(gameID)
	if err != nil {
		t.Fatalf("GetSession after update failed: %v", err)
	}

	if updatedSession.Duration <= 0 {
		t.Fatalf("Session duration should be greater than 0")
	}

	// Test StopMonitoring
	stoppedSession, err := monitor.StopMonitoring(gameID)
	if err != nil {
		t.Fatalf("StopMonitoring failed: %v", err)
	}

	if stoppedSession.IsActive {
		t.Fatalf("Stopped session should not be active")
	}

	if stoppedSession.Duration <= 0 {
		t.Fatalf("Stopped session should have duration")
	}

	// Test that session is no longer retrievable
	_, err = monitor.GetSession(gameID)
	if err == nil {
		t.Fatalf("GetSession should fail for stopped session")
	}
}

// TestMonitorMultipleSessions tests multiple concurrent sessions
func TestMonitorMultipleSessions(t *testing.T) {
	monitor := New()

	// Start multiple sessions
	for i := 1; i <= 3; i++ {
		gameID := fmt.Sprintf("game-%d", i)
		processID := 10000 + i

		if err := monitor.StartMonitoring(gameID, processID); err != nil {
			t.Fatalf("StartMonitoring failed for %s: %v", gameID, err)
		}
	}

	// Check all sessions exist
	allSessions := monitor.GetAllSessions()
	if len(allSessions) != 3 {
		t.Fatalf("Expected 3 active sessions, got %d", len(allSessions))
	}

	// Stop one session
	if _, err := monitor.StopMonitoring("game-1"); err != nil {
		t.Fatalf("StopMonitoring failed: %v", err)
	}

	// Verify only 2 sessions remain
	allSessions = monitor.GetAllSessions()
	if len(allSessions) != 2 {
		t.Fatalf("Expected 2 active sessions after stopping one, got %d", len(allSessions))
	}
}
