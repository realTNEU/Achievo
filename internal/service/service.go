package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"achievo/internal/config"
	"achievo/internal/detector"
	"achievo/internal/models"
	"achievo/internal/monitor"
	"achievo/internal/rules"
	"achievo/internal/scanner"
	"achievo/internal/steam"
	"achievo/internal/storage"
	"achievo/internal/tracker"
)

// Service is the main orchestrator
type Service struct {
	config      *config.Config
	storage     *storage.Storage
	detector    *detector.Detector
	monitor     *monitor.Monitor
	steam       *steam.SteamClient
	ruleEngine  *rules.RuleEngine
	scanner     *scanner.Scanner
	tracker     *tracker.Tracker
	running     bool
	activeGames map[string]*GameContext
	mu          sync.RWMutex
	stopChan    chan struct{}
}

// GameContext tracks context for an active game
type GameContext struct {
	Game      *models.Game
	Session   *models.Session
	Rules     *rules.GameRules
	Watches   []string
}

// New creates a new Service
func New(cfg *config.Config) (*Service, error) {
	// Initialize storage
	stor, err := storage.New(cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize components
	det := detector.New()
	mon := monitor.New()
	steamClient := steam.New(cfg.Steam.APIKey, stor)
	ruleEng := rules.New()
	scn := scanner.New()
	trk := tracker.New(stor)

	service := &Service{
		config:      cfg,
		storage:     stor,
		detector:    det,
		monitor:     mon,
		steam:       steamClient,
		ruleEngine:  ruleEng,
		scanner:     scn,
		tracker:     trk,
		activeGames: make(map[string]*GameContext),
		stopChan:    make(chan struct{}),
	}

	return service, nil
}

// Start starts the service
func (s *Service) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting Achievo service...")

	// Load game rules
	if err := s.loadGameRules(); err != nil {
		log.Printf("Warning: failed to load some game rules: %v", err)
	}

	// Register known games from storage
	if err := s.loadKnownGames(); err != nil {
		log.Printf("Warning: failed to load known games: %v", err)
	}

	// Start detection loop
	go s.detectionLoop()

	// Start file event processing
	go s.fileEventLoop()

	// Start rule evaluation loop
	go s.ruleEvaluationLoop()

	// Start session monitoring
	go s.sessionMonitoringLoop()

	log.Println("Achievo service started")
	return nil
}

// Stop stops the service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	log.Println("Stopping Achievo service...")

	// Stop all active monitoring
	s.mu.Lock()
	for gameID, ctx := range s.activeGames {
		s.stopGameMonitoring(gameID, ctx)
	}
	s.mu.Unlock()

	// Close scanner
	if err := s.scanner.Close(); err != nil {
		log.Printf("Error closing scanner: %v", err)
	}

	// Close storage
	if err := s.storage.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("Achievo service stopped")
	return nil
}

// loadGameRules loads all game rule files
func (s *Service) loadGameRules() error {
	rulesDir := s.config.Paths.GameRules
	if rulesDir == "" {
		return nil
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, that's okay
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(rulesDir, entry.Name())
		if err := s.ruleEngine.LoadRules(path); err != nil {
			log.Printf("Failed to load rules from %s: %v", path, err)
			continue
		}

		log.Printf("Loaded rules from %s", path)
	}

	return nil
}

// loadKnownGames loads known games from storage
func (s *Service) loadKnownGames() error {
	// This would load games from storage
	// For now, games are registered when detected
	return nil
}

// detectionLoop continuously detects running games
func (s *Service) detectionLoop() {
	ticker := time.NewTicker(time.Duration(s.config.Detection.ScanInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.detectGames()
		}
	}
}

// detectGames detects currently running games
func (s *Service) detectGames() {
	detected, err := s.detector.Detect()
	if err != nil {
		log.Printf("Error detecting games: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for new games
	for _, game := range detected {
		if _, exists := s.activeGames[game.ID]; !exists {
			// New game detected
			s.startGameMonitoring(game)
		}
	}

	// Check for stopped games
	for gameID, ctx := range s.activeGames {
		if !s.detector.IsGameRunning(gameID) {
			s.stopGameMonitoring(gameID, ctx)
		}
	}
}

// startGameMonitoring starts monitoring a game
func (s *Service) startGameMonitoring(game *models.Game) {
	log.Printf("Starting monitoring for game: %s (%s)", game.Name, game.ID)

	// Get or create process ID
	processID, err := s.detector.GetProcessID(game.ID)
	if err != nil {
		log.Printf("Failed to get process ID for game %s: %v", game.ID, err)
		return
	}

	// Start session monitoring
	if err := s.monitor.StartMonitoring(game.ID, processID); err != nil {
		log.Printf("Failed to start monitoring for game %s: %v", game.ID, err)
		return
	}

	// Get session
	session, err := s.monitor.GetSession(game.ID)
	if err != nil {
		log.Printf("Failed to get session for game %s: %v", game.ID, err)
		return
	}

	// Save game to storage
	game.LastPlayed = time.Now()
	if err := s.storage.SaveGame(game); err != nil {
		log.Printf("Failed to save game %s: %v", game.ID, err)
	}

	// Save session
	if err := s.storage.SaveSession(session); err != nil {
		log.Printf("Failed to save session for game %s: %v", game.ID, err)
	}

	// Load game rules
	gameRules, err := s.ruleEngine.GetRules(game.ID)
	if err != nil {
		log.Printf("No rules found for game %s: %v", game.ID, err)
		gameRules = nil
	}

	// Create game context
	ctx := &GameContext{
		Game:    game,
		Session: session,
		Rules:   gameRules,
		Watches: make([]string, 0),
	}

	// Start file watching if rules exist
	if gameRules != nil && game.InstallDirectory != "" {
		s.startFileWatching(ctx)
	}

	// Fetch Steam achievements if Steam game
	if game.Type == models.GameTypeSteam && game.SteamAppID != "" {
		go s.fetchSteamAchievements(game.SteamAppID)
	}

	// Create game start event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-start-%d", game.ID, time.Now().Unix()),
		Type:      models.EventTypeGameStart,
		GameID:    game.ID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"game_name": game.Name,
			"process_id": processID,
		},
		Synced: false,
	}
	s.storage.SaveEvent(&event)

	s.activeGames[game.ID] = ctx
}

// stopGameMonitoring stops monitoring a game
func (s *Service) stopGameMonitoring(gameID string, ctx *GameContext) {
	log.Printf("Stopping monitoring for game: %s", gameID)

	// Stop session monitoring
	session, err := s.monitor.StopMonitoring(gameID)
	if err == nil && session != nil {
		// Update game playtime
		game := ctx.Game
		game.TotalPlaytime += session.Duration
		game.LastPlayed = time.Now()
		s.storage.SaveGame(game)

		// Save session
		s.storage.SaveSession(session)
	}

	// Stop file watching
	for _, watchPath := range ctx.Watches {
		s.scanner.StopWatching(watchPath)
	}

	// Create game stop event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-stop-%d", gameID, time.Now().Unix()),
		Type:      models.EventTypeGameStop,
		GameID:    gameID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"game_name": ctx.Game.Name,
		},
		Synced: false,
	}
	s.storage.SaveEvent(&event)

	delete(s.activeGames, gameID)
}

// startFileWatching starts file watching for a game
func (s *Service) startFileWatching(ctx *GameContext) {
	if ctx.Rules == nil || ctx.Game.InstallDirectory == "" {
		return
	}

	// Watch game directory
	patterns := []string{"*.log", "*.txt", "*.dat", "*.save", "*.sav"}
	if err := s.scanner.WatchDirectory(ctx.Game.InstallDirectory, patterns, true); err != nil {
		log.Printf("Failed to watch directory for game %s: %v", ctx.Game.ID, err)
		return
	}

	ctx.Watches = append(ctx.Watches, ctx.Game.InstallDirectory)
	log.Printf("Started watching directory: %s", ctx.Game.InstallDirectory)
}

// fetchSteamAchievements fetches achievements from Steam API
func (s *Service) fetchSteamAchievements(appID string) {
	log.Printf("Fetching Steam achievements for app ID: %s", appID)
	achievements, err := s.steam.FetchAchievements(appID)
	if err != nil {
		log.Printf("Failed to fetch Steam achievements for %s: %v", appID, err)
		return
	}

	log.Printf("Fetched %d achievements for Steam app %s", len(achievements), appID)
}

// fileEventLoop processes file system events
func (s *Service) fileEventLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		case event, ok := <-s.scanner.GetEvents():
			if !ok {
				return
			}
			s.handleFileEvent(event)
		}
	}
}

// handleFileEvent handles a file system event
func (s *Service) handleFileEvent(event scanner.FileEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find which game this event belongs to
	for gameID, ctx := range s.activeGames {
		if ctx.Game.InstallDirectory != "" {
			rel, err := filepath.Rel(ctx.Game.InstallDirectory, event.Path)
			if err == nil && rel != ".." && !filepath.IsAbs(rel) {
				// This event belongs to this game
				s.evaluateRulesForFile(gameID, ctx, event)
				break
			}
		}
	}
}

// evaluateRulesForFile evaluates rules based on a file event
func (s *Service) evaluateRulesForFile(gameID string, ctx *GameContext, event scanner.FileEvent) {
	if ctx.Rules == nil {
		return
	}

	context := &rules.EvaluationContext{
		GameID:    gameID,
		FilePath:  event.Path,
		Timestamp: event.Timestamp,
	}

	// Read file content if needed
	if event.Type == "write" || event.Type == "create" {
		data, err := os.ReadFile(event.Path)
		if err == nil {
			context.FileContent = string(data)
		}
	}

	// Evaluate rules
	unlocked, err := s.ruleEngine.Evaluate(gameID, context)
	if err != nil {
		log.Printf("Error evaluating rules for game %s: %v", gameID, err)
		return
	}

	// Unlock achievements
	for _, achievementID := range unlocked {
		if err := s.tracker.UnlockAchievement(gameID, achievementID); err != nil {
			log.Printf("Failed to unlock achievement %s for game %s: %v", achievementID, gameID, err)
		} else {
			log.Printf("Unlocked achievement %s for game %s", achievementID, gameID)
		}
	}
}

// ruleEvaluationLoop periodically evaluates rules
func (s *Service) ruleEvaluationLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.evaluateAllRules()
		}
	}
}

// evaluateAllRules evaluates rules for all active games
func (s *Service) evaluateAllRules() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for gameID, ctx := range s.activeGames {
		if ctx.Rules == nil {
			continue
		}

		context := &rules.EvaluationContext{
			GameID:    gameID,
			Timestamp: time.Now(),
		}

		unlocked, err := s.ruleEngine.Evaluate(gameID, context)
		if err != nil {
			log.Printf("Error evaluating rules for game %s: %v", gameID, err)
			continue
		}

		for _, achievementID := range unlocked {
			if err := s.tracker.UnlockAchievement(gameID, achievementID); err != nil {
				log.Printf("Failed to unlock achievement %s for game %s: %v", achievementID, gameID, err)
			}
		}
	}
}

// sessionMonitoringLoop monitors active sessions
func (s *Service) sessionMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.updateSessions()
		}
	}
}

// updateSessions updates all active sessions
func (s *Service) updateSessions() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for gameID := range s.activeGames {
		if err := s.monitor.UpdateSession(gameID); err != nil {
			// Session might have ended
			continue
		}

		session, err := s.monitor.GetSession(gameID)
		if err == nil {
			s.storage.SaveSession(session)
		}
	}

	// Check for stopped processes
	stopped := s.monitor.CheckProcesses()
	for _, gameID := range stopped {
		if ctx, exists := s.activeGames[gameID]; exists {
			s.mu.RUnlock()
			s.mu.Lock()
			s.stopGameMonitoring(gameID, ctx)
			s.mu.Unlock()
			s.mu.RLock()
		}
	}
}

